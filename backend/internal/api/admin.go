package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/store"
)

// The admin's review of professional applications. One admin (any account
// in ADMIN_EMAILS) checks each applicant's documents by hand, looks the
// council number up on the council's register, then approves or rejects
// with a reason the applicant will see.

func pathID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return 0, invalid("Unknown id.")
	}
	return id, nil
}

func (s *Server) listApplications(w http.ResponseWriter, r *http.Request) error {
	if _, err := s.requireAdmin(r); err != nil {
		return err
	}
	status := r.URL.Query().Get("status")
	switch status {
	case "":
		status = store.StatusPending
	case store.StatusPending, store.StatusApproved, store.StatusRejected:
	case "all":
		status = ""
	default:
		return invalid("Status must be pending, approved, rejected or all.")
	}
	applications, err := s.store.ListApplications(r.Context(), status)
	if err != nil {
		return err
	}
	out := []*adminApplication{}
	for _, a := range applications {
		view, err := s.adminApplication(r.Context(), a)
		if err != nil {
			return err
		}
		out = append(out, view)
	}
	return writeOK(w, out)
}

func (s *Server) document(w http.ResponseWriter, r *http.Request) error {
	if _, err := s.requireAdmin(r); err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	doc, err := s.store.Document(r.Context(), id)
	if err != nil {
		return err
	}
	if doc == nil {
		return fail(http.StatusNotFound, "Document not found.")
	}
	h := w.Header()
	// Detected from the bytes at upload time: only images and PDFs.
	h.Set("Content-Type", doc.ContentType)
	h.Set("Content-Length", strconv.Itoa(len(doc.Data)))
	h.Set("Content-Disposition", "inline")
	h.Set("Cache-Control", "private, no-store")
	h.Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = w.Write(doc.Data)
	}
	return nil
}

// applicationInPath returns the application named in the path, or a 404.
func (s *Server) applicationInPath(r *http.Request) (*store.Application, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	application, err := s.store.ApplicationByID(r.Context(), id)
	if err != nil {
		return nil, err
	}
	if application == nil {
		return nil, fail(http.StatusNotFound, "Application not found.")
	}
	return application, nil
}

func (s *Server) approve(w http.ResponseWriter, r *http.Request) error {
	admin, err := s.requireAdmin(r)
	if err != nil {
		return err
	}
	application, err := s.applicationInPath(r)
	if err != nil {
		return err
	}
	if application.Status == store.StatusApproved {
		return fail(http.StatusConflict, "Already approved.")
	}
	return s.decide(w, r, admin, application, true, nil)
}

// reject also revokes an approval (e.g. a licence turns out to be invalid).
func (s *Server) reject(w http.ResponseWriter, r *http.Request) error {
	admin, err := s.requireAdmin(r)
	if err != nil {
		return err
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		return err
	}
	// Shown to the applicant, so they know what to fix.
	reason := strings.TrimSpace(body.Reason)
	if chars(reason) < 3 || chars(reason) > 500 {
		return invalid("Please give a reason of 3 to 500 characters.")
	}
	application, err := s.applicationInPath(r)
	if err != nil {
		return err
	}
	if application.Status == store.StatusRejected {
		return fail(http.StatusConflict, "Already rejected.")
	}
	return s.decide(w, r, admin, application, false, &reason)
}

func (s *Server) decide(w http.ResponseWriter, r *http.Request, admin *store.User, a *store.Application, approve bool, reason *string) error {
	decided, err := s.store.DecideApplication(r.Context(), a.ID, admin.Email, approve, reason)
	if err != nil {
		return err
	}
	view, err := s.adminApplication(r.Context(), decided)
	if err != nil {
		return err
	}
	return writeOK(w, view)
}
