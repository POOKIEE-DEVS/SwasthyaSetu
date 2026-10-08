package api

import (
	"context"
	"errors"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/consult"
	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/realtime"
	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/store"
)

// A patient requests a professional; a verified professional accepts;
// either side ends the call.
//
// Patients never sign in: an emergency must not wait on a login, and this
// path never touches the database. Seeing the waiting list (which includes
// shared chats) and accepting are for verified professionals only.

// callTicket is everything one participant needs to join the call.
type callTicket struct {
	Consultation consult.Public `json:"consultation"`
	Role         string         `json:"role"`
	// Room-scoped secret for the signalling socket. Only the two
	// participants ever hold one, which is what stops a third person joining.
	Token      string               `json:"token"`
	IceServers []realtime.IceServer `json:"ice_servers"`
}

type queueMessage struct {
	Type          string           `json:"type"`
	Consultations []consult.Public `json:"consultations"`
}

func (s *Server) queueSnapshot() any {
	waiting := s.consults.Waiting()
	out := make([]consult.Public, 0, len(waiting))
	for _, c := range waiting {
		out = append(out, c.Public())
	}
	return queueMessage{Type: "queue", Consultations: out}
}

func (s *Server) broadcastQueue() { s.hub.BroadcastQueue(s.queueSnapshot) }

func (s *Server) requestProfessional(w http.ResponseWriter, r *http.Request) error {
	var body struct {
		PatientName *string `json:"patient_name"`
		// The patient's chat with the AI, shared only when they tick the
		// consent box.
		Summary *string `json:"summary"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		return err
	}
	if body.PatientName == nil || strings.TrimSpace(*body.PatientName) == "" || chars(*body.PatientName) > 60 {
		return invalid("Please enter your name (up to 60 characters).")
	}
	if body.Summary != nil && chars(*body.Summary) > 6000 {
		return invalid("The shared chat is too long.")
	}
	c := s.consults.Create(*body.PatientName, body.Summary)
	s.broadcastQueue()
	writeJSON(w, http.StatusCreated, callTicket{
		Consultation: c.Public(), Role: consult.Patient, Token: c.PatientToken,
		IceServers: s.ice.Servers(r.Context()),
	})
	return nil
}

func (s *Server) waitingPatients(w http.ResponseWriter, r *http.Request) error {
	if _, err := s.requireProfessional(r); err != nil {
		return err
	}
	return writeOK(w, s.queueSnapshot().(queueMessage).Consultations)
}

func (s *Server) accept(w http.ResponseWriter, r *http.Request) error {
	pro, err := s.requireProfessional(r)
	if err != nil {
		return err
	}
	c, err := s.consults.Accept(r.PathValue("id"), pro.badge(), pro.user.ID)
	switch {
	case errors.Is(err, consult.ErrNotFound):
		return fail(http.StatusNotFound, "This request no longer exists.")
	case errors.Is(err, consult.ErrUnavailable):
		return fail(http.StatusConflict, "Another doctor has already taken this request.")
	case err != nil:
		return err
	}
	s.broadcastQueue()
	return writeOK(w, callTicket{
		Consultation: c.Public(), Role: consult.Doctor, Token: c.DoctorToken,
		IceServers: s.ice.Servers(r.Context()),
	})
}

func (s *Server) end(w http.ResponseWriter, r *http.Request) error {
	var body struct {
		Token *string `json:"token"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		return err
	}
	if body.Token == nil {
		return invalid("The call token is missing.")
	}
	c, completed, err := s.consults.End(r.PathValue("id"), *body.Token)
	if errors.Is(err, consult.ErrNotFound) {
		return fail(http.StatusNotFound, "Not found.")
	}
	if err != nil {
		return err
	}
	if completed {
		s.recordHelp(r.Context(), c)
	}
	// A patient who gives up while waiting leaves the professionals' queue.
	s.broadcastQueue()
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// recordHelp adds a finished call to the professional's help record. It
// never fails the hang-up: with the database down, the record is skipped
// and logged.
func (s *Server) recordHelp(ctx context.Context, c consult.Consultation) {
	if c.ProfessionalUserID == 0 || c.AcceptedAt.IsZero() {
		return
	}
	if !s.db.Ready() {
		s.log.Warn("help record skipped: database unavailable")
		return
	}
	ended := time.Now()
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	_, err := s.store.RecordHelp(ctx, store.HelpRecord{
		ProfessionalID:  c.ProfessionalUserID,
		ConsultationID:  c.ID,
		PatientName:     c.PatientName,
		StartedAt:       c.AcceptedAt,
		EndedAt:         ended,
		DurationSeconds: int(math.Max(0, math.RoundToEven(ended.Sub(c.AcceptedAt).Seconds()))),
	})
	if err != nil {
		s.log.Warn("could not save help record", "error", err)
	}
}

type helpedPerson struct {
	PatientName     string  `json:"patient_name"`
	StartedAt       float64 `json:"started_at"`
	DurationSeconds int     `json:"duration_seconds"`
}

type helpSummary struct {
	Count  int            `json:"count"`
	People []helpedPerson `json:"people"`
}

// peopleHelped is the signed-in professional's own record, and nobody else's.
func (s *Server) peopleHelped(w http.ResponseWriter, r *http.Request) error {
	pro, err := s.requireProfessional(r)
	if err != nil {
		return err
	}
	count, people, err := s.store.HelpSummary(r.Context(), pro.user.ID)
	if err != nil {
		return err
	}
	out := helpSummary{Count: count, People: []helpedPerson{}}
	for _, p := range people {
		out.People = append(out.People, helpedPerson{
			PatientName: p.PatientName, StartedAt: store.Unix(p.StartedAt), DurationSeconds: p.DurationSeconds,
		})
	}
	return writeOK(w, out)
}
