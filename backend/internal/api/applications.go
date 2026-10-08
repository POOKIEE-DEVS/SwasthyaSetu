package api

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/store"
	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/verify"
)

// A professional applies for verification (KYC); the rules are in package
// verify. Until an admin approves the application, the applicant can't see
// or call patients.

var (
	textFields = map[string]bool{
		"role": true, "full_name": true, "phone": true, "citizenship_number": true,
		"citizenship_district": true, "consent": true, "council_number": true,
		"institution": true, "recommender_name": true, "recommender_nmc": true,
	}
	fileFields = map[string]bool{
		"citizenship_front": true, "citizenship_back": true, "council_certificate": true,
		"recommendation_letter": true, "selfie": true,
	}
)

const maxTextField = 160 // characters

func (s *Server) myApplication(w http.ResponseWriter, r *http.Request) error {
	user, err := s.currentUser(r)
	if err != nil {
		return err
	}
	application, err := s.store.ApplicationForUser(r.Context(), user.ID)
	if err != nil {
		return err
	}
	if application == nil {
		return writeOK(w, nil) // JSON null: not applied yet
	}
	view, err := s.applicationPublic(r.Context(), application)
	if err != nil {
		return err
	}
	return writeOK(w, view)
}

func (s *Server) submitApplication(w http.ResponseWriter, r *http.Request) error {
	user, err := s.currentUser(r)
	if err != nil {
		return err
	}
	text, files, err := s.readForm(w, r)
	if err != nil {
		return err
	}
	if !store.IsProfessionalRole(text["role"]) {
		return invalid("Please choose your profession.")
	}
	for _, value := range text {
		if chars(value) > maxTextField {
			return invalid("Please keep each field under 160 characters.")
		}
	}
	if consent, ok := parseBool(text["consent"]); !ok || !consent {
		return invalid("Please agree to the document check before submitting.")
	}
	existing, err := s.store.ApplicationForUser(r.Context(), user.ID)
	if err != nil {
		return err
	}
	if existing != nil && existing.Status == store.StatusApproved {
		return fail(http.StatusConflict, "You are already verified.")
	}

	fields, docs, err := verify.Check(verify.Form{
		Role:                text["role"],
		FullName:            text["full_name"],
		Phone:               text["phone"],
		CitizenshipNumber:   text["citizenship_number"],
		CitizenshipDistrict: text["citizenship_district"],
		CouncilNumber:       text["council_number"],
		Institution:         text["institution"],
		RecommenderName:     text["recommender_name"],
		RecommenderNMC:      text["recommender_nmc"],
		Files:               files,
	}, s.cfg.MaxUploadBytes)
	if err != nil {
		return err
	}
	application, err := s.store.SubmitApplication(r.Context(), user, fields, docs)
	if err != nil {
		return err
	}
	view, err := s.applicationPublic(r.Context(), application)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, view)
	return nil
}

// readForm streams the multipart form, keeping each file up to the size
// limit plus one byte (enough to tell that it is too large) and never
// writing anything to disk.
func (s *Server) readForm(w http.ResponseWriter, r *http.Request) (map[string]string, map[string]*verify.Upload, error) {
	limit := s.cfg.MaxUploadBytes
	r.Body = http.MaxBytesReader(w, r.Body, int64(len(fileFields)+1)*limit+1<<20)
	reader, err := r.MultipartReader()
	if err != nil {
		return nil, nil, invalid("Please send the application as a form.")
	}
	text := map[string]string{}
	files := map[string]*verify.Upload{}
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			return text, files, nil
		}
		if err != nil {
			return nil, nil, formError(err)
		}
		name := part.FormName()
		_, params, _ := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
		_, isFile := params["filename"]
		switch {
		case fileFields[name] && isFile:
			data, err := io.ReadAll(io.LimitReader(part, limit+1))
			if err != nil {
				return nil, nil, formError(err)
			}
			files[name] = &verify.Upload{Filename: params["filename"], Data: data}
		case textFields[name] && !isFile:
			data, err := io.ReadAll(io.LimitReader(part, 16<<10))
			if err != nil {
				return nil, nil, formError(err)
			}
			text[name] = string(data)
		}
		_ = part.Close()
	}
}

func formError(err error) error {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		return fail(http.StatusRequestEntityTooLarge, "The upload is too large. Please send smaller files.")
	}
	return invalid("The form could not be read. Please try again.")
}

// parseBool reads a form checkbox the way the Python backend did.
func parseBool(value string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "0", "false", "f", "no", "n", "off":
		return false, true
	case "1", "true", "t", "yes", "y", "on":
		return true, true
	}
	return false, false
}
