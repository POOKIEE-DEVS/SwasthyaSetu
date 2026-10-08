package store

import (
	"context"
	"time"

	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/database"
)

// Application statuses.
const (
	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusRejected = "rejected"
)

// Application is a professional's verification request (KYC). One per
// account; a rejected applicant edits and resubmits the same one.
type Application struct {
	ID                  int64
	UserID              int64
	Role                string // one of ProfessionalRoles
	FullName            string
	Phone               string
	CitizenshipNumber   string
	CitizenshipDistrict string
	// The council registration number (doctors, pharmacists, nurses,
	// paramedics).
	CouncilNumber *string
	// MBBS students: their college, and the doctor recommending them.
	Institution     *string
	RecommenderName *string
	RecommenderNMC  *string
	Status          string
	RejectionReason *string
	SubmittedAt     time.Time
	ReviewedAt      *time.Time
	ReviewedBy      *string
}

// ApplicationFields are what the applicant fills in.
type ApplicationFields struct {
	Role                string
	FullName            string
	Phone               string
	CitizenshipNumber   string
	CitizenshipDistrict string
	CouncilNumber       *string
	Institution         *string
	RecommenderName     *string
	RecommenderNMC      *string
}

// NewDocument is an uploaded file that passed the checks in package verify.
type NewDocument struct {
	Kind        string
	ContentType string // detected from the bytes, never taken from the browser
	Data        []byte
}

// DocumentInfo describes a stored document without its bytes.
type DocumentInfo struct {
	ID          int64
	Kind        string
	ContentType string
	Size        int64
}

// Document is a stored document with its bytes.
type Document struct {
	DocumentInfo
	Data []byte
}

// AuditEvent records who submitted, approved or rejected an application.
type AuditEvent struct {
	Action     string // submitted | resubmitted | approved | rejected
	ActorEmail string
	Reason     *string
	At         time.Time
}

// IsVerified reports whether the user may see and call patients: their
// application is approved and their account role matches it.
func IsVerified(u *User, a *Application) bool {
	return a != nil && a.Status == StatusApproved && u.Role != nil && *u.Role == a.Role
}

const applicationColumns = `id, user_id, role, full_name, phone, citizenship_number,
	citizenship_district, council_number, institution, recommender_name, recommender_nmc,
	status, rejection_reason, submitted_at, reviewed_at, reviewed_by`

func scanApplication(row rowScanner) (*Application, error) {
	var a Application
	err := row.Scan(&a.ID, &a.UserID, &a.Role, &a.FullName, &a.Phone, &a.CitizenshipNumber,
		&a.CitizenshipDistrict, &a.CouncilNumber, &a.Institution, &a.RecommenderName,
		&a.RecommenderNMC, &a.Status, &a.RejectionReason, timeCol{&a.SubmittedAt},
		nullTimeCol{&a.ReviewedAt}, &a.ReviewedBy)
	return &a, err
}

// ApplicationForUser returns a user's application, or nil.
func (s *Store) ApplicationForUser(ctx context.Context, userID int64) (*Application, error) {
	return applicationForUser(ctx, s.db, userID)
}

func applicationForUser(ctx context.Context, q database.Querier, userID int64) (*Application, error) {
	return one(scanApplication(q.QueryRowContext(ctx,
		`SELECT `+applicationColumns+` FROM applications WHERE user_id = $1`, userID)))
}

// ApplicationByID returns an application, or nil.
func (s *Store) ApplicationByID(ctx context.Context, id int64) (*Application, error) {
	return one(scanApplication(s.db.QueryRowContext(ctx,
		`SELECT `+applicationColumns+` FROM applications WHERE id = $1`, id)))
}

// ListApplications returns applications with a status ("" for all). Pending
// ones come oldest first, a fair queue; the others newest first.
func (s *Store) ListApplications(ctx context.Context, status string) ([]*Application, error) {
	query := `SELECT ` + applicationColumns + ` FROM applications`
	var args []any
	if status != "" {
		query += ` WHERE status = $1`
		args = append(args, status)
	}
	if status == StatusPending {
		query += ` ORDER BY submitted_at, id`
	} else {
		query += ` ORDER BY submitted_at DESC, id DESC`
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Application
	for rows.Next() {
		a, err := scanApplication(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// SubmitApplication creates the user's application, or replaces a pending or
// rejected one (back to pending, old documents removed). It also makes the
// applied role the account's role, and writes the audit event.
func (s *Store) SubmitApplication(ctx context.Context, user *User, f ApplicationFields, docs []NewDocument) (*Application, error) {
	var id int64
	err := s.db.InTx(ctx, func(tx *database.Tx) error {
		existing, err := applicationForUser(ctx, tx, user.ID)
		if err != nil {
			return err
		}
		now := Now()
		action := "submitted"
		if existing == nil {
			err = tx.QueryRowContext(ctx,
				`INSERT INTO applications (user_id, role, full_name, phone, citizenship_number,
					citizenship_district, council_number, institution, recommender_name,
					recommender_nmc, status, submitted_at)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING id`,
				user.ID, f.Role, f.FullName, f.Phone, f.CitizenshipNumber, f.CitizenshipDistrict,
				f.CouncilNumber, f.Institution, f.RecommenderName, f.RecommenderNMC,
				StatusPending, now,
			).Scan(&id)
			if err != nil {
				return err
			}
		} else {
			id, action = existing.ID, "resubmitted"
			_, err = tx.ExecContext(ctx,
				`UPDATE applications SET role = $1, full_name = $2, phone = $3,
					citizenship_number = $4, citizenship_district = $5, council_number = $6,
					institution = $7, recommender_name = $8, recommender_nmc = $9,
					status = $10, rejection_reason = NULL, reviewed_at = NULL,
					reviewed_by = NULL, submitted_at = $11
				 WHERE id = $12`,
				f.Role, f.FullName, f.Phone, f.CitizenshipNumber, f.CitizenshipDistrict,
				f.CouncilNumber, f.Institution, f.RecommenderName, f.RecommenderNMC,
				StatusPending, now, id)
			if err != nil {
				return err
			}
			if _, err = tx.ExecContext(ctx, `DELETE FROM documents WHERE application_id = $1`, id); err != nil {
				return err
			}
		}
		if _, err = tx.ExecContext(ctx, `UPDATE users SET role = $1 WHERE id = $2`, f.Role, user.ID); err != nil {
			return err
		}
		for _, d := range docs {
			_, err = tx.ExecContext(ctx,
				`INSERT INTO documents (application_id, kind, content_type, size, data, created_at)
				 VALUES ($1, $2, $3, $4, $5, $6)`,
				id, d.Kind, d.ContentType, len(d.Data), d.Data, now)
			if err != nil {
				return err
			}
		}
		_, err = tx.ExecContext(ctx,
			`INSERT INTO audit_events (application_id, actor_email, action, at) VALUES ($1, $2, $3, $4)`,
			id, user.Email, action, now)
		return err
	})
	if err != nil {
		return nil, err
	}
	role := f.Role
	user.Role = &role
	return s.ApplicationByID(ctx, id)
}

// DecideApplication approves or rejects (reason required) an application,
// and writes the audit event. Rejecting an approved one revokes it.
func (s *Store) DecideApplication(ctx context.Context, applicationID int64, adminEmail string, approve bool, reason *string) (*Application, error) {
	status, action := StatusRejected, "rejected"
	rejection := reason
	if approve {
		status, action, rejection = StatusApproved, "approved", nil
	}
	now := Now()
	err := s.db.InTx(ctx, func(tx *database.Tx) error {
		_, err := tx.ExecContext(ctx,
			`UPDATE applications SET status = $1, rejection_reason = $2, reviewed_at = $3, reviewed_by = $4
			 WHERE id = $5`,
			status, rejection, now, adminEmail, applicationID)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx,
			`INSERT INTO audit_events (application_id, actor_email, action, reason, at)
			 VALUES ($1, $2, $3, $4, $5)`,
			applicationID, adminEmail, action, reason, now)
		return err
	})
	if err != nil {
		return nil, err
	}
	return s.ApplicationByID(ctx, applicationID)
}

// Documents lists an application's documents, oldest first.
func (s *Store) Documents(ctx context.Context, applicationID int64) ([]DocumentInfo, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, kind, content_type, size FROM documents WHERE application_id = $1 ORDER BY id`,
		applicationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DocumentInfo{}
	for rows.Next() {
		var d DocumentInfo
		if err := rows.Scan(&d.ID, &d.Kind, &d.ContentType, &d.Size); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// Document returns a document with its bytes, or nil.
func (s *Store) Document(ctx context.Context, id int64) (*Document, error) {
	var d Document
	err := s.db.QueryRowContext(ctx,
		`SELECT id, kind, content_type, size, data FROM documents WHERE id = $1`, id,
	).Scan(&d.ID, &d.Kind, &d.ContentType, &d.Size, &d.Data)
	return one(&d, err)
}

// History returns an application's audit events, oldest first.
func (s *Store) History(ctx context.Context, applicationID int64) ([]AuditEvent, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT action, actor_email, reason, at FROM audit_events WHERE application_id = $1 ORDER BY id`,
		applicationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AuditEvent{}
	for rows.Next() {
		var e AuditEvent
		if err := rows.Scan(&e.Action, &e.ActorEmail, &e.Reason, timeCol{&e.At}); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
