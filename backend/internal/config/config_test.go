package config

import (
	"bufio"
	"slices"
	"strings"
	"testing"
	"time"
)

func load(t *testing.T, env map[string]string) *Config {
	t.Helper()
	c, err := FromLookup(func(key string) (string, bool) {
		v, ok := env[key]
		return v, ok
	})
	if err != nil {
		t.Fatalf("FromLookup: %v", err)
	}
	return c
}

func TestDefaults(t *testing.T) {
	c := load(t, nil)
	if c.Environment != "development" || c.Port != "8000" || c.StaticDir != "static" {
		t.Errorf("unexpected defaults: %+v", c)
	}
	if c.ModelTimeout != 120*time.Second || c.ChatMaxHistoryMessages != 8 ||
		c.ChatMaxMessageChars != 2000 || c.ChatRateLimitPerMinute != 15 {
		t.Errorf("unexpected chat defaults: %+v", c)
	}
	if c.DatabaseKind() != "sqlite" || c.DevLoginEnabled() || c.ModelConfigured() {
		t.Errorf("unexpected defaults: %+v", c)
	}
	if !slices.Equal(c.CORSOrigins, []string{"http://localhost:3000"}) {
		t.Errorf("CORSOrigins = %q", c.CORSOrigins)
	}
}

// Regression from the Python version: plain comma-separated values once
// crashed startup.
func TestListValuesParse(t *testing.T) {
	c := load(t, map[string]string{
		"CORS_ORIGINS": "http://localhost:3000, https://demo.example",
		"TURN_URLS":    "turn:turn.example:3478",
		"STUN_URLS":    "stun:a:3478,stun:b:3478",
	})
	if !slices.Equal(c.CORSOrigins, []string{"http://localhost:3000", "https://demo.example"}) {
		t.Errorf("CORSOrigins = %q", c.CORSOrigins)
	}
	if !slices.Equal(c.TURNURLs, []string{"turn:turn.example:3478"}) {
		t.Errorf("TURNURLs = %q", c.TURNURLs)
	}
	if !slices.Equal(c.STUNURLs, []string{"stun:a:3478", "stun:b:3478"}) {
		t.Errorf("STUNURLs = %q", c.STUNURLs)
	}
}

func TestEmptyListValues(t *testing.T) {
	c := load(t, map[string]string{"CORS_ORIGINS": "", "TURN_URLS": ""})
	if len(c.CORSOrigins) != 0 || len(c.TURNURLs) != 0 {
		t.Errorf("want empty lists, got %q and %q", c.CORSOrigins, c.TURNURLs)
	}
}

func TestDatabaseKind(t *testing.T) {
	for raw, want := range map[string]string{
		"postgres://u:p@host/db?sslmode=require": "postgres",
		"postgresql://u:p@host/db":               "postgres",
		// A pasted value may carry stray spaces.
		" postgresql://u:p@host/db ": "postgres",
		" sqlite:///./local.db ":     "sqlite",
	} {
		c := load(t, map[string]string{"DATABASE_URL": raw})
		if c.DatabaseKind() != want {
			t.Errorf("%q: kind %q, want %q", raw, c.DatabaseKind(), want)
		}
		if c.DatabaseURL != strings.TrimSpace(raw) {
			t.Errorf("%q: not trimmed: %q", raw, c.DatabaseURL)
		}
	}
}

func TestDevLoginIsNeverOnInProduction(t *testing.T) {
	if !load(t, map[string]string{"DEV_LOGIN": "true"}).DevLoginEnabled() {
		t.Error("dev login should be on in development")
	}
	c := load(t, map[string]string{"DEV_LOGIN": "true", "ENVIRONMENT": "production"})
	if c.DevLoginEnabled() {
		t.Error("dev login must be off in production")
	}
}

func TestAdminEmailsAreCaseInsensitive(t *testing.T) {
	c := load(t, map[string]string{"ADMIN_EMAILS": "Admin@Example.com, second@example.com"})
	if !slices.Equal(c.AdminEmails, []string{"admin@example.com", "second@example.com"}) {
		t.Errorf("AdminEmails = %q", c.AdminEmails)
	}
	if !c.IsAdmin("ADMIN@example.COM") || c.IsAdmin("someone@example.com") {
		t.Error("IsAdmin should compare emails case-insensitively")
	}
}

func TestPublicURLFallsBackToRenderExternalURL(t *testing.T) {
	render := map[string]string{"RENDER_EXTERNAL_URL": "https://swasthyasetu.onrender.com"}
	if got := load(t, render).PublicURL; got != "https://swasthyasetu.onrender.com" {
		t.Errorf("PublicURL = %q", got)
	}
	render["PUBLIC_URL"] = "https://custom.example"
	if got := load(t, render).PublicURL; got != "https://custom.example" {
		t.Errorf("PublicURL = %q", got)
	}
}

func TestInvalidValuesAreReported(t *testing.T) {
	_, err := FromLookup(func(key string) (string, bool) {
		switch key {
		case "ENVIRONMENT":
			return "staging", true
		case "DEV_LOGIN":
			return "maybe", true
		case "SESSION_DAYS":
			return "thirty", true
		}
		return "", false
	})
	if err == nil {
		t.Fatal("want an error")
	}
	for _, name := range []string{"ENVIRONMENT", "DEV_LOGIN", "SESSION_DAYS"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error %q does not name %s", err, name)
		}
	}
}

func TestModelTimeoutAcceptsFractions(t *testing.T) {
	c := load(t, map[string]string{"MODEL_TIMEOUT_SECONDS": "1.5"})
	if c.ModelTimeout != 1500*time.Millisecond {
		t.Errorf("ModelTimeout = %v", c.ModelTimeout)
	}
}

func TestParseDotEnv(t *testing.T) {
	text := strings.Join([]string{
		"# comment",
		"",
		"HF_SPACE_ID=user/space",
		"export DEV_LOGIN=true",
		`GOOGLE_CLIENT_ID="quoted value"`,
		"LOG_LEVEL=DEBUG # inline comment",
		"lower_case=works",
		"not a pair",
	}, "\n")
	values, err := parseDotEnv(bufio.NewScanner(strings.NewReader(text)))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"HF_SPACE_ID":      "user/space",
		"DEV_LOGIN":        "true",
		"GOOGLE_CLIENT_ID": "quoted value",
		"LOG_LEVEL":        "DEBUG",
		"LOWER_CASE":       "works",
	}
	for k, v := range want {
		if values[k] != v {
			t.Errorf("%s = %q, want %q", k, values[k], v)
		}
	}
	if len(values) != len(want) {
		t.Errorf("got %d values: %v", len(values), values)
	}
}
