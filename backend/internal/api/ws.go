package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/coder/websocket"

	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/consult"
	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/realtime"
)

// The WebSockets.
//
// /ws/doctors
//
//	The live list of waiting patients, for verified professionals only
//	(checked from the session cookie). A snapshot is pushed on connect and
//	after every change.
//
// /ws/consultations/{id}?token=...
//
//	WebRTC signalling for one call. The token decides the role (patient or
//	doctor), and only those two tokens exist, so nobody else can join.
//	Offer, answer, ICE candidates and hangup are relayed to the other
//	participant.
//
// Who makes the WebRTC offer: whoever is already in the room when the other
// participant arrives. They receive "peer-joined" and make the offer. The
// same rule recovers from a page refresh: the returning participant arrives,
// and the one who stayed makes a fresh offer.
//
// A refused socket is accepted first and then closed with a code (4401 not
// verified, 4403 unknown or ended call), so the browser can tell why;
// refusing the handshake itself would only show it a failed connection.

const maxSignal = 1 << 20 // bytes; SDP offers are a few kilobytes

func (s *Server) acceptSocket(w http.ResponseWriter, r *http.Request) (*websocket.Conn, bool) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// Our own pages are always allowed; these are the dev origins.
		OriginPatterns: s.wsOrigins,
	})
	if err != nil {
		s.log.Info("websocket refused", "path", r.URL.Path, "error", err)
		return nil, false
	}
	conn.SetReadLimit(maxSignal)
	return conn, true
}

// readUntilClosed reads (and ignores) messages until the socket closes.
// Reading is what notices a disconnect and answers pings.
func readUntilClosed(ctx context.Context, conn *websocket.Conn) {
	for {
		if _, _, err := conn.Read(ctx); err != nil {
			return
		}
	}
}

func (s *Server) queueSocket(w http.ResponseWriter, r *http.Request) {
	user, err := s.optionalUser(r)
	var pro *professional
	if err == nil && user != nil {
		pro, err = s.verifiedProfessional(r.Context(), user)
	}
	conn, ok := s.acceptSocket(w, r)
	if !ok {
		return
	}
	if err != nil {
		s.log.Warn("queue socket: could not check the professional", "error", err)
		_ = conn.Close(websocket.StatusInternalError, "try again")
		return
	}
	if pro == nil {
		_ = conn.Close(realtime.CloseNotVerified, "")
		return
	}

	client := realtime.NewClient(conn, s.log)
	s.hub.JoinQueue(client, s.queueSnapshot)
	defer func() {
		s.hub.LeaveQueue(client)
		client.Close(websocket.StatusNormalClosure, "")
	}()
	readUntilClosed(r.Context(), conn)
}

type signal struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type relayedSignal struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
	From    string          `json:"from"`
}

type signalError struct {
	Type   string `json:"type"`
	Detail string `json:"detail"`
}

var signalTypes = map[string]bool{"offer": true, "answer": true, "ice-candidate": true, "hangup": true}

// parseSignal accepts the four signal types, with an object or null (or no)
// payload.
func parseSignal(data []byte) (signal, bool) {
	var sig signal
	if err := json.Unmarshal(data, &sig); err != nil || !signalTypes[sig.Type] {
		return sig, false
	}
	payload := bytes.TrimSpace(sig.Payload)
	switch {
	case len(payload) == 0 || bytes.Equal(payload, []byte("null")):
		sig.Payload = nil
	case payload[0] != '{':
		return sig, false
	}
	return sig, true
}

func (s *Server) callSocket(w http.ResponseWriter, r *http.Request) {
	callID := r.PathValue("id")
	c, err := s.consults.Get(callID)
	role := ""
	if err == nil {
		role = c.RoleFor(r.URL.Query().Get("token"))
	}
	conn, ok := s.acceptSocket(w, r)
	if !ok {
		return
	}
	if role == "" || c.Status == consult.Ended {
		_ = conn.Close(realtime.CloseUnauthorized, "")
		return
	}

	client := realtime.NewClient(conn, s.log)
	// Who accepted, so the patient sees "Verified Doctor" and their name.
	s.hub.JoinCall(callID, role, c.Professional, client)
	defer func() {
		s.hub.LeaveCall(callID, role, client)
		client.Close(websocket.StatusNormalClosure, "")
	}()

	for {
		_, data, err := conn.Read(r.Context())
		if err != nil {
			return
		}
		sig, ok := parseSignal(data)
		if !ok {
			client.Send(signalError{Type: "error", Detail: "unsupported signal"})
			continue
		}
		s.hub.Relay(callID, client, relayedSignal{Type: sig.Type, Payload: sig.Payload, From: role})
	}
}
