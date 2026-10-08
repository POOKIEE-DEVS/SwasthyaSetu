package ai

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// Messages the patient may see when the model can't answer.
const (
	MsgNotConfigured = "The AI model is not configured."
	MsgTimeout       = "The AI model took too long to answer. It may be waking up; " +
		"please try again in a minute."
	MsgUnavailable = "The AI model is unavailable right now. Please try again shortly."
	MsgNoAnswer    = "The assistant couldn't finish an answer. Please try again, " +
		"or press Talk to a professional."
)

// ModelError is a failure whose message is safe to show to the patient.
type ModelError struct{ Message string }

func (e *ModelError) Error() string { return e.Message }

// Generator produces a reply for a conversation. MedGemma is the real one;
// tests use fakes.
type Generator interface {
	Generate(ctx context.Context, messages []Message) (string, error)
}

// MedGemma generates replies through the model server's /generate endpoint,
// which takes the conversation as a JSON string and returns the reply text.
type MedGemma struct {
	gradio  *Gradio // nil when no model is configured
	timeout time.Duration
	log     *slog.Logger
}

// NewMedGemma returns a client for a Space id or app URL (empty: not
// configured).
func NewMedGemma(source, token string, timeout time.Duration, log *slog.Logger) *MedGemma {
	m := &MedGemma{timeout: timeout, log: log}
	if source != "" {
		m.gradio = NewGradio(source, token, &http.Client{Transport: modelTransport()})
	}
	return m
}

func modelTransport() *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.DialContext = (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext
	t.TLSHandshakeTimeout = 10 * time.Second
	return t
}

// Generate asks the model and returns its patient-facing reply. Errors are
// *ModelError.
func (m *MedGemma) Generate(ctx context.Context, messages []Message) (string, error) {
	if m.gradio == nil {
		return "", &ModelError{MsgNotConfigured}
	}
	payload, err := json.Marshal(messages)
	if err != nil {
		return "", err
	}
	callCtx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()
	raw, err := m.gradio.Call(callCtx, "generate", string(payload))
	if err == nil {
		var reply string
		if err = json.Unmarshal(raw, &reply); err != nil {
			err = errors.New("gradio: the reply is not text")
		} else if reply = StripThinking(reply); reply == "" {
			return "", &ModelError{MsgNoAnswer}
		} else {
			return reply, nil
		}
	}
	if errors.Is(callCtx.Err(), context.DeadlineExceeded) && ctx.Err() == nil {
		return "", &ModelError{MsgTimeout}
	}
	// A stale address (the app restarted elsewhere) is the common cause:
	// look it up again on the next request.
	m.gradio.Reset()
	if ctx.Err() == nil {
		m.log.Warn("medgemma request failed", "error", err)
	}
	return "", &ModelError{MsgUnavailable}
}

// Answer returns the model's reply, never its analysis.
//
// If the reply reads as the model working through the question ("The user
// has asked ... Plan: ..."), it asks once more with FallbackSystemPrompt and
// only the latest message. Different input, so a different answer even with
// greedy decoding. If that is analysis too, it fails with MsgNoAnswer rather
// than show it.
func Answer(ctx context.Context, gen Generator, history []Message, maxHistory int, log *slog.Logger) (string, error) {
	reply, err := gen.Generate(ctx, BuildModelMessages(history, maxHistory, ""))
	if err != nil {
		return "", err
	}
	if !LooksLikeReasoning(reply) {
		return reply, nil
	}
	log.Warn("model replied with its analysis; retrying with the fallback prompt")
	retry, err := gen.Generate(ctx, BuildModelMessages(history[len(history)-1:], 1, FallbackSystemPrompt))
	if err != nil {
		return "", err
	}
	if LooksLikeReasoning(retry) {
		return "", &ModelError{MsgNoAnswer}
	}
	return retry, nil
}
