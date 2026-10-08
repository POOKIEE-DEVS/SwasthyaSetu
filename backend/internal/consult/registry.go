// Package consult is the in-memory queue of patients waiting for a
// professional, and the calls in progress.
//
// State lives in this process only and is lost on restart, which is fine for
// the demo. It is also why the backend must run as exactly one process: a
// second one would hold a separate queue and separate call rooms.
//
// Every method takes the registry's lock, so each is atomic: two doctors
// pressing "Accept" at once cannot both win.
package consult

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

// Status of a consultation.
const (
	Waiting = "waiting"
	Active  = "active"
	Ended   = "ended"
)

// Roles of the two participants.
const (
	Patient = "patient"
	Doctor  = "doctor" // whichever verified professional accepted
)

var (
	// ErrNotFound means there is no such consultation (or the token is wrong).
	ErrNotFound = errors.New("consultation not found")
	// ErrUnavailable means it was already accepted, or has ended.
	ErrUnavailable = errors.New("consultation unavailable")
)

// Badge is who accepted the call, shown to the patient as "Verified Doctor"
// (or Nurse, Paramedic, ...).
type Badge struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

// Consultation is one request for a professional. Values handed out by the
// registry are copies; change them only through the registry.
type Consultation struct {
	ID           string
	PatientName  string
	Summary      *string
	PatientToken string
	DoctorToken  string
	Professional *Badge
	// The account that accepted, and when: for their help record. Never
	// sent to clients.
	ProfessionalUserID int64
	AcceptedAt         time.Time
	Status             string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// Public is what the queue and the participants see. Never includes tokens.
type Public struct {
	ID           string  `json:"id"`
	PatientName  string  `json:"patient_name"`
	Summary      *string `json:"summary"`
	Status       string  `json:"status"`
	CreatedAt    float64 `json:"created_at"`
	Professional *Badge  `json:"professional"`
}

// Public returns the client-facing view.
func (c Consultation) Public() Public {
	return Public{
		ID:           c.ID,
		PatientName:  c.PatientName,
		Summary:      c.Summary,
		Status:       c.Status,
		CreatedAt:    unix(c.CreatedAt),
		Professional: c.Professional,
	}
}

func unix(t time.Time) float64 { return float64(t.UnixMicro()) / 1e6 }

// RoleFor says which participant a token belongs to ("" for neither). The
// comparison takes the same time however much of the token matches.
func (c Consultation) RoleFor(token string) string {
	if token == "" {
		return ""
	}
	if subtle.ConstantTimeCompare([]byte(token), []byte(c.PatientToken)) == 1 {
		return Patient
	}
	if c.DoctorToken != "" && subtle.ConstantTimeCompare([]byte(token), []byte(c.DoctorToken)) == 1 {
		return Doctor
	}
	return ""
}

// Registry holds the consultations.
type Registry struct {
	mu    sync.Mutex
	items map[string]*Consultation
	ttl   time.Duration
	now   func() time.Time
}

// NewRegistry drops unanswered or finished requests ttl after their last
// change.
func NewRegistry(ttl time.Duration) *Registry {
	return &Registry{items: map[string]*Consultation{}, ttl: ttl, now: time.Now}
}

func (r *Registry) purgeExpired() {
	cutoff := r.now().Add(-r.ttl)
	for id, c := range r.items {
		if c.UpdatedAt.Before(cutoff) {
			delete(r.items, id)
		}
	}
}

// Create adds a patient's request to the queue.
func (r *Registry) Create(patientName string, summary *string) Consultation {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.purgeExpired()
	var trimmed *string
	if summary != nil {
		if s := strings.TrimSpace(*summary); s != "" {
			trimmed = &s
		}
	}
	now := r.now()
	c := &Consultation{
		ID:           token(8),
		PatientName:  strings.TrimSpace(patientName),
		Summary:      trimmed,
		PatientToken: token(24),
		Status:       Waiting,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	r.items[c.ID] = c
	return *c
}

// Get returns a consultation.
func (r *Registry) Get(id string) (Consultation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.items[id]
	if !ok {
		return Consultation{}, ErrNotFound
	}
	return *c, nil
}

// Waiting returns the queue, oldest first.
func (r *Registry) Waiting() []Consultation {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.purgeExpired()
	out := []Consultation{}
	for _, c := range r.items {
		if c.Status == Waiting {
			out = append(out, *c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

// Accept gives a waiting consultation to a verified professional.
func (r *Registry) Accept(id string, badge Badge, professionalUserID int64) (Consultation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.items[id]
	if !ok {
		return Consultation{}, ErrNotFound
	}
	if c.Status != Waiting {
		return Consultation{}, ErrUnavailable
	}
	now := r.now()
	c.Status = Active
	c.Professional = &badge
	c.ProfessionalUserID = professionalUserID
	c.DoctorToken = token(24)
	c.AcceptedAt, c.UpdatedAt = now, now
	return *c, nil
}

// End ends a call for a participant (by their token). It also reports
// whether this ended an accepted call, the first time only: that is when it
// goes in the help record (both participants report the end, and a patient
// may give up while still waiting).
func (r *Registry) End(id, participantToken string) (Consultation, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.items[id]
	if !ok || c.RoleFor(participantToken) == "" {
		return Consultation{}, false, ErrNotFound
	}
	completed := c.Status == Active
	c.Status = Ended
	c.UpdatedAt = r.now()
	return *c, completed, nil
}

// Clear forgets everything (between rehearsals or tests).
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = map[string]*Consultation{}
}

// token returns n random bytes, URL-safe base64 encoded.
func token(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b) // never fails
	return base64.RawURLEncoding.EncodeToString(b)
}
