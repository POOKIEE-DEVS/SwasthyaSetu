// Package auth is sign-in: Google OAuth (authorization code + PKCE) and the
// session cookie.
//
// Flow:
//  1. LoginURL makes a random state and PKCE verifier, keeps them in memory
//     for 10 minutes, and returns Google's consent-screen URL. The state
//     also goes into a short-lived cookie, which ties the callback to the
//     browser that started it (stops login CSRF).
//  2. Google redirects back with a code and the state. TakePending checks
//     the state; Exchange swaps the code (with the client secret and the
//     verifier) for an ID token; ProfileFromIDToken reads it.
//  3. The caller creates or updates the user and issues a session cookie.
//
// The ID token comes straight from Google's token endpoint over TLS,
// authenticated with our client secret, so Google's guidance is that its
// signature need not be checked again; its issuer, audience, expiry and
// verified-email claims still are.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/store"
)

// Cookie names and paths.
const (
	SessionCookie   = "swasthya_session"
	StateCookie     = "swasthya_oauth_state"
	StateCookiePath = "/api/v1/auth"
	StateCookieAge  = 600 // seconds
)

const (
	pendingTTL  = 10 * time.Minute
	maxPending  = 1000
	googleAuth  = "https://accounts.google.com/o/oauth2/v2/auth"
	googleToken = "https://oauth2.googleapis.com/token"
)

var googleIssuers = []string{"accounts.google.com", "https://accounts.google.com"}

// Error is a failed sign-in. Its message is safe to show to the user.
type Error struct{ Message string }

func (e *Error) Error() string { return e.Message }

// randomToken returns n random bytes, URL-safe base64 encoded.
func randomToken(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b) // never fails
	return base64.RawURLEncoding.EncodeToString(b)
}

// NewSessionToken returns a session cookie value and the hash to store.
// Only the hash is kept, so a database leak doesn't hand out sessions.
func NewSessionToken() (token, hash string) {
	token = randomToken(32)
	return token, HashToken(token)
}

// HashToken is the stored form of a session token.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// SafeNext only allows same-site paths, so the login can't redirect to
// another site.
func SafeNext(value string) string {
	if strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//") && !strings.Contains(value, `\`) {
		return value
	}
	return "/"
}

// Pending is a sign-in that was started and not finished yet.
type Pending struct {
	Verifier string
	Next     string
	created  time.Time
	seq      uint64
}

// Google runs the OAuth flow. Pending sign-ins live in memory: the backend
// runs as one process, and a sign-in that spans a restart is just retried.
type Google struct {
	ClientID     string
	ClientSecret string
	AuthURL      string
	TokenURL     string // tests point it at a fake
	client       *http.Client

	mu      sync.Mutex
	pending map[string]Pending
	seq     uint64
	now     func() time.Time
}

// NewGoogle returns the OAuth flow for a client.
func NewGoogle(clientID, clientSecret string) *Google {
	return &Google{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AuthURL:      googleAuth,
		TokenURL:     googleToken,
		client:       &http.Client{Timeout: 10 * time.Second},
		pending:      map[string]Pending{},
		now:          time.Now,
	}
}

// Configured reports whether Google sign-in can be offered.
func (g *Google) Configured() bool { return g.ClientID != "" && g.ClientSecret != "" }

func (g *Google) purge() {
	cutoff := g.now().Add(-pendingTTL)
	for state, p := range g.pending {
		if p.created.Before(cutoff) {
			delete(g.pending, state)
		}
	}
	for len(g.pending) > maxPending { // drop the oldest
		oldest, first := "", uint64(0)
		for state, p := range g.pending {
			if oldest == "" || p.seq < first {
				oldest, first = state, p.seq
			}
		}
		delete(g.pending, oldest)
	}
}

// LoginURL starts a sign-in and returns Google's consent URL and the state.
func (g *Google) LoginURL(redirectURI, next string) (string, string) {
	state, verifier := randomToken(24), randomToken(48)
	challenge := sha256.Sum256([]byte(verifier))

	g.mu.Lock()
	g.seq++
	g.pending[state] = Pending{Verifier: verifier, Next: SafeNext(next), created: g.now(), seq: g.seq}
	g.purge()
	g.mu.Unlock()

	query := url.Values{
		"client_id":             {g.ClientID},
		"redirect_uri":          {redirectURI},
		"response_type":         {"code"},
		"scope":                 {"openid email profile"},
		"state":                 {state},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(challenge[:])},
		"code_challenge_method": {"S256"},
		"prompt":                {"select_account"},
	}
	return g.AuthURL + "?" + query.Encode(), state
}

// TakePending finishes a pending sign-in: the state must match the cookie
// set when it started.
func (g *Google) TakePending(state, cookieState string) (Pending, error) {
	if cookieState == "" || subtle.ConstantTimeCompare([]byte(state), []byte(cookieState)) != 1 {
		return Pending{}, &Error{"Sign-in expired or was started in another tab. Try again."}
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.purge()
	p, ok := g.pending[state]
	if !ok {
		return Pending{}, &Error{"Sign-in expired. Please try again."}
	}
	delete(g.pending, state)
	return p, nil
}

// Exchange swaps an authorization code for Google's ID token.
func (g *Google) Exchange(ctx context.Context, code, verifier, redirectURI string) (string, error) {
	failed := &Error{"Google sign-in failed. Please try again."}
	form := url.Values{
		"code":          {code},
		"client_id":     {g.ClientID},
		"client_secret": {g.ClientSecret},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
		"code_verifier": {verifier},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", failed
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := g.client.Do(req)
	if err != nil {
		return "", failed
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", failed
	}
	var tokens struct {
		IDToken string `json:"id_token"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&tokens); err != nil {
		return "", failed
	}
	return tokens.IDToken, nil
}

// ProfileFromIDToken checks an ID token's claims and returns the profile.
func ProfileFromIDToken(idToken, clientID string, now time.Time) (store.GoogleProfile, error) {
	failed := &Error{"Google sign-in failed. Please try again."}
	parts := strings.Split(idToken, ".")
	if len(parts) < 2 {
		return store.GoogleProfile{}, failed
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[1], "="))
	if err != nil {
		return store.GoogleProfile{}, failed
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return store.GoogleProfile{}, failed
	}

	iss, _ := claims["iss"].(string)
	if iss != googleIssuers[0] && iss != googleIssuers[1] {
		return store.GoogleProfile{}, &Error{"Google sign-in failed (issuer)."}
	}
	if aud, _ := claims["aud"].(string); aud != clientID || aud == "" {
		return store.GoogleProfile{}, &Error{"Google sign-in failed (audience)."}
	}
	if exp, ok := number(claims["exp"]); !ok || exp < float64(now.UnixNano())/1e9 {
		return store.GoogleProfile{}, &Error{"Google sign-in expired. Please try again."}
	}
	sub, email := text(claims["sub"]), text(claims["email"])
	if sub == "" || email == "" {
		return store.GoogleProfile{}, &Error{"Google did not share an email address."}
	}
	if verified, _ := claims["email_verified"].(bool); !verified {
		return store.GoogleProfile{}, &Error{"Please use a Google account with a verified email."}
	}

	name := text(claims["name"])
	if name == "" {
		name = email
	}
	name = strings.TrimSpace(name)
	if utf8.RuneCountInString(name) > 120 {
		name = string([]rune(name)[:120])
	}
	profile := store.GoogleProfile{Sub: sub, Email: strings.ToLower(email), Name: name}
	if picture, ok := claims["picture"].(string); ok {
		profile.Picture = &picture
	}
	return profile, nil
}

// number reads a JSON number, or a number in a string.
func number(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case string:
		f, err := strconv.ParseFloat(n, 64)
		return f, err == nil
	}
	return 0, false
}

// text reads a JSON string or number as text.
func text(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case nil:
		return ""
	}
	return fmt.Sprint(v)
}
