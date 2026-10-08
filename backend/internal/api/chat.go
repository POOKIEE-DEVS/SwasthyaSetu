package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/ai"
	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/store"
)

// The first-aid chat. Guests chat without an account and nothing is stored;
// a signed-in patient's conversation is saved after each reply.

type chatRequest struct {
	// The browser holds the conversation and sends it every time, so the
	// model call stays stateless and guests need no account.
	Messages []ai.Message `json:"messages"`
	// Signed-in patients only: the saved chat this message continues.
	ChatID *string `json:"chat_id"`
}

type chatResponse struct {
	Reply string `json:"reply"`
	// The latest message mentions an obvious emergency, so the page shows
	// "Call 102 / Talk to a professional" whatever the reply says.
	Urgent bool `json:"urgent"`
	// Set when the exchange was saved to the signed-in patient's history.
	ChatID *string `json:"chat_id"`
}

// validMessages checks a conversation's shape: 1 to 100 turns, each from
// the user or the assistant, none empty.
func validMessages(messages []ai.Message) error {
	if len(messages) < 1 || len(messages) > 100 {
		return invalid("A conversation needs between 1 and 100 messages.")
	}
	for _, m := range messages {
		if m.Role != "user" && m.Role != "assistant" {
			return invalid("Each message must be from the user or the assistant.")
		}
		if m.Content == "" {
			return invalid("Messages can't be empty.")
		}
	}
	return nil
}

func (s *Server) chat(w http.ResponseWriter, r *http.Request) error {
	var body chatRequest
	if err := decodeJSON(w, r, &body); err != nil {
		return err
	}
	if err := validMessages(body.Messages); err != nil {
		return err
	}
	if body.ChatID != nil && chars(*body.ChatID) > 32 {
		return invalid("Unknown chat.")
	}
	if !s.limiter.Allow(clientIP(r)) {
		return fail(http.StatusTooManyRequests, "Too many messages. Please wait a minute and try again.")
	}
	latest := body.Messages[len(body.Messages)-1]
	if latest.Role != "user" {
		return invalid("The last message must be from the user.")
	}
	for _, m := range body.Messages {
		if chars(m.Content) > s.cfg.ChatMaxMessageChars {
			return invalid(fmt.Sprintf("Messages must be under %d characters.", s.cfg.ChatMaxMessageChars))
		}
	}

	urgent := ai.IsUrgent(latest.Content)
	var reply string
	if ai.IsOffTopic(latest.Content) {
		// Plainly not about health: a fixed answer, and the model never sees
		// it. Anything health-related or urgent goes on to the model.
		reply = ai.OffTopicReply(latest.Content)
	} else {
		var err error
		if reply, err = ai.Answer(r.Context(), s.model, body.Messages, s.cfg.ChatMaxHistoryMessages, s.log); err != nil {
			return err
		}
	}

	return writeOK(w, chatResponse{Reply: reply, Urgent: urgent, ChatID: s.saveExchange(r, body, reply)})
}

// saveExchange stores the conversation for a signed-in patient and returns
// the chat id. It never fails the chat: the reply matters more than the
// history, and the emergency path must not depend on the database.
func (s *Server) saveExchange(r *http.Request, body chatRequest, reply string) *string {
	token := sessionToken(r)
	if token == "" {
		return nil
	}
	// Finish even if the browser has stopped waiting.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 10*time.Second)
	defer cancel()
	user, err := s.userForToken(ctx, token)
	if err != nil || user == nil {
		if err != nil {
			s.log.Warn("could not look up the signed-in user; chat not saved", "error", err)
		}
		return nil
	}
	chatID := ""
	if body.ChatID != nil {
		chatID = *body.ChatID
	}
	id, err := s.store.SaveExchange(ctx, user.ID, chatID, toStore(body.Messages), reply)
	if err != nil {
		s.log.Warn("could not save chat history", "error", err)
		return nil
	}
	return &id
}

func toStore(messages []ai.Message) []store.Message {
	out := make([]store.Message, len(messages))
	for i, m := range messages {
		out[i] = store.Message{Role: m.Role, Content: m.Content}
	}
	return out
}

func writeOK(w http.ResponseWriter, v any) error {
	writeJSON(w, http.StatusOK, v)
	return nil
}

// --- Saved chats --------------------------------------------------------------

type chatSummary struct {
	ID           string  `json:"id"`
	Title        string  `json:"title"`
	UpdatedAt    float64 `json:"updated_at"`
	MessageCount int     `json:"message_count"`
}

type chatDetail struct {
	ID        string          `json:"id"`
	Title     string          `json:"title"`
	UpdatedAt float64         `json:"updated_at"`
	Messages  []store.Message `json:"messages"`
}

func (s *Server) listChats(w http.ResponseWriter, r *http.Request) error {
	user, err := s.currentUser(r)
	if err != nil {
		return err
	}
	chats, err := s.store.ListChats(r.Context(), user.ID)
	if err != nil {
		return err
	}
	out := []chatSummary{}
	for _, c := range chats {
		out = append(out, chatSummary{ID: c.ID, Title: c.Title, UpdatedAt: store.Unix(c.UpdatedAt), MessageCount: c.MessageCount})
	}
	return writeOK(w, out)
}

// ownedChat is the user's chat with the id in the path, or a 404 (also for
// someone else's chat, so ids can't be probed).
func (s *Server) ownedChat(r *http.Request) (*store.User, *store.Chat, error) {
	user, err := s.currentUser(r)
	if err != nil {
		return nil, nil, err
	}
	chat, err := s.store.OwnedChat(r.Context(), user.ID, r.PathValue("id"))
	if err != nil {
		return nil, nil, err
	}
	if chat == nil {
		return nil, nil, fail(http.StatusNotFound, "Chat not found.")
	}
	return user, chat, nil
}

func (s *Server) openChat(w http.ResponseWriter, r *http.Request) error {
	_, chat, err := s.ownedChat(r)
	if err != nil {
		return err
	}
	messages, err := s.store.ChatMessages(r.Context(), chat.ID)
	if err != nil {
		return err
	}
	return writeOK(w, chatDetail{ID: chat.ID, Title: chat.Title, UpdatedAt: store.Unix(chat.UpdatedAt), Messages: messages})
}

// saveChat keeps a conversation that was started before signing in.
func (s *Server) saveChat(w http.ResponseWriter, r *http.Request) error {
	user, err := s.currentUser(r)
	if err != nil {
		return err
	}
	var body struct {
		Messages []ai.Message `json:"messages"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		return err
	}
	if err := validMessages(body.Messages); err != nil {
		return err
	}
	chat, err := s.store.CreateChat(r.Context(), user.ID, toStore(body.Messages))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, chatSummary{
		ID: chat.ID, Title: chat.Title, UpdatedAt: store.Unix(chat.UpdatedAt), MessageCount: len(body.Messages),
	})
	return nil
}

func (s *Server) deleteChat(w http.ResponseWriter, r *http.Request) error {
	user, chat, err := s.ownedChat(r)
	if err != nil {
		return err
	}
	if err := s.store.DeleteChat(r.Context(), user.ID, chat.ID); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
