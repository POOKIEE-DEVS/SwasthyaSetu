package api

import (
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/config"
)

func TestSignedOutMe(t *testing.T) {
	got := newEnv(t).client().get("/api/v1/auth/me").m()
	if !reflect.DeepEqual(got, obj{"user": nil, "google_enabled": false, "dev_login": true}) {
		t.Fatalf("%v", got)
	}
}

func TestDevLoginSessionAndLogout(t *testing.T) {
	c := newEnv(t).client()
	c.signIn("Ram@Example.com", "Ram")
	me := c.get("/api/v1/auth/me").m()["user"].(map[string]any)
	if me["email"] != "ram@example.com" || me["role"] != nil || me["is_admin"] != false || me["verification"] != nil {
		t.Fatalf("%v", me)
	}
	r := c.post("/api/v1/auth/logout", nil)
	if r.status != http.StatusNoContent || !strings.Contains(r.header.Get("Set-Cookie"), "Max-Age=0") {
		t.Fatalf("%d %v", r.status, r.header)
	}
	if c.get("/api/v1/auth/me").m()["user"] != nil {
		t.Fatal("still signed in")
	}
}

func TestSessionCookieAttributes(t *testing.T) {
	c := newEnv(t).client()
	r := c.post("/api/v1/auth/dev-login", obj{"email": "ram@example.com", "name": "Ram"})
	cookie := r.header.Get("Set-Cookie")
	for _, want := range []string{"swasthya_session=", "Path=/", "HttpOnly", "SameSite=Lax", "Max-Age=2592000"} {
		if !strings.Contains(cookie, want) {
			t.Errorf("cookie %q lacks %q", cookie, want)
		}
	}
	if strings.Contains(cookie, "Secure") {
		t.Error("plain http must not get a Secure cookie")
	}
	// Behind Render's proxy the browser uses https.
	r = c.request(http.MethodPost, "/api/v1/auth/dev-login",
		strings.NewReader(`{"email":"ram@example.com","name":"Ram"}`), "application/json",
		"X-Forwarded-Proto", "https")
	if !strings.Contains(r.header.Get("Set-Cookie"), "Secure") {
		t.Error("https needs a Secure cookie")
	}
}

func TestDevLoginIsOffInProduction(t *testing.T) {
	c := newEnv(t, withConfig(func(cfg *config.Config) { cfg.Environment = "production" })).client()
	r := c.post("/api/v1/auth/dev-login", obj{"email": "x@example.com", "name": "X"})
	if r.status != http.StatusNotFound {
		t.Fatalf("%d", r.status)
	}
	if c.get("/api/v1/auth/me").m()["dev_login"] != false {
		t.Fatal("me should say dev login is off")
	}
}

func TestDevLoginValidation(t *testing.T) {
	c := newEnv(t).client()
	for _, body := range []obj{
		{"email": "not-an-email", "name": "X"},
		{"email": "x@example.com", "name": ""},
		{"email": "x@example.com", "name": strings.Repeat("n", 121)},
	} {
		if r := c.post("/api/v1/auth/dev-login", body); r.status != http.StatusUnprocessableEntity {
			t.Errorf("%v: %d", body, r.status)
		}
	}
}

func TestAdminComesFromAdminEmails(t *testing.T) {
	c := newEnv(t).client()
	if c.signIn(strings.ToUpper(adminEmail), "Admin")["is_admin"] != true {
		t.Fatal("not admin")
	}
}

func TestChooseRole(t *testing.T) {
	c := newEnv(t).client()
	if r := c.post("/api/v1/auth/role", obj{"role": "patient"}); r.status != http.StatusUnauthorized {
		t.Fatalf("%d", r.status)
	}
	c.signIn("gita@example.com", "Gita")
	if chosen := c.post("/api/v1/auth/role", obj{"role": "patient"}); chosen.m()["role"] != "patient" {
		t.Fatalf("%s", chosen.body)
	}
	if bad := c.post("/api/v1/auth/role", obj{"role": "admin"}); bad.status != http.StatusUnprocessableEntity {
		t.Fatalf("%d", bad.status)
	}
}

func TestNursesAndParamedicsCanChooseTheirRole(t *testing.T) {
	c := newEnv(t).client()
	for _, role := range []string{"nurse", "paramedic"} {
		c.clearCookies()
		c.signIn(role+"@example.com", "Test User")
		chosen := c.post("/api/v1/auth/role", obj{"role": role})
		if chosen.status != http.StatusOK || chosen.m()["role"] != role {
			t.Fatalf("%d %s", chosen.status, chosen.body)
		}
	}
}

func TestGoogleLoginNeedsConfiguration(t *testing.T) {
	r := newEnv(t).client().get("/api/v1/auth/google/login")
	if r.status != http.StatusNotFound || r.detail() != "Google sign-in is not configured." {
		t.Fatalf("%d %s", r.status, r.body)
	}
}

// startGoogle begins a sign-in and returns the state and the consent URL's
// parameters.
func (c *client) startGoogle(next string) (string, url.Values) {
	c.e.t.Helper()
	r := c.get("/api/v1/auth/google/login?next=" + url.QueryEscape(next))
	if r.status != http.StatusSeeOther {
		c.e.t.Fatalf("login: %d %s", r.status, r.body)
	}
	consent, _ := url.Parse(r.header.Get("Location"))
	if consent.Host != "accounts.google.com" {
		c.e.t.Fatalf("consent URL %s", consent)
	}
	return consent.Query().Get("state"), consent.Query()
}

func TestGoogleRoundTrip(t *testing.T) {
	option, exchanges, attach := withGoogle(t)
	e := newEnv(t, option)
	attach(e)
	c := e.client()

	state, params := c.startGoogle("/doctor/")
	if params.Get("code_challenge_method") != "S256" || params.Get("scope") != "openid email profile" {
		t.Fatalf("%v", params)
	}
	redirectURI := params.Get("redirect_uri")
	if redirectURI != e.http.URL+"/api/v1/auth/google/callback" {
		t.Fatalf("redirect_uri %q", redirectURI)
	}

	back := c.get("/api/v1/auth/google/callback?code=abc&state=" + url.QueryEscape(state))
	// A new user picks a role first, then continues to /doctor/.
	if back.status != http.StatusSeeOther || back.header.Get("Location") != "/account/?next=%2Fdoctor%2F" {
		t.Fatalf("%d %v", back.status, back.header)
	}
	forms := exchanges.all()
	if len(forms) != 1 || forms[0].Get("code") != "abc" || forms[0].Get("redirect_uri") != redirectURI ||
		forms[0].Get("code_verifier") == "" {
		t.Fatalf("%v", forms)
	}
	me := c.get("/api/v1/auth/me").m()
	u := me["user"].(map[string]any)
	if me["google_enabled"] != true || u["email"] != "sita@example.com" || u["name"] != "Sita Sharma" {
		t.Fatalf("%v", me)
	}
}

func TestPublicURLDecidesTheRedirectURI(t *testing.T) {
	option, _, attach := withGoogle(t)
	e := newEnv(t, option, func(s *setup) {
		configure := s.configure
		s.configure = func(cfg *config.Config) { configure(cfg); cfg.PublicURL = "https://swasthya.example/" }
	})
	attach(e)
	_, params := e.client().startGoogle("/")
	if params.Get("redirect_uri") != "https://swasthya.example/api/v1/auth/google/callback" {
		t.Fatalf("%v", params.Get("redirect_uri"))
	}
}

func TestReturningUserGoesStraightToNext(t *testing.T) {
	option, _, attach := withGoogle(t)
	e := newEnv(t, option)
	attach(e)
	c := e.client()
	var back *response
	for range 2 {
		state, _ := c.startGoogle("/")
		back = c.get("/api/v1/auth/google/callback?code=c&state=" + url.QueryEscape(state))
		c.post("/api/v1/auth/role", obj{"role": "patient"})
	}
	if back.header.Get("Location") != "/" {
		t.Fatalf("%v", back.header)
	}
}

func TestCallbackRejectsStateFromAnotherBrowser(t *testing.T) {
	option, exchanges, attach := withGoogle(t)
	e := newEnv(t, option)
	attach(e)
	c := e.client()
	state, _ := c.startGoogle("/")
	c.clearCookies() // the callback arrives without our state cookie

	back := c.get("/api/v1/auth/google/callback?code=c&state=" + url.QueryEscape(state))
	if !strings.HasPrefix(back.header.Get("Location"), "/account/?error=") {
		t.Fatalf("%v", back.header)
	}
	if len(exchanges.all()) != 0 {
		t.Fatal("the code must never be exchanged")
	}
	if c.get("/api/v1/auth/me").m()["user"] != nil {
		t.Fatal("signed in")
	}
}

func TestCancelledConsent(t *testing.T) {
	option, _, attach := withGoogle(t)
	e := newEnv(t, option)
	attach(e)
	c := e.client()
	state, _ := c.startGoogle("/")
	back := c.get("/api/v1/auth/google/callback?error=access_denied&state=" + url.QueryEscape(state))
	location, _ := url.Parse(back.header.Get("Location"))
	if location.Path != "/account/" || location.Query().Get("error") != "Sign-in was cancelled." {
		t.Fatalf("%v", back.header)
	}
}

func TestAdminSkipsRoleSelection(t *testing.T) {
	option, _, attach := withGoogle(t, "sita@example.com")
	e := newEnv(t, option)
	attach(e)
	c := e.client()
	state, _ := c.startGoogle("/admin/")
	back := c.get("/api/v1/auth/google/callback?code=c&state=" + url.QueryEscape(state))
	if back.header.Get("Location") != "/admin/" {
		t.Fatalf("%v", back.header)
	}
}

func TestOpenRedirectsAreRefused(t *testing.T) {
	option, _, attach := withGoogle(t, "sita@example.com")
	e := newEnv(t, option)
	attach(e)
	c := e.client()
	state, _ := c.startGoogle("https://evil.example/")
	back := c.get("/api/v1/auth/google/callback?code=c&state=" + url.QueryEscape(state))
	if back.header.Get("Location") != "/" {
		t.Fatalf("%v", back.header)
	}
}
