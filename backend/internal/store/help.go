package store

import (
	"context"
	"time"
)

// recentHelped is how many people a professional's list shows; the count
// always covers every call.
const recentHelped = 50

// HelpRecord is one person a verified professional helped: written once,
// when an accepted call ends.
type HelpRecord struct {
	ProfessionalID  int64
	ConsultationID  string
	PatientName     string
	StartedAt       time.Time // when the professional accepted
	EndedAt         time.Time
	DurationSeconds int
}

// HelpedPerson is a row of a professional's own record.
type HelpedPerson struct {
	PatientName     string
	StartedAt       time.Time
	DurationSeconds int
}

// RecordHelp saves a finished call. It reports false, without an error, if
// the call was already recorded: both participants report the end.
func (s *Store) RecordHelp(ctx context.Context, r HelpRecord) (bool, error) {
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO help_records (professional_id, consultation_id, patient_name, started_at,
			ended_at, duration_seconds)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (consultation_id) DO NOTHING`,
		r.ProfessionalID, r.ConsultationID, r.PatientName,
		r.StartedAt.UTC().Truncate(time.Microsecond), r.EndedAt.UTC().Truncate(time.Microsecond),
		r.DurationSeconds)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}

// HelpSummary returns how many people a professional has helped, and the
// most recent of them, newest first.
func (s *Store) HelpSummary(ctx context.Context, professionalID int64) (int, []HelpedPerson, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT count(*) FROM help_records WHERE professional_id = $1`, professionalID).Scan(&count)
	if err != nil {
		return 0, nil, err
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT patient_name, started_at, duration_seconds FROM help_records
		 WHERE professional_id = $1 ORDER BY started_at DESC, id DESC LIMIT $2`,
		professionalID, recentHelped)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	people := []HelpedPerson{}
	for rows.Next() {
		var p HelpedPerson
		if err := rows.Scan(&p.PatientName, timeCol{&p.StartedAt}, &p.DurationSeconds); err != nil {
			return 0, nil, err
		}
		people = append(people, p)
	}
	return count, people, rows.Err()
}
