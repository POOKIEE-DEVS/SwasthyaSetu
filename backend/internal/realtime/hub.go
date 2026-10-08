// Package realtime runs the WebSockets: the live queue verified
// professionals watch, and the signalling that sets up each WebRTC call. It
// also hands out the ICE (STUN/TURN) servers the browsers need.
//
// Every socket has its own writer goroutine and outbound queue, so a slow
// browser never holds up the others, and messages to one browser always
// arrive in the order they were sent. The hub's lock covers room changes and
// sends together, so everyone sees changes in the same order.
//
// All of this lives in one process, which is why the backend must run as a
// single instance.
package realtime

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// Close codes the frontend tells apart. 4000-4999 are application-defined.
const (
	CloseReplaced     websocket.StatusCode = 4000 // the same participant reconnected
	CloseNotVerified  websocket.StatusCode = 4401 // not a verified professional
	CloseUnauthorized websocket.StatusCode = 4403 // unknown call, wrong token, or ended
)

const (
	sendQueue    = 64
	writeTimeout = 10 * time.Second
	// Like uvicorn's defaults: a ping every 20 seconds keeps proxies from
	// closing quiet sockets, and a missing pong within 20 seconds means the
	// browser is gone.
	pingInterval = 20 * time.Second
	pingTimeout  = 20 * time.Second
)

// Client is one open socket.
type Client struct {
	conn   *websocket.Conn
	send   chan []byte
	done   chan struct{}
	once   sync.Once
	log    *slog.Logger
	closed sync.WaitGroup
}

// NewClient wraps an accepted connection and starts its writer.
func NewClient(conn *websocket.Conn, log *slog.Logger) *Client {
	c := &Client{conn: conn, send: make(chan []byte, sendQueue), done: make(chan struct{}), log: log}
	c.closed.Add(1)
	go c.writeLoop()
	return c
}

func (c *Client) writeLoop() {
	defer c.closed.Done()
	ping := time.NewTicker(pingInterval)
	defer ping.Stop()
	for {
		select {
		case <-c.done:
			return
		case msg := <-c.send:
			ctx, cancel := context.WithTimeout(context.Background(), writeTimeout)
			err := c.conn.Write(ctx, websocket.MessageText, msg)
			cancel()
			if err != nil {
				c.Close(websocket.StatusGoingAway, "")
				return
			}
		case <-ping.C:
			ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
			err := c.conn.Ping(ctx)
			cancel()
			if err != nil {
				c.Close(websocket.StatusGoingAway, "")
				return
			}
		}
	}
}

// Send queues a JSON message. It never blocks: a browser too slow to keep
// up is disconnected, and reconnects to a fresh state.
func (c *Client) Send(msg any) {
	data, err := json.Marshal(msg)
	if err != nil {
		c.log.Error("websocket message not sent", "error", err)
		return
	}
	select {
	case <-c.done:
	case c.send <- data:
	default:
		c.log.Warn("websocket too slow; disconnecting it")
		c.Close(websocket.StatusTryAgainLater, "too slow")
	}
}

// Close stops the writer and closes the socket with a code. It returns at
// once; the close handshake finishes in the background.
func (c *Client) Close(code websocket.StatusCode, reason string) {
	c.once.Do(func() {
		close(c.done)
		go func() { _ = c.conn.Close(code, reason) }()
	})
}

// Wait blocks until the writer has stopped.
func (c *Client) Wait() { c.closed.Wait() }

// Hub tracks who is connected where.
type Hub struct {
	mu    sync.Mutex
	rooms map[string]map[*Client]struct{}
	// The socket holding each (call, role) slot. A reconnect for the same
	// role (a page refresh) replaces the old socket instead of being refused.
	slots map[slot]*Client
	log   *slog.Logger
}

type slot struct{ call, role string }

// NewHub returns an empty hub.
func NewHub(log *slog.Logger) *Hub {
	return &Hub{rooms: map[string]map[*Client]struct{}{}, slots: map[slot]*Client{}, log: log}
}

func (h *Hub) join(room string, c *Client) {
	members := h.rooms[room]
	if members == nil {
		members = map[*Client]struct{}{}
		h.rooms[room] = members
	}
	members[c] = struct{}{}
}

func (h *Hub) leave(room string, c *Client) {
	if members := h.rooms[room]; members != nil {
		delete(members, c)
		if len(members) == 0 {
			delete(h.rooms, room)
		}
	}
}

func (h *Hub) broadcast(room string, msg any, exclude *Client) {
	for c := range h.rooms[room] {
		if c != exclude {
			c.Send(msg)
		}
	}
}

// RoomSize is the number of sockets in a room.
func (h *Hub) RoomSize(room string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.rooms[room])
}

// --- The professionals' live queue ------------------------------------------

// QueueRoom is the room every online verified professional is in.
const QueueRoom = "doctors:queue"

// JoinQueue adds a professional's socket and sends it the current queue.
func (h *Hub) JoinQueue(c *Client, snapshot func() any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.join(QueueRoom, c)
	c.Send(snapshot())
}

// LeaveQueue removes a professional's socket.
func (h *Hub) LeaveQueue(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.leave(QueueRoom, c)
}

// BroadcastQueue sends everyone in the queue room a fresh snapshot. It is
// taken under the hub's lock, so snapshots arrive in order.
func (h *Hub) BroadcastQueue(snapshot func() any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.rooms[QueueRoom]) > 0 {
		h.broadcast(QueueRoom, snapshot(), nil)
	}
}

// --- Call signalling ----------------------------------------------------------

// The messages of the signalling protocol (see package api for the flow).
type (
	joinedMsg struct {
		Type         string `json:"type"`
		Role         string `json:"role"`
		PeerPresent  bool   `json:"peer_present"`
		Professional any    `json:"professional"`
	}
	peerJoinedMsg struct {
		Type         string `json:"type"`
		Role         string `json:"role"`
		Professional any    `json:"professional"`
	}
	peerLeftMsg struct {
		Type string `json:"type"`
		Role string `json:"role"`
	}
)

func callRoom(callID string) string { return "consultation:" + callID }

func otherRole(role string) string {
	if role == "patient" {
		return "doctor"
	}
	return "patient"
}

// JoinCall puts a participant's socket in their call. A previous socket for
// the same participant is closed with CloseReplaced. The newcomer is told
// whether the other participant is already there; the one already there is
// told who arrived, and makes the WebRTC offer. Only one side ever gets
// "peer-joined" for an arrival, so both can never offer at once.
// professional is the badge of whoever accepted (nil for none).
func (h *Hub) JoinCall(callID, role string, professional any, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	key := slot{callID, role}
	if previous := h.slots[key]; previous != nil && previous != c {
		h.leave(callRoom(callID), previous)
		previous.Close(CloseReplaced, "")
	}
	h.slots[key] = c
	room := callRoom(callID)
	h.join(room, c)
	c.Send(joinedMsg{
		Type: "joined", Role: role, Professional: professional,
		PeerPresent: h.slots[slot{callID, otherRole(role)}] != nil,
	})
	h.broadcast(room, peerJoinedMsg{Type: "peer-joined", Role: role, Professional: professional}, c)
}

// Relay forwards a signal to the other participant.
func (h *Hub) Relay(callID string, from *Client, msg any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.broadcast(callRoom(callID), msg, from)
}

// LeaveCall removes a participant's socket. If it still held their slot
// (it wasn't replaced by a reconnect), the other participant hears
// "peer-left".
func (h *Hub) LeaveCall(callID, role string, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	room := callRoom(callID)
	h.leave(room, c)
	key := slot{callID, role}
	if h.slots[key] == c {
		delete(h.slots, key)
		h.broadcast(room, peerLeftMsg{Type: "peer-left", Role: role}, nil)
	}
}

// CloseAll closes every socket, e.g. on shutdown. Browsers reconnect to the
// next instance.
func (h *Hub) CloseAll() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, members := range h.rooms {
		for c := range members {
			c.Close(websocket.StatusGoingAway, "server restarting")
		}
	}
}
