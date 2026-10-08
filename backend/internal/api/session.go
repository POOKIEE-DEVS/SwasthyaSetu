package api

import (
	"context"
	"net/http"

	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/auth"
	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/consult"
	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/store"
)

// Who is making a request: the signed-in user, read from the session cookie.
// Every protected endpoint checks this on the server. Hiding a button in the
// browser is never the only guard.

func sessionToken(r *http.Request) string {
	c, err := r.Cookie(auth.SessionCookie)
	if err != nil {
		return ""
	}
	return c.Value
}

// optionalUser is the signed-in user, or nil. Without a session cookie it
// never touches the database.
func (s *Server) optionalUser(r *http.Request) (*store.User, error) {
	return s.userForToken(r.Context(), sessionToken(r))
}

func (s *Server) userForToken(ctx context.Context, token string) (*store.User, error) {
	if token == "" {
		return nil, nil
	}
	if err := s.db.EnsureReady(ctx); err != nil {
		return nil, err
	}
	return s.store.UserForSession(ctx, auth.HashToken(token))
}

// currentUser is the signed-in user, or a 401.
func (s *Server) currentUser(r *http.Request) (*store.User, error) {
	user, err := s.optionalUser(r)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fail(http.StatusUnauthorized, "Please sign in first.")
	}
	return user, nil
}

// requireAdmin is the signed-in admin (an email in ADMIN_EMAILS), or a
// 401/403.
func (s *Server) requireAdmin(r *http.Request) (*store.User, error) {
	user, err := s.currentUser(r)
	if err != nil {
		return nil, err
	}
	if !s.cfg.IsAdmin(user.Email) {
		return nil, fail(http.StatusForbidden, "Admins only.")
	}
	return user, nil
}

// professional is a verified doctor, pharmacist, nurse, paramedic or MBBS
// student.
type professional struct {
	user        *store.User
	application *store.Application
}

// badge is what the patient sees: the name the admin checked against the
// citizenship certificate.
func (p professional) badge() consult.Badge {
	return consult.Badge{Name: p.application.FullName, Role: p.application.Role}
}

// verifiedProfessional returns the user's verification, or nil if they are
// not a verified professional.
func (s *Server) verifiedProfessional(ctx context.Context, user *store.User) (*professional, error) {
	application, err := s.store.ApplicationForUser(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	if !store.IsVerified(user, application) {
		return nil, nil
	}
	return &professional{user: user, application: application}, nil
}

// requireProfessional is the signed-in verified professional, or a 401/403.
func (s *Server) requireProfessional(r *http.Request) (*professional, error) {
	user, err := s.currentUser(r)
	if err != nil {
		return nil, err
	}
	pro, err := s.verifiedProfessional(r.Context(), user)
	if err != nil {
		return nil, err
	}
	if pro == nil {
		return nil, fail(http.StatusForbidden, "Only verified medical professionals can do this.")
	}
	return pro, nil
}

func (s *Server) setSessionCookie(w http.ResponseWriter, r *http.Request, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   s.cfg.SessionDays * 24 * 3600,
		HttpOnly: true,
		Secure:   scheme(r) == "https",
		// Lax: sent on normal navigation, never on cross-site form posts.
		SameSite: http.SameSiteLaxMode,
	})
}

func clearCookie(w http.ResponseWriter, r *http.Request, name, path string) {
	http.SetCookie(w, &http.Cookie{
		Name: name, Value: "", Path: path, MaxAge: -1, HttpOnly: true,
		Secure: scheme(r) == "https", SameSite: http.SameSiteLaxMode,
	})
}
