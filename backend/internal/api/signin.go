package api

import (
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/auth"
	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/store"
)

// Sign in with Google, sign out, who am I, and choosing a role.
//
// Signing in is optional for patients: chat and "talk to a professional"
// work without it. It is required for professionals (to apply and, once
// verified, to take calls) and for the admin.

func (s *Server) me(w http.ResponseWriter, r *http.Request) error {
	user, err := s.optionalUser(r)
	if err != nil {
		return err
	}
	out := meResponse{GoogleEnabled: s.google.Configured(), DevLogin: s.cfg.DevLoginEnabled()}
	if user != nil {
		if out.User, err = s.userPublic(r.Context(), user); err != nil {
			return err
		}
	}
	return writeOK(w, out)
}

func (s *Server) googleLogin(w http.ResponseWriter, r *http.Request) error {
	if !s.google.Configured() {
		return fail(http.StatusNotFound, "Google sign-in is not configured.")
	}
	next := "/"
	if values, ok := r.URL.Query()["next"]; ok {
		next = values[0]
	}
	consent, state := s.google.LoginURL(s.redirectURI(r), next)
	http.SetCookie(w, &http.Cookie{
		Name: auth.StateCookie, Value: state, Path: auth.StateCookiePath, MaxAge: auth.StateCookieAge,
		HttpOnly: true, Secure: scheme(r) == "https", SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, consent, http.StatusSeeOther)
	return nil
}

// googleCallback is where Google sends the browser back. It always answers
// with a redirect: to the next page, to choosing a role, or to the account
// page with an error to show.
func (s *Server) googleCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	clearCookie(w, r, auth.StateCookie, auth.StateCookiePath)
	failed := func(message string) {
		http.Redirect(w, r, "/account/?"+url.Values{"error": {message}}.Encode(), http.StatusSeeOther)
	}

	cookieState := ""
	if c, err := r.Cookie(auth.StateCookie); err == nil {
		cookieState = c.Value
	}
	pending, err := s.google.TakePending(q.Get("state"), cookieState)
	if err == nil && (q.Get("error") != "" || q.Get("code") == "") {
		err = &auth.Error{Message: "Sign-in was cancelled."}
	}
	var profile store.GoogleProfile
	if err == nil {
		var idToken string
		if idToken, err = s.google.Exchange(r.Context(), q.Get("code"), pending.Verifier, s.redirectURI(r)); err == nil {
			profile, err = auth.ProfileFromIDToken(idToken, s.google.ClientID, time.Now())
		}
	}
	var authErr *auth.Error
	if errors.As(err, &authErr) {
		failed(authErr.Message)
		return
	}
	if err != nil {
		s.log.Error("google sign-in failed", "error", err)
		failed("Google sign-in failed. Please try again.")
		return
	}

	user, token, err := s.signInGoogle(r, profile)
	if err != nil {
		s.log.Error("could not sign in", "error", err)
		failed(msgDatabaseDown)
		return
	}
	destination := pending.Next
	// A new account picks a role first; the admin needs none to review.
	if user.Role == nil && !s.cfg.IsAdmin(user.Email) {
		destination = "/account/?" + url.Values{"next": {pending.Next}}.Encode()
	}
	s.setSessionCookie(w, r, token)
	http.Redirect(w, r, destination, http.StatusSeeOther)
}

func (s *Server) signInGoogle(r *http.Request, profile store.GoogleProfile) (*store.User, string, error) {
	ctx := r.Context()
	if err := s.db.EnsureReady(ctx); err != nil {
		return nil, "", err
	}
	user, err := s.store.UpsertGoogleUser(ctx, profile)
	if err != nil {
		return nil, "", err
	}
	token, err := s.createSession(r, user)
	return user, token, err
}

func (s *Server) createSession(r *http.Request, user *store.User) (string, error) {
	token, hash := auth.NewSessionToken()
	expires := time.Now().Add(time.Duration(s.cfg.SessionDays) * 24 * time.Hour)
	return token, s.store.CreateSession(r.Context(), hash, user.ID, expires)
}

// logout always clears the cookie, even if the database can't be reached
// to forget the session: on a shared phone, the next person must not stay
// signed in as the last.
func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if token := sessionToken(r); token != "" {
		err := s.db.EnsureReady(r.Context())
		if err == nil {
			err = s.store.DeleteSession(r.Context(), auth.HashToken(token))
		}
		if err != nil {
			s.log.Warn("session not deleted from the database", "error", err)
		}
	}
	clearCookie(w, r, auth.SessionCookie, "/")
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) chooseRole(w http.ResponseWriter, r *http.Request) error {
	user, err := s.currentUser(r)
	if err != nil {
		return err
	}
	var body struct {
		Role string `json:"role"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		return err
	}
	if !store.IsRole(body.Role) {
		return invalid("Please choose patient, doctor, pharmacist, nurse, paramedic or MBBS student.")
	}
	application, err := s.store.ApplicationForUser(r.Context(), user.ID)
	if err != nil {
		return err
	}
	if application != nil && application.Status == store.StatusApproved && body.Role != application.Role {
		return fail(http.StatusConflict, "Your account is verified for a professional role and can't switch.")
	}
	if err := s.store.SetRole(r.Context(), user.ID, body.Role); err != nil {
		return err
	}
	user.Role = &body.Role
	view, err := s.userPublic(r.Context(), user)
	if err != nil {
		return err
	}
	return writeOK(w, view)
}

var emailRE = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// devLogin signs in without Google, for local development and the smoke
// test. Off unless DEV_LOGIN=true, and always off in production.
func (s *Server) devLogin(w http.ResponseWriter, r *http.Request) error {
	if !s.cfg.DevLoginEnabled() {
		return fail(http.StatusNotFound, "Not found.")
	}
	var body struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		return err
	}
	if !emailRE.MatchString(body.Email) || chars(body.Email) > 320 {
		return invalid("Please enter a valid email address.")
	}
	if chars(body.Name) < 1 || chars(body.Name) > 120 {
		return invalid("Please enter your name (up to 120 characters).")
	}
	if err := s.db.EnsureReady(r.Context()); err != nil {
		return err
	}
	user, err := s.store.UpsertDevUser(r.Context(), body.Email, body.Name)
	if err != nil {
		return err
	}
	token, err := s.createSession(r, user)
	if err != nil {
		return err
	}
	view, err := s.userPublic(r.Context(), user)
	if err != nil {
		return err
	}
	s.setSessionCookie(w, r, token)
	return writeOK(w, view)
}
