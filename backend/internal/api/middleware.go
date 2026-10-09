package api

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"math"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"time"
)

// statusRecorder remembers the status code for the request log.
type statusRecorder struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if !r.wrote {
		r.status, r.wrote = code, true
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if !r.wrote {
		r.status, r.wrote = http.StatusOK, true
	}
	return r.ResponseWriter.Write(b)
}

// Unwrap lets http.ResponseController reach the connection (flushing, and
// the WebSocket upgrade).
func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

// Hijack hands the connection over for a WebSocket.
func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("hijacking not supported")
	}
	r.status, r.wrote = http.StatusSwitchingProtocols, true
	return hj.Hijack()
}

// Flush sends buffered data.
func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func newRequestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// observe gives every response an X-Request-ID (the caller's, or a new one),
// logs each API request with its status and duration, and turns a panic
// into a 500 instead of a dropped connection.
func (s *Server) observe(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" || len(requestID) > 200 {
			requestID = newRequestID()
		}
		w.Header().Set("X-Request-ID", requestID)
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		started := time.Now()

		defer func() {
			if p := recover(); p != nil {
				if p == http.ErrAbortHandler {
					panic(p)
				}
				s.log.Error("panic", "request_id", requestID, "path", r.URL.Path,
					"panic", p, "stack", string(debug.Stack()))
				if !rec.wrote {
					writeJSON(rec, http.StatusInternalServerError, detail{msgInternal})
				}
			}
			if strings.HasPrefix(r.URL.Path, "/api/") {
				ms := float64(time.Since(started).Microseconds()) / 1000
				s.log.Info("request",
					"request_id", requestID,
					"method", r.Method,
					"path", r.URL.Path,
					"status_code", rec.status,
					"duration_ms", math.Round(ms*10)/10,
				)
			}
		}()
		next.ServeHTTP(rec, r)
	})
}

// cors lets the configured origins call the API with cookies; only needed
// when `next dev` serves the frontend on another port. In production the
// frontend comes from this server, so CORS_ORIGINS is empty.
func cors(origins []string, next http.Handler) http.Handler {
	allowed := map[string]bool{}
	for _, o := range origins {
		allowed[strings.TrimRight(o, "/")] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}
		ok := allowed[origin] || allowed["*"]
		h := w.Header()
		h.Add("Vary", "Origin")
		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			if !ok {
				http.Error(w, "Disallowed CORS origin", http.StatusBadRequest)
				return
			}
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Access-Control-Allow-Credentials", "true")
			h.Set("Access-Control-Allow-Methods", "DELETE, GET, HEAD, OPTIONS, PATCH, POST, PUT")
			if requested := r.Header.Get("Access-Control-Request-Headers"); requested != "" {
				h.Set("Access-Control-Allow-Headers", requested)
			}
			h.Set("Access-Control-Max-Age", "600")
			w.WriteHeader(http.StatusOK)
			return
		}
		if ok {
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Access-Control-Allow-Credentials", "true")
		}
		next.ServeHTTP(w, r)
	})
}

// clientIP is the address the rate limit counts. Behind Render's proxy the
// connection comes from the proxy, so the first X-Forwarded-For entry is
// used.
func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		first, _, _ := strings.Cut(forwarded, ",")
		if ip := strings.TrimSpace(first); ip != "" {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// scheme is "https" when the browser used HTTPS, including behind a proxy
// that terminates TLS and says so in X-Forwarded-Proto.
func scheme(r *http.Request) string {
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		first, _, _ := strings.Cut(proto, ",")
		if p := strings.ToLower(strings.TrimSpace(first)); p == "https" || p == "http" {
			return p
		}
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}
