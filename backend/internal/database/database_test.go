package database

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/logging"
)

func TestSQLiteSetupIsIdempotent(t *testing.T) {
	ctx := context.Background()
	db := Open("sqlite://", logging.Discard())
	defer db.Close()
	if db.Kind() != "sqlite" {
		t.Fatalf("kind = %q", db.Kind())
	}
	if err := db.Init(ctx); err != nil {
		t.Fatal(err)
	}
	db.SetReady(false)
	if err := db.Init(ctx); err != nil { // IF NOT EXISTS: safe to repeat
		t.Fatal(err)
	}
	var tables int
	err := db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type = 'table'`).Scan(&tables)
	if err != nil || tables != 8 {
		t.Fatalf("tables = %d, err = %v", tables, err)
	}
}

func TestPlaceholdersBindByNumber(t *testing.T) {
	ctx := context.Background()
	db := Open("sqlite://", logging.Discard())
	defer db.Close()
	if err := db.Init(ctx); err != nil {
		t.Fatal(err)
	}
	created := time.Date(2026, 10, 4, 9, 30, 15, 123456000, time.UTC)
	// $2 before $1: SQLite must still bind each value to its own number.
	_, err := db.ExecContext(ctx,
		`INSERT INTO users (name, email, created_at) VALUES ($2, $1, $3)`,
		"sita@example.com", "Sita", created)
	if err != nil {
		t.Fatal(err)
	}
	var email, name string
	var got time.Time
	err = db.QueryRowContext(ctx, `SELECT email, name, created_at FROM users`).Scan(&email, &name, &got)
	if err != nil {
		t.Fatal(err)
	}
	if email != "sita@example.com" || name != "Sita" {
		t.Errorf("email %q, name %q", email, name)
	}
	if !got.Equal(created) {
		t.Errorf("created_at = %v, want %v", got, created)
	}
}

func TestForeignKeysAreEnforced(t *testing.T) {
	ctx := context.Background()
	db := Open("sqlite:///:memory:", logging.Discard())
	defer db.Close()
	if err := db.Init(ctx); err != nil {
		t.Fatal(err)
	}
	_, err := db.ExecContext(ctx,
		`INSERT INTO sessions (token_hash, user_id, created_at, expires_at) VALUES ('h', 99, $1, $1)`,
		time.Now().UTC())
	if err == nil {
		t.Fatal("a session for a missing user should be refused")
	}
}

func TestUnusableURLLeavesTheDatabaseUnavailable(t *testing.T) {
	ctx := context.Background()
	for _, url := range []string{"mysql://nope", "postgres://u:p@[bad"} {
		db := Open(url, logging.Discard())
		if db.TryInit(ctx) || db.Ready() {
			t.Errorf("%q: should not be ready", url)
		}
		if err := db.EnsureReady(ctx); !errors.Is(err, ErrUnavailable) {
			t.Errorf("%q: EnsureReady = %v", url, err)
		}
		if _, err := db.ExecContext(ctx, "SELECT 1"); err == nil {
			t.Errorf("%q: queries should fail", url)
		}
		_ = db.Close()
	}
}

func TestEnsureReadyRetriesAtMostEveryInterval(t *testing.T) {
	ctx := context.Background()
	db := Open("sqlite://", logging.Discard())
	defer db.Close()
	db.lastAttempt.Store(time.Now().UnixNano())
	if err := db.EnsureReady(ctx); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("a retry right after an attempt should wait: %v", err)
	}
	db.lastAttempt.Store(time.Now().Add(-RetryInterval - time.Second).UnixNano())
	if err := db.EnsureReady(ctx); err != nil {
		t.Fatalf("the retry should set the database up: %v", err)
	}
}

func TestSchemaMatchesEachDialect(t *testing.T) {
	pg := schema("postgres")
	lite := schema("sqlite")
	if len(pg) != len(lite) || len(pg) != 19 {
		t.Fatalf("statements: postgres %d, sqlite %d", len(pg), len(lite))
	}
	if pg[0][:33] != "CREATE TABLE IF NOT EXISTS users " {
		t.Errorf("first statement: %q", pg[0])
	}
}
