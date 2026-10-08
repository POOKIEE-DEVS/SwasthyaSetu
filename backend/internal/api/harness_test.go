package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/ai"
	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/config"
	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/database"
	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/logging"
	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/store"
	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/testdb"
)

// The test harness: a real server on a local port with a fresh database,
// a fake model, and clients that keep cookies like a browser. No network:
// the model and Google are faked.

const (
	adminEmail  = "admin@example.com"
	doctorEmail = "doctor@example.com"
	doctorName  = "Dr. Anita Karki"
	modelReply  = "1. Stay calm.\n2. Rest."
)

// fakeModel replaces the model server and records what it was sent.
type fakeModel struct {
	mu       sync.Mutex
	calls    [][]ai.Message
	generate func(call int, messages []ai.Message) (string, error)
}

func (f *fakeModel) Generate(_ context.Context, messages []ai.Message) (string, error) {
	f.mu.Lock()
	f.calls = append(f.calls, messages)
	n, generate := len(f.calls), f.generate
	f.mu.Unlock()
	if generate != nil {
		return generate(n, messages)
	}
	return modelReply, nil
}

func (f *fakeModel) set(generate func(call int, messages []ai.Message) (string, error)) {
	f.mu.Lock()
	f.generate = generate
	f.mu.Unlock()
}

func (f *fakeModel) sent() [][]ai.Message {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([][]ai.Message(nil), f.calls...)
}

type env struct {
	t      *testing.T
	cfg    *config.Config
	db     *database.DB
	server *Server
	http   *httptest.Server
	model  *fakeModel
}

type setup struct {
	configure func(*config.Config)
	db        *database.DB
	model     ai.Generator // default: the fake
}

func newEnv(t *testing.T, options ...func(*setup)) *env {
	t.Helper()
	cfg := config.Defaults()
	cfg.StaticDir = "does-not-exist"
	cfg.DevLogin = true
	cfg.AdminEmails = []string{adminEmail}
	st := &setup{}
	for _, option := range options {
		option(st)
	}
	if st.configure != nil {
		st.configure(cfg)
	}
	if st.db == nil {
		st.db = testdb.Open(t)
	}
	e := &env{t: t, cfg: cfg, db: st.db, model: &fakeModel{}}
	model := st.model
	if model == nil {
		model = e.model
	}
	e.server = New(cfg, st.db, model, logging.Discard())
	e.http = httptest.NewServer(e.server.Handler())
	t.Cleanup(func() {
		e.server.Shutdown()
		e.http.Close()
		_ = e.server.Close()
	})
	return e
}

func withConfig(configure func(*config.Config)) func(*setup) {
	return func(s *setup) { s.configure = configure }
}

// client is a browser: it keeps cookies and doesn't follow redirects.
type client struct {
	e    *env
	http *http.Client
}

func (e *env) client() *client {
	c := &client{e: e}
	c.clearCookies()
	return c
}

func (c *client) clearCookies() {
	jar, _ := cookiejar.New(nil)
	c.http = &http.Client{
		Jar:           jar,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

type response struct {
	t      *testing.T
	status int
	header http.Header
	body   []byte
}

func (r *response) decode(v any) {
	r.t.Helper()
	if err := json.Unmarshal(r.body, v); err != nil {
		r.t.Fatalf("response %d is not JSON: %s", r.status, r.body)
	}
}

// m returns the body as a JSON object.
func (r *response) m() map[string]any {
	var out map[string]any
	r.decode(&out)
	return out
}

// list returns the body as a JSON array of objects.
func (r *response) list() []map[string]any {
	var out []map[string]any
	r.decode(&out)
	return out
}

func (r *response) detail() string {
	d, _ := r.m()["detail"].(string)
	return d
}

func (c *client) request(method, path string, body io.Reader, contentType string, headers ...string) *response {
	c.e.t.Helper()
	req, err := http.NewRequest(method, c.e.http.URL+path, body)
	if err != nil {
		c.e.t.Fatal(err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	resp, err := c.http.Do(req)
	if err != nil {
		c.e.t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return &response{t: c.e.t, status: resp.StatusCode, header: resp.Header, body: data}
}

func (c *client) get(path string, headers ...string) *response {
	return c.request(http.MethodGet, path, nil, "", headers...)
}

func (c *client) delete(path string) *response {
	return c.request(http.MethodDelete, path, nil, "")
}

// post sends JSON (or no body for nil).
func (c *client) post(path string, body any) *response {
	if body == nil {
		return c.request(http.MethodPost, path, nil, "")
	}
	data, _ := json.Marshal(body)
	return c.request(http.MethodPost, path, bytes.NewReader(data), "application/json")
}

type obj = map[string]any

// signIn uses the development login; the client keeps the session cookie.
func (c *client) signIn(email, name string) map[string]any {
	c.e.t.Helper()
	r := c.post("/api/v1/auth/dev-login", obj{"email": email, "name": name})
	if r.status != http.StatusOK {
		c.e.t.Fatalf("dev login: %d %s", r.status, r.body)
	}
	return r.m()
}

// makeProfessional signs the client in as a professional whose application
// has a status. Written straight to the database; the review flow itself is
// tested in applications_test.go.
func (c *client) makeProfessional(email, name, role, status string) map[string]any {
	c.e.t.Helper()
	user := c.signIn(email, name)
	ctx := context.Background()
	s := c.e.server.store
	u, err := s.UserByID(ctx, int64(user["id"].(float64)))
	if err != nil || u == nil {
		c.e.t.Fatalf("user: %v", err)
	}
	council := "12345"
	a, err := s.SubmitApplication(ctx, u, store.ApplicationFields{
		Role: role, FullName: name, Phone: "9800000000", CitizenshipNumber: "27-01-71-12345",
		CitizenshipDistrict: "Kathmandu", CouncilNumber: &council,
	}, nil)
	if err != nil {
		c.e.t.Fatal(err)
	}
	switch status {
	case store.StatusApproved:
		_, err = s.DecideApplication(ctx, a.ID, adminEmail, true, nil)
	case store.StatusRejected:
		reason := "Not found on the register."
		_, err = s.DecideApplication(ctx, a.ID, adminEmail, false, &reason)
	}
	if err != nil {
		c.e.t.Fatal(err)
	}
	return user
}

func (c *client) makeDoctor() map[string]any {
	return c.makeProfessional(doctorEmail, doctorName, "doctor", store.StatusApproved)
}

// --- WebSockets -------------------------------------------------------------

func (c *client) dial(path string) (*websocket.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	wsURL := "ws" + strings.TrimPrefix(c.e.http.URL, "http") + path
	conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{HTTPClient: c.http})
	if err == nil {
		c.e.t.Cleanup(func() { _ = conn.CloseNow() })
	}
	return conn, err
}

func (c *client) mustDial(path string) *websocket.Conn {
	c.e.t.Helper()
	conn, err := c.dial(path)
	if err != nil {
		c.e.t.Fatalf("dial %s: %v", path, err)
	}
	return conn
}

func readMessage(t *testing.T, conn *websocket.Conn) (map[string]any, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, data, err := conn.Read(ctx)
	if err != nil {
		return nil, err
	}
	var msg map[string]any
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatalf("not JSON: %s", data)
	}
	return msg, nil
}

func receive(t *testing.T, conn *websocket.Conn) map[string]any {
	t.Helper()
	msg, err := readMessage(t, conn)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return msg
}

func send(t *testing.T, conn *websocket.Conn, msg any) {
	t.Helper()
	data, _ := json.Marshal(msg)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// closeCode is the code a socket was closed with (-1 if it wasn't).
func closeCode(t *testing.T, conn *websocket.Conn) websocket.StatusCode {
	t.Helper()
	_, err := readMessage(t, conn)
	return websocket.CloseStatus(err)
}

// --- Google -------------------------------------------------------------------

func fakeIDToken(overrides obj) string {
	claims := obj{
		"iss": "https://accounts.google.com", "aud": "client-123",
		"exp": float64(time.Now().Add(10 * time.Minute).Unix()), "sub": "google-sub-1",
		"email": "Sita@Example.com", "email_verified": true, "name": "Sita Sharma",
		"picture": "https://example.com/p.png",
	}
	for k, v := range overrides {
		claims[k] = v
	}
	payload, _ := json.Marshal(claims)
	return "header." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
}

// googleExchange records the code exchanges a fake Google token endpoint got.
type googleExchange struct {
	mu    sync.Mutex
	forms []url.Values
}

func (g *googleExchange) all() []url.Values {
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([]url.Values(nil), g.forms...)
}

// withGoogle configures Google sign-in, with its token endpoint faked.
func withGoogle(t *testing.T, admins ...string) (func(*setup), *googleExchange, func(*env)) {
	exchanges := &googleExchange{}
	token := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		exchanges.mu.Lock()
		exchanges.forms = append(exchanges.forms, r.PostForm)
		exchanges.mu.Unlock()
		writeJSON(w, http.StatusOK, obj{"id_token": fakeIDToken(nil)})
	}))
	t.Cleanup(token.Close)
	option := withConfig(func(cfg *config.Config) {
		cfg.GoogleClientID, cfg.GoogleClientSecret = "client-123", "secret"
		if len(admins) > 0 {
			cfg.AdminEmails = admins
		}
	})
	attach := func(e *env) { e.server.Google().TokenURL = token.URL }
	return option, exchanges, attach
}
