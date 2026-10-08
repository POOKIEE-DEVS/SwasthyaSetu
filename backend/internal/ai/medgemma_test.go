package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/logging"
)

// fakeGradio speaks Gradio's call API, like model-space/app.py does.
type fakeGradio struct {
	*httptest.Server
	mu          sync.Mutex
	configCalls int
	sent        [][]Message
	// respond returns the event name and data line for a call.
	respond func(messages []Message) (string, string)
	delay   time.Duration
}

func newFakeGradio(t *testing.T) *fakeGradio {
	f := &fakeGradio{respond: func(m []Message) (string, string) {
		reply, _ := json.Marshal([]string{"1. Stay calm.\n2. Rest."})
		return "complete", string(reply)
	}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /config", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.configCalls++
		f.mu.Unlock()
		fmt.Fprint(w, `{"api_prefix": "/gradio_api", "version": "6.28.0"}`)
	})
	mux.HandleFunc("POST /gradio_api/call/generate", func(w http.ResponseWriter, r *http.Request) {
		var body struct{ Data []string }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Data) != 1 {
			http.Error(w, "bad body", http.StatusUnprocessableEntity)
			return
		}
		var messages []Message
		if err := json.Unmarshal([]byte(body.Data[0]), &messages); err != nil {
			http.Error(w, "bad messages", http.StatusUnprocessableEntity)
			return
		}
		f.mu.Lock()
		f.sent = append(f.sent, messages)
		id := len(f.sent) - 1
		f.mu.Unlock()
		fmt.Fprintf(w, `{"event_id": "e%d"}`, id)
	})
	mux.HandleFunc("GET /gradio_api/call/generate/{event}", func(w http.ResponseWriter, r *http.Request) {
		var id int
		fmt.Sscanf(r.PathValue("event"), "e%d", &id)
		f.mu.Lock()
		messages := f.sent[id]
		delay := f.delay
		f.mu.Unlock()
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "event: heartbeat\ndata: null\n\n")
		w.(http.Flusher).Flush()
		select {
		case <-time.After(delay):
		case <-r.Context().Done():
			return
		}
		event, data := f.respond(messages)
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
	})
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)
	return f
}

func say(text string) []Message { return []Message{{Role: "user", Content: text}} }

func modelError(t *testing.T, err error) string {
	t.Helper()
	var me *ModelError
	if !errors.As(err, &me) {
		t.Fatalf("want a *ModelError, got %v", err)
	}
	return me.Message
}

func TestGenerateSendsTheConversationAndReturnsTheReply(t *testing.T) {
	f := newFakeGradio(t)
	m := NewMedGemma(f.URL, "", 5*time.Second, logging.Discard())
	ctx := context.Background()
	messages := BuildModelMessages(say("मलाई ज्वरो आयो"), 8, "")

	for i := 0; i < 2; i++ {
		reply, err := m.Generate(ctx, messages)
		if err != nil || reply != "1. Stay calm.\n2. Rest." {
			t.Fatalf("reply %q, err %v", reply, err)
		}
	}
	if !reflect.DeepEqual(f.sent[0], messages) {
		t.Fatalf("sent %v", f.sent[0])
	}
	if f.configCalls != 1 {
		t.Fatalf("the app's config should be read once, was read %d times", f.configCalls)
	}
}

func TestGenerateStripsThinking(t *testing.T) {
	f := newFakeGradio(t)
	f.respond = func([]Message) (string, string) {
		return "complete", `["<unused94>thought\nhmm<unused95>1. Cool the burn."]`
	}
	reply, err := NewMedGemma(f.URL, "", 5*time.Second, logging.Discard()).Generate(context.Background(), say("burn"))
	if err != nil || reply != "1. Cool the burn." {
		t.Fatalf("reply %q, err %v", reply, err)
	}
}

func TestAnUnfinishedThoughtIsNoAnswer(t *testing.T) {
	f := newFakeGradio(t)
	f.respond = func([]Message) (string, string) { return "complete", `["<unused94>thought\nI need to"]` }
	_, err := NewMedGemma(f.URL, "", 5*time.Second, logging.Discard()).Generate(context.Background(), say("burn"))
	if msg := modelError(t, err); msg != MsgNoAnswer {
		t.Fatalf("message %q", msg)
	}
}

func TestModelErrorsAreClearAndReconnect(t *testing.T) {
	f := newFakeGradio(t)
	f.respond = func([]Message) (string, string) {
		return "error", `{"error": "CUDA out of memory", "visible": true}`
	}
	m := NewMedGemma(f.URL, "", 5*time.Second, logging.Discard())
	_, err := m.Generate(context.Background(), say("burn"))
	if msg := modelError(t, err); msg != MsgUnavailable {
		t.Fatalf("message %q", msg)
	}
	_, _ = m.Generate(context.Background(), say("burn"))
	if f.configCalls != 2 {
		t.Fatalf("a failure should look the app up again; config read %d times", f.configCalls)
	}
}

func TestSlowModelTimesOut(t *testing.T) {
	f := newFakeGradio(t)
	f.delay = 2 * time.Second
	start := time.Now()
	_, err := NewMedGemma(f.URL, "", 200*time.Millisecond, logging.Discard()).Generate(context.Background(), say("burn"))
	if msg := modelError(t, err); msg != MsgTimeout {
		t.Fatalf("message %q", msg)
	}
	if time.Since(start) > time.Second {
		t.Fatal("the timeout was not applied")
	}
}

func TestUnconfiguredModel(t *testing.T) {
	_, err := NewMedGemma("", "", time.Second, logging.Discard()).Generate(context.Background(), say("hi"))
	if msg := modelError(t, err); msg != MsgNotConfigured {
		t.Fatalf("message %q", msg)
	}
}

func TestUnreachableModel(t *testing.T) {
	_, err := NewMedGemma("http://127.0.0.1:1/", "", 2*time.Second, logging.Discard()).Generate(context.Background(), say("hi"))
	if msg := modelError(t, err); msg != MsgUnavailable {
		t.Fatalf("message %q", msg)
	}
}

func TestSpaceIDIsLookedUpOnHuggingFace(t *testing.T) {
	f := newFakeGradio(t)
	var askedFor string
	hf := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		askedFor = r.URL.Path
		if r.Header.Get("Authorization") != "" {
			t.Error("a token must only go to Hugging Face hosts")
		}
		fmt.Fprintf(w, `{"id": "pookie/medgemma", "host": %q}`, f.URL)
	}))
	defer hf.Close()

	g := NewGradio("pookie/medgemma", "hf_secret", hf.Client())
	g.hfAPI = hf.URL
	out, err := g.Call(context.Background(), "generate", `[{"role":"user","content":"hi"}]`)
	if err != nil || string(out) != `"1. Stay calm.\n2. Rest."` {
		t.Fatalf("out %s, err %v", out, err)
	}
	if askedFor != "/api/spaces/pookie/medgemma" {
		t.Fatalf("looked up %q", askedFor)
	}
	if _, err := NewGradio("not-a-space-id", "", nil).Call(context.Background(), "generate"); err == nil {
		t.Fatal("a malformed Space id should fail")
	}
}

func TestTokenOnlyGoesToHuggingFace(t *testing.T) {
	for raw, want := range map[string]bool{
		"https://huggingface.co/api/spaces/a/b": true,
		"https://pookie-medgemma.hf.space/":     true,
		"https://abc123.gradio.live/":           false,
		"https://evil.example/hf.space":         false,
	} {
		u, _ := url.Parse(raw)
		if isHuggingFace(u) != want {
			t.Errorf("isHuggingFace(%s) != %v", raw, want)
		}
	}
}

func TestReadResult(t *testing.T) {
	cases := []struct {
		stream, out, err string
	}{
		{"event: heartbeat\ndata: null\n\nevent: complete\ndata: [\"ok\"]\n\n", `"ok"`, ""},
		{"event: complete\ndata: [\"no blank line\"]", `"no blank line"`, ""},
		{"event: complete\r\ndata: [\"crlf\"]\r\n\r\n", `"crlf"`, ""},
		{"event: error\ndata: {\"error\": \"boom\"}\n\n", "", "boom"},
		{"event: heartbeat\ndata: null\n\n", "", "ended without a result"},
		{"event: complete\ndata: []\n\n", "", "empty result"},
	}
	for _, c := range cases {
		out, err := readResult(strings.NewReader(c.stream))
		if c.err != "" {
			if err == nil || !strings.Contains(err.Error(), c.err) {
				t.Errorf("%q: err %v, want %q", c.stream, err, c.err)
			}
			continue
		}
		if err != nil || string(out) != c.out {
			t.Errorf("%q: out %s, err %v", c.stream, out, err)
		}
	}
}

// scripted returns canned replies in order and records what it was sent.
type scripted struct {
	replies []string
	calls   [][]Message
}

func (s *scripted) Generate(_ context.Context, messages []Message) (string, error) {
	s.calls = append(s.calls, messages)
	return s.replies[len(s.calls)-1], nil
}

func TestAnalysisIsRetriedOnceWithTheFallbackPrompt(t *testing.T) {
	gen := &scripted{replies: []string{analysisNE, "१. टोकेको ठाउँ धुनुहोस्।"}}
	history := []Message{
		{Role: "user", Content: "Hello"}, {Role: "assistant", Content: "Hi"},
		{Role: "user", Content: "मौरीले टोक्यो"},
	}
	reply, err := Answer(context.Background(), gen, history, 8, logging.Discard())
	if err != nil || reply != "१. टोकेको ठाउँ धुनुहोस्।" {
		t.Fatalf("reply %q, err %v", reply, err)
	}
	want := []Message{{Role: "system", Content: FallbackSystemPrompt}, {Role: "user", Content: "मौरीले टोक्यो"}}
	if len(gen.calls) != 2 || !reflect.DeepEqual(gen.calls[1], want) {
		t.Fatalf("calls %v", gen.calls)
	}
}

func TestAnalysisIsNeverShown(t *testing.T) {
	gen := &scripted{replies: []string{analysisEN, analysisEN}}
	_, err := Answer(context.Background(), gen, say("Someone fell"), 8, logging.Discard())
	if msg := modelError(t, err); msg != MsgNoAnswer {
		t.Fatalf("message %q", msg)
	}
}

// With TEST_GRADIO_URL pointing at a running model-space/app.py
// (SWASTHYA_MOCK_MODEL=1 is enough), this checks the real protocol.
func TestAgainstARealGradioApp(t *testing.T) {
	source := os.Getenv("TEST_GRADIO_URL")
	if source == "" {
		t.Skip("TEST_GRADIO_URL not set")
	}
	m := NewMedGemma(source, "", 60*time.Second, logging.Discard())
	reply, err := m.Generate(context.Background(), BuildModelMessages(say("नमस्ते, I cut my finger"), 8, ""))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(reply, "नमस्ते, I cut my finger") {
		t.Fatalf("reply %q", reply)
	}
}
