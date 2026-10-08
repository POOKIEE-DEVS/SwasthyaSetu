package store

import (
	"context"
	"strings"
	"time"
)

// Roles a user can hold. Admin access is not a role: see ADMIN_EMAILS.
const (
	RolePatient    = "patient"
	RoleDoctor     = "doctor"
	RolePharmacist = "pharmacist"
	RoleNurse      = "nurse"
	RoleParamedic  = "paramedic"
	RoleStudent    = "student"
)

// ProfessionalRoles are everyone who may answer a patient, once verified.
var ProfessionalRoles = []string{RoleDoctor, RolePharmacist, RoleNurse, RoleParamedic, RoleStudent}

// IsProfessionalRole reports whether role is one of ProfessionalRoles.
func IsProfessionalRole(role string) bool {
	for _, r := range ProfessionalRoles {
		if r == role {
			return true
		}
	}
	return false
}

// IsRole reports whether role is patient or a professional role.
func IsRole(role string) bool { return role == RolePatient || IsProfessionalRole(role) }

// User is an account.
type User struct {
	ID int64
	// Google's stable account id; nil for development logins.
	GoogleSub  *string
	Email      string
	Name       string
	PictureURL *string
	// nil until chosen after the first sign-in.
	Role      *string
	CreatedAt time.Time
}

const userColumns = `id, google_sub, email, name, picture_url, role, created_at`

type rowScanner interface{ Scan(dest ...any) error }

func scanUser(row rowScanner) (*User, error) {
	var u User
	err := row.Scan(&u.ID, &u.GoogleSub, &u.Email, &u.Name, &u.PictureURL, &u.Role, timeCol{&u.CreatedAt})
	return &u, err
}

// UserByID returns a user, or nil if there is none.
func (s *Store) UserByID(ctx context.Context, id int64) (*User, error) {
	return one(scanUser(s.db.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users WHERE id = $1`, id)))
}

// UserByEmail returns a user, or nil if there is none.
func (s *Store) UserByEmail(ctx context.Context, email string) (*User, error) {
	return one(scanUser(s.db.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users WHERE email = $1`, email)))
}

// UserByGoogleSub returns a user, or nil if there is none.
func (s *Store) UserByGoogleSub(ctx context.Context, sub string) (*User, error) {
	return one(scanUser(s.db.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users WHERE google_sub = $1`, sub)))
}

func (s *Store) insertUser(ctx context.Context, u *User) error {
	u.CreatedAt = Now()
	return s.db.QueryRowContext(ctx,
		`INSERT INTO users (google_sub, email, name, picture_url, role, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		u.GoogleSub, u.Email, u.Name, u.PictureURL, u.Role, u.CreatedAt,
	).Scan(&u.ID)
}

// GoogleProfile is what a Google sign-in tells us about the person.
type GoogleProfile struct {
	Sub     string
	Email   string // lower-cased
	Name    string
	Picture *string
}

// UpsertGoogleUser finds the account for a Google profile (by Google id,
// then by email, for someone who first used the development login) or
// creates it, and refreshes its email, name and picture.
func (s *Store) UpsertGoogleUser(ctx context.Context, p GoogleProfile) (*User, error) {
	u, err := s.UserByGoogleSub(ctx, p.Sub)
	if err != nil {
		return nil, err
	}
	if u == nil {
		if u, err = s.UserByEmail(ctx, p.Email); err != nil {
			return nil, err
		}
	}
	sub := p.Sub
	if u == nil {
		u = &User{GoogleSub: &sub, Email: p.Email, Name: p.Name, PictureURL: p.Picture}
		return u, s.insertUser(ctx, u)
	}
	u.GoogleSub, u.Email, u.Name, u.PictureURL = &sub, p.Email, p.Name, p.Picture
	_, err = s.db.ExecContext(ctx,
		`UPDATE users SET google_sub = $1, email = $2, name = $3, picture_url = $4 WHERE id = $5`,
		u.GoogleSub, u.Email, u.Name, u.PictureURL, u.ID)
	return u, err
}

// UpsertDevUser finds or creates an account by email, for the development
// login.
func (s *Store) UpsertDevUser(ctx context.Context, email, name string) (*User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	u, err := s.UserByEmail(ctx, email)
	if err != nil || u != nil {
		return u, err
	}
	u = &User{Email: email, Name: strings.TrimSpace(name)}
	return u, s.insertUser(ctx, u)
}

// SetRole changes a user's role.
func (s *Store) SetRole(ctx context.Context, userID int64, role string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE users SET role = $1 WHERE id = $2`, role, userID)
	return err
}

// --- Sessions -------------------------------------------------------------

// CreateSession stores a signed-in browser. Only the hash of its cookie is
// kept, so a database leak doesn't hand out working sessions.
func (s *Store) CreateSession(ctx context.Context, tokenHash string, userID int64, expires time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (token_hash, user_id, created_at, expires_at) VALUES ($1, $2, $3, $4)`,
		tokenHash, userID, Now(), expires.UTC().Truncate(time.Microsecond))
	return err
}

// UserForSession returns the user a session belongs to, or nil if the
// session is unknown or expired.
func (s *Store) UserForSession(ctx context.Context, tokenHash string) (*User, error) {
	var expires time.Time
	row := s.db.QueryRowContext(ctx,
		`SELECT u.id, u.google_sub, u.email, u.name, u.picture_url, u.role, u.created_at, s.expires_at
		 FROM sessions s JOIN users u ON u.id = s.user_id WHERE s.token_hash = $1`, tokenHash)
	var u User
	err := row.Scan(&u.ID, &u.GoogleSub, &u.Email, &u.Name, &u.PictureURL, &u.Role,
		timeCol{&u.CreatedAt}, timeCol{&expires})
	user, err := one(&u, err)
	if user == nil || err != nil {
		return nil, err
	}
	if expires.Before(time.Now()) {
		return nil, nil
	}
	return user, nil
}

// DeleteSession signs a browser out.
func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	return err
}
