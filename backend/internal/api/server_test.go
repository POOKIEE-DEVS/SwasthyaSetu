package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/config"
	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/database"
	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/logging"
)

func TestHealth(t *testing.T) {
	c := newEnv(t).client()
	body := c.get("/health").m()
	if body["status"] != "ok" || body["version"] != Version || body["model_configured"] != false ||
		body["turn_configured"] != false || body["google_sign_in_configured"] != false ||
		body["database_ready"] != true {
		t.Fatalf("health %v", body)
	}
	if kind := body["database"]; kind != "sqlite" && kind != "postgres" {
		t.Fatalf("database %v", kind)
	}
	if _, ok := body["uptime_seconds"].(float64); !ok {
		t.Fatalf("uptime %v", body["uptime_seconds"])
	}
}

func TestHealthUnderAPIPrefix(t *testing.T) {
	if r := newEnv(t).client().get("/api/v1/health"); r.status != http.StatusOK {
		t.Fatalf("status %d", r.status)
	}
}

func TestRequestIDIsEchoed(t *testing.T) {
	c := newEnv(t).client()
	if got := c.get("/api/v1/health", "X-Request-ID", "abc").header.Get("X-Request-ID"); got != "abc" {
		t.Fatalf("X-Request-ID = %q", got)
	}
	if got := c.get("/api/v1/health").header.Get("X-Request-ID"); len(got) != 32 {
		t.Fatalf("a new request id should be made: %q", got)
	}
}

func TestUnknownAPIPathIsAJSON404(t *testing.T) {
	r := newEnv(t).client().get("/api/v1/nope")
	if r.status != http.StatusNotFound || r.detail() != "Not Found" {
		t.Fatalf("%d %s", r.status, r.body)
	}
}

// A wrong DATABASE_URL must not take the emergency chat down with it.
func TestUnreachableDatabaseDoesNotStopTheApp(t *testing.T) {
	broken := database.Open("postgres://u:p@127.0.0.1:1/none?connect_timeout=1", logging.Discard())
	broken.TryInit(t.Context())
	e := newEnv(t, func(s *setup) { s.db = broken })
	c := e.client()

	health := c.get("/health").m()
	if health["status"] != "ok" || health["database_ready"] != false || health["database"] != "postgres" {
		t.Fatalf("health %v", health)
	}
	// Guests can still chat and request a professional.
	if r := c.post("/api/v1/chat", say("Burn")); r.status != http.StatusOK {
		t.Fatalf("chat: %d %s", r.status, r.body)
	}
	if r := c.post("/api/v1/consultations", obj{"patient_name": "Ram"}); r.status != http.StatusCreated {
		t.Fatalf("consultation: %d %s", r.status, r.body)
	}
	// Signing in says why it can't, instead of hanging or crashing.
	r := c.post("/api/v1/auth/dev-login", obj{"email": "x@example.com", "name": "X"})
	if r.status != http.StatusServiceUnavailable || !strings.Contains(r.detail(), "unavailable") {
		t.Fatalf("dev login: %d %s", r.status, r.body)
	}
	if me := c.get("/api/v1/auth/me"); me.status != http.StatusOK || me.m()["user"] != nil {
		t.Fatalf("signed-out me: %d %s", me.status, me.body)
	}
}

func TestFrontendIsServedFromTheSameOrigin(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		path := filepath.Join(dir, filepath.FromSlash(name))
		_ = os.MkdirAll(filepath.Dir(path), 0o755)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("index.html", "<h1>home</h1>")
	write("patient/index.html", "<h1>patient</h1>")
	write("manifest.webmanifest", "{}")
	write("_next/static/chunks/app.js", "console.log(1)")
	write("404.html", "<h1>lost</h1>")
	c := newEnv(t, withConfig(func(cfg *config.Config) { cfg.StaticDir = dir })).client()

	if r := c.get("/"); !strings.Contains(string(r.body), "home") || r.header.Get("Cache-Control") != "no-cache" {
		t.Fatalf("home: %d %s %v", r.status, r.body, r.header)
	}
	if r := c.get("/patient/"); !strings.Contains(string(r.body), "patient") {
		t.Fatalf("patient: %d %s", r.status, r.body)
	}
	if r := c.get("/patient?x=1"); r.status != http.StatusTemporaryRedirect || r.header.Get("Location") != "/patient/?x=1" {
		t.Fatalf("a directory without its slash should redirect: %d %v", r.status, r.header)
	}
	if loc := c.get("//patient").header.Get("Location"); !strings.HasPrefix(loc, "/") || strings.HasPrefix(loc, "//") {
		t.Fatalf("redirects must stay on this site: %q", loc)
	}
	if ct := c.get("/manifest.webmanifest").header.Get("Content-Type"); !strings.HasPrefix(ct, "application/manifest+json") {
		t.Fatalf("manifest type %q", ct)
	}
	if cache := c.get("/_next/static/chunks/app.js").header.Get("Cache-Control"); !strings.Contains(cache, "immutable") {
		t.Fatalf("hashed assets should be cached: %q", cache)
	}
	if r := c.get("/no/such/page/"); r.status != http.StatusNotFound || !strings.Contains(string(r.body), "lost") {
		t.Fatalf("404 page: %d %s", r.status, r.body)
	}
	if r := c.get("/../../etc/passwd"); r.status == http.StatusOK {
		t.Fatalf("path traversal: %d %s", r.status, r.body)
	}
	// API routes still win over the frontend.
	if r := c.get("/api/v1/health"); r.m()["status"] != "ok" {
		t.Fatalf("health: %s", r.body)
	}
	if r := c.get("/api/v1/nope"); r.status != http.StatusNotFound || r.detail() != "Not Found" {
		t.Fatalf("unknown API path: %d %s", r.status, r.body)
	}
	if r := c.post("/", nil); r.status != http.StatusMethodNotAllowed {
		t.Fatalf("POST to a page: %d", r.status)
	}
}

func TestCORSForTheDevelopmentFrontend(t *testing.T) {
	c := newEnv(t).client()
	r := c.request(http.MethodOptions, "/api/v1/chat", nil, "",
		"Origin", "http://localhost:3000",
		"Access-Control-Request-Method", "POST",
		"Access-Control-Request-Headers", "content-type")
	if r.status != http.StatusOK || r.header.Get("Access-Control-Allow-Origin") != "http://localhost:3000" ||
		r.header.Get("Access-Control-Allow-Credentials") != "true" ||
		r.header.Get("Access-Control-Allow-Headers") != "content-type" {
		t.Fatalf("preflight: %d %v", r.status, r.header)
	}
	if r := c.get("/api/v1/health", "Origin", "http://localhost:3000"); r.header.Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatalf("simple request: %v", r.header)
	}
	r = c.request(http.MethodOptions, "/api/v1/chat", nil, "",
		"Origin", "https://evil.example", "Access-Control-Request-Method", "POST")
	if r.status != http.StatusBadRequest || r.header.Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("other origins: %d %v", r.status, r.header)
	}
}

func TestClientIPAndScheme(t *testing.T) {
	r, _ := http.NewRequest(http.MethodGet, "http://x/", nil)
	r.RemoteAddr = "10.0.0.1:5000"
	if clientIP(r) != "10.0.0.1" || scheme(r) != "http" {
		t.Fatal("direct request")
	}
	r.Header.Set("X-Forwarded-For", "203.0.113.7, 10.1.1.1")
	r.Header.Set("X-Forwarded-Proto", "https")
	if clientIP(r) != "203.0.113.7" || scheme(r) != "https" {
		t.Fatalf("behind a proxy: %s %s", clientIP(r), scheme(r))
	}
}
