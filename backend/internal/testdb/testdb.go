// Package testdb gives each test a fresh, empty database: in-memory SQLite,
// or, when TEST_DATABASE_URL is set, a throwaway schema in that Postgres
// database (dropped afterwards). CI runs the suite both ways.
package testdb

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"strings"
	"testing"

	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/database"
	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/logging"
)

// Open returns a ready database that is removed when the test ends.
func Open(t testing.TB) *database.DB {
	t.Helper()
	ctx := context.Background()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		db := database.Open("sqlite://", logging.Discard())
		t.Cleanup(func() { _ = db.Close() })
		if err := db.Init(ctx); err != nil {
			t.Fatalf("set up SQLite: %v", err)
		}
		return db
	}

	admin := database.Open(url, logging.Discard())
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	schema := "test_" + hex.EncodeToString(b)
	if _, err := admin.ExecContext(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	sep := "?"
	if strings.Contains(url, "?") {
		sep = "&"
	}
	// Unknown connection-string parameters are sent to the server as
	// settings, so this scopes every query to the new schema.
	db := database.Open(url+sep+"search_path="+schema, logging.Discard())
	t.Cleanup(func() {
		_ = db.Close()
		_, _ = admin.ExecContext(context.Background(), `DROP SCHEMA `+schema+` CASCADE`)
		_ = admin.Close()
	})
	if err := db.Init(ctx); err != nil {
		t.Fatalf("set up Postgres: %v", err)
	}
	return db
}
