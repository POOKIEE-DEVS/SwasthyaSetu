package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"unicode/utf8"

	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/ai"
	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/auth"
	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/database"
	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/verify"
)

// Errors are JSON objects with a "detail" message, which the frontend shows
// as it is.
type detail struct {
	Detail string `json:"detail"`
}

const (
	msgDatabaseDown = "Accounts are unavailable right now. Please try again in a minute."
	msgInternal     = "Something went wrong on our side. Please try again."
	jsonBodyLimit   = 2 << 20
)

// httpError is an error with a status and a message for the user.
type httpError struct {
	status int
	detail string
}

func (e *httpError) Error() string { return e.detail }

func fail(status int, message string) error { return &httpError{status, message} }

// invalid is a 422: the request is understood but its content is wrong.
func invalid(message string) error { return fail(http.StatusUnprocessableEntity, message) }

// handlerFunc is an HTTP handler that may return an error, written out by
// Server.handle.
type handlerFunc func(w http.ResponseWriter, r *http.Request) error

func (s *Server) handle(h handlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			s.writeError(w, r, err)
		}
	}
}

func (s *Server) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var (
		he *httpError
		me *ai.ModelError
		ve *verify.Error
		ae *auth.Error
	)
	switch {
	case errors.As(err, &he):
		writeJSON(w, he.status, detail{he.detail})
	case errors.As(err, &me):
		writeJSON(w, http.StatusServiceUnavailable, detail{me.Message})
	case errors.As(err, &ve):
		writeJSON(w, http.StatusUnprocessableEntity, detail{ve.Message})
	case errors.As(err, &ae):
		writeJSON(w, http.StatusBadRequest, detail{ae.Message})
	case errors.Is(err, database.ErrUnavailable):
		writeJSON(w, http.StatusServiceUnavailable, detail{msgDatabaseDown})
	case errors.Is(err, context.Canceled) && r.Context().Err() != nil:
		// The browser went away; nobody is left to answer.
	default:
		s.log.Error("request failed", "method", r.Method, "path", r.URL.Path, "error", err)
		writeJSON(w, http.StatusInternalServerError, detail{msgInternal})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

// decodeJSON reads a JSON body into dst.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, jsonBodyLimit)).Decode(dst); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return fail(http.StatusRequestEntityTooLarge, "The request is too large.")
		}
		return invalid("The request could not be read. Please try again.")
	}
	return nil
}

// chars counts characters, not bytes: "नमस्ते" is 6, not 18.
func chars(s string) int { return utf8.RuneCountInString(s) }
