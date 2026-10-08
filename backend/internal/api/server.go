// Package api is the HTTP server: the REST API under /api/v1, the
// WebSockets under /ws, and the built frontend at "/", all from one origin.
//
// One origin means no CORS and no build-time API URL in production. The
// server must run as a single process: the queue and the call rooms live
// in its memory.
package api

import (
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/ai"
	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/auth"
	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/config"
	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/consult"
	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/database"
	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/ratelimit"
	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/realtime"
	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/store"
)

// Version is reported by /health.
const Version = "0.3.0"

// Server holds everything the handlers share.
type Server struct {
	cfg      *config.Config
	log      *slog.Logger
	db       *database.DB
	store    *store.Store
	model    ai.Generator
	consults *consult.Registry
	hub      *realtime.Hub
	ice      *realtime.ICE
	limiter  *ratelimit.Limiter
	google   *auth.Google
	web      *staticSite // nil when there is no built frontend
	started  time.Time
	// Hosts whose pages may open our WebSockets besides the request's own
	// host: the public URL (in case a proxy changes the Host header) and
	// the CORS origins (for `next dev`).
	wsOrigins []string
}

// New wires a server. The model is usually ai.NewMedGemma; tests pass a
// fake.
func New(cfg *config.Config, db *database.DB, model ai.Generator, log *slog.Logger) *Server {
	s := &Server{
		cfg:      cfg,
		log:      log,
		db:       db,
		store:    store.New(db),
		model:    model,
		consults: consult.NewRegistry(cfg.ConsultationTTL),
		hub:      realtime.NewHub(log),
		ice:      realtime.NewICE(cfg, log),
		limiter:  ratelimit.New(cfg.ChatRateLimitPerMinute, time.Minute),
		google:   auth.NewGoogle(cfg.GoogleClientID, cfg.GoogleClientSecret),
		started:  time.Now(),
	}
	for _, origin := range append([]string{cfg.PublicURL}, cfg.CORSOrigins...) {
		if u, err := url.Parse(origin); err == nil && u.Host != "" {
			s.wsOrigins = append(s.wsOrigins, u.Host)
		}
	}
	if web, err := newStatic(cfg.StaticDir); err == nil {
		s.web = web
	}
	return s
}

// ServingFrontend reports whether STATIC_DIR held a built frontend.
func (s *Server) ServingFrontend() bool { return s.web != nil }

// Google is the OAuth flow (tests point it at a fake token endpoint).
func (s *Server) Google() *auth.Google { return s.google }

// Shutdown closes every WebSocket, so browsers reconnect to the next
// instance.
func (s *Server) Shutdown() { s.hub.CloseAll() }

// Close releases the frontend directory.
func (s *Server) Close() error {
	if s.web == nil {
		return nil
	}
	return s.web.root.Close()
}

// Handler returns the HTTP handler with every route.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	h := s.handle

	// /health at the root too, for the hosting platform's health check.
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /api/v1/health", s.health)

	mux.HandleFunc("POST /api/v1/chat", h(s.chat))
	mux.HandleFunc("GET /api/v1/chats", h(s.listChats))
	mux.HandleFunc("POST /api/v1/chats", h(s.saveChat))
	mux.HandleFunc("GET /api/v1/chats/{id}", h(s.openChat))
	mux.HandleFunc("DELETE /api/v1/chats/{id}", h(s.deleteChat))

	mux.HandleFunc("POST /api/v1/consultations", h(s.requestProfessional))
	mux.HandleFunc("GET /api/v1/consultations", h(s.waitingPatients))
	mux.HandleFunc("GET /api/v1/consultations/helped", h(s.peopleHelped))
	mux.HandleFunc("POST /api/v1/consultations/{id}/accept", h(s.accept))
	mux.HandleFunc("POST /api/v1/consultations/{id}/end", h(s.end))

	mux.HandleFunc("GET /api/v1/auth/me", h(s.me))
	mux.HandleFunc("GET /api/v1/auth/google/login", h(s.googleLogin))
	mux.HandleFunc("GET /api/v1/auth/google/callback", s.googleCallback)
	mux.HandleFunc("POST /api/v1/auth/logout", s.logout)
	mux.HandleFunc("POST /api/v1/auth/role", h(s.chooseRole))
	mux.HandleFunc("POST /api/v1/auth/dev-login", h(s.devLogin))

	mux.HandleFunc("GET /api/v1/applications/me", h(s.myApplication))
	mux.HandleFunc("POST /api/v1/applications", h(s.submitApplication))

	mux.HandleFunc("GET /api/v1/admin/applications", h(s.listApplications))
	mux.HandleFunc("GET /api/v1/admin/documents/{id}", h(s.document))
	mux.HandleFunc("POST /api/v1/admin/applications/{id}/approve", h(s.approve))
	mux.HandleFunc("POST /api/v1/admin/applications/{id}/reject", h(s.reject))

	mux.HandleFunc("GET /ws/doctors", s.queueSocket)
	mux.HandleFunc("GET /ws/consultations/{id}", s.callSocket)

	notFound := func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, detail{"Not Found"})
	}
	mux.HandleFunc("/api/", notFound)
	mux.HandleFunc("/ws/", notFound)
	// Everything else is the frontend. The API and WebSocket routes above
	// are more specific, so they always win.
	if s.web != nil {
		mux.Handle("/", s.web)
	} else {
		mux.HandleFunc("/", notFound)
	}

	var handler http.Handler = mux
	if len(s.cfg.CORSOrigins) > 0 {
		handler = cors(s.cfg.CORSOrigins, handler)
	}
	return s.observe(handler)
}

type healthResponse struct {
	Status        string  `json:"status"`
	Version       string  `json:"version"`
	UptimeSeconds float64 `json:"uptime_seconds"`
	// Configuration presence only. Neither is probed live: probing would
	// wake a sleeping GPU Space on every health check.
	ModelConfigured bool `json:"model_configured"`
	TURNConfigured  bool `json:"turn_configured"`
	// "sqlite" in production means DATABASE_URL is missing: accounts and
	// approvals would be lost on the next restart.
	Database string `json:"database"`
	// false: DATABASE_URL is wrong or the database is down. Chat and patient
	// calls still work; sign-in, verification and history do not.
	DatabaseReady          bool `json:"database_ready"`
	GoogleSignInConfigured bool `json:"google_sign_in_configured"`
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{
		Status:                 "ok",
		Version:                Version,
		UptimeSeconds:          math.Round(time.Since(s.started).Seconds()*1000) / 1000,
		ModelConfigured:        s.cfg.ModelConfigured(),
		TURNConfigured:         s.cfg.TURNConfigured(),
		Database:               s.db.Kind(),
		DatabaseReady:          s.db.Ready(),
		GoogleSignInConfigured: s.cfg.GoogleConfigured(),
	})
}

// redirectURI is where Google sends the browser back. Behind Render's
// proxy, X-Forwarded-Proto makes the scheme https, so a URL derived from
// the request matches what Google has registered; PUBLIC_URL (or Render's
// RENDER_EXTERNAL_URL) wins when set.
func (s *Server) redirectURI(r *http.Request) string {
	base := strings.TrimRight(strings.TrimSpace(s.cfg.PublicURL), "/")
	if base == "" {
		base = scheme(r) + "://" + r.Host
	}
	return base + "/api/v1/auth/google/callback"
}
