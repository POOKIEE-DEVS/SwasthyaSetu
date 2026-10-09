// Package store holds every SQL query: accounts and sessions, professional
// applications with their documents and audit log, saved chats, and help
// records. Handlers never write SQL themselves.
//
// The queries run unchanged on Postgres and SQLite (see package database).
// Times are stored in UTC at microsecond precision, which is what Postgres
// keeps.
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/database"
)

// Store runs the application's queries.
type Store struct {
	db *database.DB
}

// New returns a Store on db.
func New(db *database.DB) *Store { return &Store{db: db} }

// DB is the underlying database handle.
func (s *Store) DB() *database.DB { return s.db }

// Now is the current time as the database stores it.
func Now() time.Time { return time.Now().UTC().Truncate(time.Microsecond) }

// Unix converts a stored time to the Unix seconds clients receive.
func Unix(t time.Time) float64 { return float64(t.UnixMicro()) / 1e6 }

// timeCol scans a timestamp from either driver. Postgres returns time.Time;
// SQLite usually does too, but rows written by other tools (for example
// "2006-01-02 15:04:05.999999", with no zone) may come back as text.
type timeCol struct{ dst *time.Time }

func (c timeCol) Scan(src any) error {
	t, err := parseTime(src)
	if err != nil {
		return err
	}
	if t == nil {
		return errors.New("store: NULL in a NOT NULL time column")
	}
	*c.dst = *t
	return nil
}

// nullTimeCol scans a nullable timestamp.
type nullTimeCol struct{ dst **time.Time }

func (c nullTimeCol) Scan(src any) error {
	t, err := parseTime(src)
	if err != nil {
		return err
	}
	*c.dst = t
	return nil
}

var textTimeFormats = []string{
	"2006-01-02 15:04:05.999999999-07:00",
	"2006-01-02T15:04:05.999999999-07:00",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02T15:04:05.999999999",
}

func parseTime(src any) (*time.Time, error) {
	var text string
	switch v := src.(type) {
	case nil:
		return nil, nil
	case time.Time:
		t := v.UTC()
		return &t, nil
	case string:
		text = v
	case []byte:
		text = string(v)
	default:
		return nil, fmt.Errorf("store: cannot read %T as a time", src)
	}
	for _, layout := range textTimeFormats {
		if t, err := time.Parse(layout, text); err == nil {
			t = t.UTC()
			return &t, nil
		}
	}
	return nil, fmt.Errorf("store: cannot parse time %q", text)
}

// one turns "no rows" into a nil result without an error.
func one[T any](v *T, err error) (*T, error) {
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return v, nil
}
