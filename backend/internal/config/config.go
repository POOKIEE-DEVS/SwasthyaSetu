// Package config loads the runtime settings from environment variables, and
// from backend/.env when it exists (real environment variables win).
//
// The names are the ones set in the Render dashboard and in local .env
// files (see backend/.env.example).
package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds every setting. Build one with Load (or Defaults, in tests).
type Config struct {
	// --- Application ---------------------------------------------------
	Environment string // "development" or "production"
	LogLevel    string
	LogJSON     bool
	Port        string
	// Only needed when the frontend runs on another origin (next dev on
	// :3000). In production the frontend is served by this app.
	CORSOrigins []string
	// Directory holding the Next.js static export, served at "/".
	StaticDir string

	// --- AI model (MedGemma behind a Gradio app) -------------------------
	// A Space id ("user/space") or the URL of any running copy of
	// model-space/app.py, e.g. a Colab "https://….gradio.live" link.
	HFSpaceID    string
	HFToken      string
	ModelTimeout time.Duration
	// MedGemma is not tuned for long conversations: only recent turns go.
	ChatMaxHistoryMessages int
	ChatMaxMessageChars    int
	ChatRateLimitPerMinute int

	// --- WebRTC ICE servers ----------------------------------------------
	STUNURLs []string
	// Option A: Cloudflare TURN mints short-lived credentials per call.
	CloudflareTURNKeyID    string
	CloudflareTURNAPIToken string
	TURNCredentialTTL      int // seconds
	// Option B: any TURN provider with static credentials (e.g. ExpressTURN).
	TURNURLs       []string
	TURNUsername   string
	TURNCredential string

	// --- Consultations -----------------------------------------------------
	// Unanswered or finished requests are dropped from memory after this.
	ConsultationTTL time.Duration

	// --- Database ----------------------------------------------------------
	// Postgres in production (Neon or Supabase). The SQLite default is for
	// local development only: Render's disk is wiped on every restart.
	DatabaseURL string

	// --- Sign-in (Google OAuth) ------------------------------------------
	GoogleClientID     string
	GoogleClientSecret string
	// The site's public base URL. Google redirects back to
	// {PublicURL}/api/v1/auth/google/callback. Render sets
	// RENDER_EXTERNAL_URL itself. Empty: derived from the request.
	PublicURL   string
	AdminEmails []string // lower-cased
	SessionDays int
	// Local development and the smoke test only: sign in without Google.
	// Always off in production (see DevLoginEnabled).
	DevLogin bool

	// --- Professional verification documents -----------------------------
	MaxUploadBytes int64
}

// Defaults returns the settings used when nothing is configured.
func Defaults() *Config {
	return &Config{
		Environment:            "development",
		LogLevel:               "INFO",
		Port:                   "8000",
		CORSOrigins:            []string{"http://localhost:3000"},
		StaticDir:              "static",
		ModelTimeout:           120 * time.Second,
		ChatMaxHistoryMessages: 8,
		ChatMaxMessageChars:    2000,
		ChatRateLimitPerMinute: 15,
		STUNURLs:               []string{"stun:stun.l.google.com:19302", "stun:stun.cloudflare.com:3478"},
		TURNCredentialTTL:      86400,
		ConsultationTTL:        60 * time.Minute,
		DatabaseURL:            "sqlite:///./swasthyasetu.db",
		SessionDays:            30,
		MaxUploadBytes:         5 * 1024 * 1024,
	}
}

// Load reads the environment, falling back to the .env file in the working
// directory, then to the defaults. Invalid values are an error: better to
// fail at startup than to run with a setting silently ignored.
func Load() (*Config, error) {
	dotenv, err := readDotEnv(".env")
	if err != nil {
		return nil, err
	}
	return FromLookup(func(key string) (string, bool) {
		if v, ok := os.LookupEnv(key); ok {
			return v, true
		}
		v, ok := dotenv[key]
		return v, ok
	})
}

// FromLookup builds a Config from any key lookup (the environment, a map in
// tests).
func FromLookup(lookup func(string) (string, bool)) (*Config, error) {
	c := Defaults()
	p := parser{lookup: lookup}

	c.Environment = strings.ToLower(p.str("ENVIRONMENT", c.Environment))
	if c.Environment != "development" && c.Environment != "production" {
		p.fail("ENVIRONMENT", c.Environment, `"development" or "production"`)
	}
	c.LogLevel = p.str("LOG_LEVEL", c.LogLevel)
	c.LogJSON = p.boolean("LOG_JSON", c.LogJSON)
	c.Port = p.str("PORT", c.Port)
	c.CORSOrigins = p.list("CORS_ORIGINS", c.CORSOrigins)
	c.StaticDir = p.str("STATIC_DIR", c.StaticDir)

	c.HFSpaceID = strings.TrimSpace(p.str("HF_SPACE_ID", ""))
	c.HFToken = strings.TrimSpace(p.str("HF_TOKEN", ""))
	c.ModelTimeout = p.seconds("MODEL_TIMEOUT_SECONDS", c.ModelTimeout)
	c.ChatMaxHistoryMessages = p.integer("CHAT_MAX_HISTORY_MESSAGES", c.ChatMaxHistoryMessages)
	c.ChatMaxMessageChars = p.integer("CHAT_MAX_MESSAGE_CHARS", c.ChatMaxMessageChars)
	c.ChatRateLimitPerMinute = p.integer("CHAT_RATE_LIMIT_PER_MINUTE", c.ChatRateLimitPerMinute)

	c.STUNURLs = p.list("STUN_URLS", c.STUNURLs)
	c.CloudflareTURNKeyID = strings.TrimSpace(p.str("CLOUDFLARE_TURN_KEY_ID", ""))
	c.CloudflareTURNAPIToken = strings.TrimSpace(p.str("CLOUDFLARE_TURN_API_TOKEN", ""))
	c.TURNCredentialTTL = p.integer("TURN_CREDENTIAL_TTL_SECONDS", c.TURNCredentialTTL)
	c.TURNURLs = p.list("TURN_URLS", nil)
	c.TURNUsername = p.str("TURN_USERNAME", "")
	c.TURNCredential = p.str("TURN_CREDENTIAL", "")

	c.ConsultationTTL = time.Duration(p.integer("CONSULTATION_TTL_MINUTES", 60)) * time.Minute

	// Pasted values sometimes carry stray spaces.
	c.DatabaseURL = strings.TrimSpace(p.str("DATABASE_URL", c.DatabaseURL))

	c.GoogleClientID = strings.TrimSpace(p.str("GOOGLE_CLIENT_ID", ""))
	c.GoogleClientSecret = strings.TrimSpace(p.str("GOOGLE_CLIENT_SECRET", ""))
	// PUBLIC_URL wins; on Render, RENDER_EXTERNAL_URL fills in by itself.
	if v, ok := lookup("PUBLIC_URL"); ok {
		c.PublicURL = strings.TrimSpace(v)
	} else {
		c.PublicURL = strings.TrimSpace(p.str("RENDER_EXTERNAL_URL", ""))
	}
	for _, email := range p.list("ADMIN_EMAILS", nil) {
		c.AdminEmails = append(c.AdminEmails, strings.ToLower(email))
	}
	c.SessionDays = p.integer("SESSION_DAYS", c.SessionDays)
	c.DevLogin = p.boolean("DEV_LOGIN", false)
	c.MaxUploadBytes = int64(p.integer("MAX_UPLOAD_BYTES", int(c.MaxUploadBytes)))

	if err := errors.Join(p.errs...); err != nil {
		return nil, err
	}
	return c, nil
}

// IsProduction reports ENVIRONMENT=production.
func (c *Config) IsProduction() bool { return c.Environment == "production" }

// DevLoginEnabled is DEV_LOGIN, and never in production.
func (c *Config) DevLoginEnabled() bool { return c.DevLogin && !c.IsProduction() }

// ModelConfigured reports whether a model server is set.
func (c *Config) ModelConfigured() bool { return c.HFSpaceID != "" }

// GoogleConfigured reports whether Google sign-in can be offered.
func (c *Config) GoogleConfigured() bool {
	return c.GoogleClientID != "" && c.GoogleClientSecret != ""
}

// CloudflareTURN reports whether Cloudflare TURN credentials are set.
func (c *Config) CloudflareTURN() bool {
	return c.CloudflareTURNKeyID != "" && c.CloudflareTURNAPIToken != ""
}

// TURNConfigured reports whether any TURN relay is set.
func (c *Config) TURNConfigured() bool { return c.CloudflareTURN() || len(c.TURNURLs) > 0 }

// IsAdmin reports whether an email belongs to ADMIN_EMAILS.
func (c *Config) IsAdmin(email string) bool {
	email = strings.ToLower(email)
	for _, admin := range c.AdminEmails {
		if admin == email {
			return true
		}
	}
	return false
}

// DatabaseKind is "postgres" for postgres:// and postgresql:// URLs, else
// "sqlite".
func (c *Config) DatabaseKind() string {
	if IsPostgresURL(c.DatabaseURL) {
		return "postgres"
	}
	return "sqlite"
}

// IsPostgresURL reports whether a DATABASE_URL names Postgres. Neon and
// Supabase hand out both spellings.
func IsPostgresURL(url string) bool {
	url = strings.TrimSpace(url)
	return strings.HasPrefix(url, "postgres://") || strings.HasPrefix(url, "postgresql://")
}

// SplitCSV splits a comma-separated value, trimming spaces and dropping
// empty entries.
func SplitCSV(value string) []string {
	var out []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

type parser struct {
	lookup func(string) (string, bool)
	errs   []error
}

func (p *parser) fail(key, value, want string) {
	p.errs = append(p.errs, fmt.Errorf("%s=%q: expected %s", key, value, want))
}

func (p *parser) str(key, fallback string) string {
	if v, ok := p.lookup(key); ok {
		return v
	}
	return fallback
}

func (p *parser) list(key string, fallback []string) []string {
	if v, ok := p.lookup(key); ok {
		return SplitCSV(v)
	}
	return fallback
}

func (p *parser) integer(key string, fallback int) int {
	v, ok := p.lookup(key)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		p.fail(key, v, "a whole number")
		return fallback
	}
	return n
}

func (p *parser) seconds(key string, fallback time.Duration) time.Duration {
	v, ok := p.lookup(key)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil || f <= 0 {
		p.fail(key, v, "a positive number of seconds")
		return fallback
	}
	return time.Duration(f * float64(time.Second))
}

func (p *parser) boolean(key string, fallback bool) bool {
	v, ok := p.lookup(key)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "t", "yes", "y", "on":
		return true
	case "0", "false", "f", "no", "n", "off":
		return false
	}
	p.fail(key, v, "true or false")
	return fallback
}

// readDotEnv parses KEY=VALUE lines. Blank lines and # comments are skipped,
// an "export " prefix is allowed, and matching quotes around a value are
// removed. A missing file is not an error.
func readDotEnv(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	defer f.Close()
	return parseDotEnv(bufio.NewScanner(f))
}

func parseDotEnv(scanner *bufio.Scanner) (map[string]string, error) {
	values := map[string]string{}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.ToUpper(strings.TrimSpace(key))
		value = strings.TrimSpace(value)
		if n := len(value); n >= 2 && (value[0] == '"' || value[0] == '\'') && value[n-1] == value[0] {
			value = value[1 : n-1]
		} else if i := strings.Index(value, " #"); i >= 0 {
			value = strings.TrimSpace(value[:i]) // an inline comment
		}
		values[key] = value
	}
	return values, scanner.Err()
}
