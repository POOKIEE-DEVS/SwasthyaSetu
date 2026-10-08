package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/database"
)

// Saved conversations for signed-in patients. Guests' chats are never
// stored. Every query filters by the owner, and a chat id belonging to
// someone else is treated as unknown, never as forbidden, so ids can't be
// probed.

const (
	maxListedChats = 50
	titleChars     = 60
)

// Message is one chat turn.
type Message struct {
	Role    string `json:"role"` // user | assistant
	Content string `json:"content"`
}

// Chat is a saved conversation.
type Chat struct {
	ID        string
	UserID    int64
	Title     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ChatSummary is a chat with its message count, for the list.
type ChatSummary struct {
	Chat
	MessageCount int
}

// ChatTitle is the first user message (or "New chat"), on one line and at
// most 60 characters.
func ChatTitle(messages []Message) string {
	first := "New chat"
	for _, m := range messages {
		if m.Role == "user" {
			first = m.Content
			break
		}
	}
	text := strings.Join(strings.Fields(first), " ")
	if utf8.RuneCountInString(text) <= titleChars {
		return text
	}
	return string([]rune(text)[:titleChars-1]) + "…"
}

func newChatID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b) // never fails
	return hex.EncodeToString(b)
}

// OwnedChat returns the user's chat with that id, or nil.
func (s *Store) OwnedChat(ctx context.Context, userID int64, chatID string) (*Chat, error) {
	if chatID == "" {
		return nil, nil
	}
	return ownedChat(ctx, s.db, userID, chatID)
}

func ownedChat(ctx context.Context, q database.Querier, userID int64, chatID string) (*Chat, error) {
	var c Chat
	err := q.QueryRowContext(ctx,
		`SELECT id, user_id, title, created_at, updated_at FROM chats WHERE id = $1 AND user_id = $2`,
		chatID, userID,
	).Scan(&c.ID, &c.UserID, &c.Title, timeCol{&c.CreatedAt}, timeCol{&c.UpdatedAt})
	return one(&c, err)
}

// CreateChat saves a new conversation.
func (s *Store) CreateChat(ctx context.Context, userID int64, messages []Message) (*Chat, error) {
	var chat *Chat
	err := s.db.InTx(ctx, func(tx *database.Tx) error {
		var err error
		chat, err = createChat(ctx, tx, userID, messages)
		return err
	})
	return chat, err
}

func createChat(ctx context.Context, tx *database.Tx, userID int64, messages []Message) (*Chat, error) {
	now := Now()
	chat := &Chat{ID: newChatID(), UserID: userID, Title: ChatTitle(messages), CreatedAt: now, UpdatedAt: now}
	_, err := tx.ExecContext(ctx,
		`INSERT INTO chats (id, user_id, title, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
		chat.ID, chat.UserID, chat.Title, chat.CreatedAt, chat.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return chat, addMessages(ctx, tx, chat, messages)
}

func addMessages(ctx context.Context, tx *database.Tx, chat *Chat, messages []Message) error {
	now := Now()
	for _, m := range messages {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO chat_messages (chat_id, role, content, created_at) VALUES ($1, $2, $3, $4)`,
			chat.ID, m.Role, m.Content, now)
		if err != nil {
			return err
		}
	}
	chat.UpdatedAt = now
	_, err := tx.ExecContext(ctx, `UPDATE chats SET updated_at = $1 WHERE id = $2`, now, chat.ID)
	return err
}

// SaveExchange stores a conversation up to and including a new reply, and
// returns the chat id. A chat id that isn't the user's starts a new chat.
// Only what isn't stored yet is added: normally the latest question and the
// reply, more if an earlier question had failed (a failed reply saves
// nothing, so a retry never duplicates a message).
func (s *Store) SaveExchange(ctx context.Context, userID int64, chatID string, history []Message, reply string) (string, error) {
	conversation := append(append([]Message{}, history...), Message{Role: "assistant", Content: reply})
	var id string
	err := s.db.InTx(ctx, func(tx *database.Tx) error {
		var chat *Chat
		var err error
		if chatID != "" {
			if chat, err = ownedChat(ctx, tx, userID, chatID); err != nil {
				return err
			}
		}
		if chat == nil {
			chat, err = createChat(ctx, tx, userID, conversation)
			if err == nil {
				id = chat.ID
			}
			return err
		}
		var stored int
		err = tx.QueryRowContext(ctx, `SELECT count(*) FROM chat_messages WHERE chat_id = $1`, chat.ID).Scan(&stored)
		if err != nil {
			return err
		}
		fresh := conversation[len(conversation)-2:]
		if stored < len(conversation) {
			fresh = conversation[stored:]
		}
		id = chat.ID
		return addMessages(ctx, tx, chat, fresh)
	})
	return id, err
}

// ListChats returns a user's most recent chats, newest first. Ties (two
// chats saved within the clock's resolution) go to the newest message.
func (s *Store) ListChats(ctx context.Context, userID int64) ([]ChatSummary, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT c.id, c.user_id, c.title, c.created_at, c.updated_at,
			(SELECT count(*) FROM chat_messages m WHERE m.chat_id = c.id)
		 FROM chats c
		 WHERE c.user_id = $1
		 ORDER BY c.updated_at DESC,
			COALESCE((SELECT max(m.id) FROM chat_messages m WHERE m.chat_id = c.id), 0) DESC
		 LIMIT $2`, userID, maxListedChats)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ChatSummary{}
	for rows.Next() {
		var c ChatSummary
		err := rows.Scan(&c.ID, &c.UserID, &c.Title, timeCol{&c.CreatedAt}, timeCol{&c.UpdatedAt}, &c.MessageCount)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ChatMessages returns a chat's messages in order.
func (s *Store) ChatMessages(ctx context.Context, chatID string) ([]Message, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT role, content FROM chat_messages WHERE chat_id = $1 ORDER BY id`, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Message{}
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.Role, &m.Content); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// DeleteChat removes a chat and its messages.
func (s *Store) DeleteChat(ctx context.Context, userID int64, chatID string) error {
	return s.db.InTx(ctx, func(tx *database.Tx) error {
		// Messages first (the chat row must go last), and only the owner's.
		_, err := tx.ExecContext(ctx,
			`DELETE FROM chat_messages WHERE chat_id IN (SELECT id FROM chats WHERE id = $1 AND user_id = $2)`,
			chatID, userID)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM chats WHERE id = $1 AND user_id = $2`, chatID, userID)
		return err
	})
}
