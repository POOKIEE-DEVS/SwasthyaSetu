package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

// FakeIDToken builds an unsigned ID token with Google's usual claims.
func FakeIDToken(overrides map[string]any) string {
	claims := map[string]any{
		"iss":            "https://accounts.google.com",
		"aud":            "client-123",
		"exp":            float64(time.Now().Add(10 * time.Minute).Unix()),
		"sub":            "google-sub-1",
		"email":          "Sita@Example.com",
		"email_verified": true,
		"name":           "Sita Sharma",
		"picture":        "https://example.com/p.png",
	}
	for k, v := range overrides {
		claims[k] = v
	}
	payload, _ := json.Marshal(claims)
	return "header." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
}

func TestIDTokenClaimsAreChecked(t *testing.T) {
	profile, err := ProfileFromIDToken(FakeIDToken(nil), "client-123", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if profile.Email != "sita@example.com" || profile.Name != "Sita Sharma" || profile.Sub != "google-sub-1" ||
		*profile.Picture != "https://example.com/p.png" {
		t.Fatalf("profile %+v", profile)
	}

	for _, overrides := range []map[string]any{
		{"iss": "https://evil.example"},
		{"aud": "someone-else"},
		{"exp": float64(time.Now().Add(-10 * time.Second).Unix())},
		{"email_verified": false},
		{"email_verified": "true"}, // must be a real boolean
		{"email": ""},
	} {
		_, err := ProfileFromIDToken(FakeIDToken(overrides), "client-123", time.Now())
		var authErr *Error
		if !errors.As(err, &authErr) {
			t.Errorf("%v: want a sign-in error, got %v", overrides, err)
		}
	}
	for _, garbage := range []string{"", "no-dots", "a.!!!.c", "a." + base64.RawURLEncoding.EncodeToString([]byte("[1]")) + ".c"} {
		if _, err := ProfileFromIDToken(garbage, "client-123", time.Now()); err == nil {
			t.Errorf("%q should be refused", garbage)
		}
	}
}

func TestNameFallsBackToEmailAndIsCapped(t *testing.T) {
	p, _ := ProfileFromIDToken(FakeIDToken(map[string]any{"name": nil, "picture": nil}), "client-123", time.Now())
	if p.Name != "Sita@Example.com" || p.Picture != nil {
		t.Fatalf("profile %+v", p)
	}
	long := ""
	for range 130 {
		long += "न"
	}
	p, _ = ProfileFromIDToken(FakeIDToken(map[string]any{"name": long}), "client-123", time.Now())
	if len([]rune(p.Name)) != 120 {
		t.Fatalf("name has %d characters", len([]rune(p.Name)))
	}
}

func TestNextIsSameSiteOnly(t *testing.T) {
	for value, want := range map[string]string{
		"/doctor/":             "/doctor/",
		"https://evil.example": "/",
		"//evil.example":       "/",
		`/\evil.example`:       "/",
		"":                     "/",
	} {
		if got := SafeNext(value); got != want {
			t.Errorf("SafeNext(%q) = %q, want %q", value, got, want)
		}
	}
}

func TestLoginURLAndPendingState(t *testing.T) {
	g := NewGoogle("client-123", "secret")
	consent, state := g.LoginURL("https://app.example/api/v1/auth/google/callback", "//evil")
	u, _ := url.Parse(consent)
	q := u.Query()
	if u.Host != "accounts.google.com" || q.Get("state") != state || q.Get("code_challenge_method") != "S256" ||
		q.Get("scope") != "openid email profile" || q.Get("prompt") != "select_account" {
		t.Fatalf("consent URL %s", consent)
	}

	if _, err := g.TakePending(state, "other"); err == nil {
		t.Fatal("a state from another browser must be refused")
	}
	p, err := g.TakePending(state, state)
	if err != nil || p.Next != "/" {
		t.Fatalf("pending %+v, err %v", p, err)
	}
	sum := sha256.Sum256([]byte(p.Verifier))
	if q.Get("code_challenge") != base64.RawURLEncoding.EncodeToString(sum[:]) {
		t.Fatal("the challenge must be the verifier's S256 hash")
	}
	if _, err := g.TakePending(state, state); err == nil {
		t.Fatal("a state works once")
	}
}

func TestPendingSignInsExpireAndAreCapped(t *testing.T) {
	g := NewGoogle("c", "s")
	now := time.Now()
	g.now = func() time.Time { return now }
	_, old := g.LoginURL("r", "/")
	now = now.Add(11 * time.Minute)
	if _, err := g.TakePending(old, old); err == nil {
		t.Fatal("a sign-in older than 10 minutes expires")
	}
	var first string
	for i := range maxPending + 5 {
		_, state := g.LoginURL("r", "/")
		if i == 0 {
			first = state
		}
	}
	if len(g.pending) != maxPending {
		t.Fatalf("%d pending", len(g.pending))
	}
	if _, ok := g.pending[first]; ok {
		t.Fatal("the oldest should be dropped first")
	}
}

func TestExchange(t *testing.T) {
	var form url.Values
	status := http.StatusOK
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		form = r.PostForm
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"id_token": "tok"}`))
	}))
	defer srv.Close()
	g := NewGoogle("client-123", "secret")
	g.TokenURL = srv.URL

	token, err := g.Exchange(context.Background(), "abc", "verifier", "https://app/cb")
	if err != nil || token != "tok" {
		t.Fatalf("token %q, err %v", token, err)
	}
	if form.Get("code") != "abc" || form.Get("code_verifier") != "verifier" || form.Get("client_secret") != "secret" ||
		form.Get("grant_type") != "authorization_code" || form.Get("redirect_uri") != "https://app/cb" {
		t.Fatalf("form %v", form)
	}
	status = http.StatusBadRequest
	if _, err := g.Exchange(context.Background(), "abc", "v", "r"); err == nil || err.Error() != "Google sign-in failed. Please try again." {
		t.Fatalf("err %v", err)
	}
}

func TestSessionTokens(t *testing.T) {
	a, hashA := NewSessionToken()
	b, _ := NewSessionToken()
	if a == b || len(a) != 43 || len(hashA) != 64 || HashToken(a) != hashA {
		t.Fatalf("tokens %q %q hash %q", a, b, hashA)
	}
}
