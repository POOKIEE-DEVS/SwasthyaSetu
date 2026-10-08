// Package verify holds the rules of professional verification (KYC): what
// each profession must provide, and the checks on uploaded documents.
//
// Everyone uploads both sides of their citizenship certificate. Then:
//   - Doctor: Nepal Medical Council (NMC) number and certificate.
//   - Pharmacist: Nepal Pharmacy Council number and certificate.
//   - Nurse: Nepal Nursing Council number and certificate.
//   - Paramedic (health assistant, CMA and similar): Nepal Health
//     Professional Council number and certificate.
//   - MBBS student: college, the recommending doctor's name and NMC number,
//     and the recommendation letter.
//
// A selfie holding the citizenship certificate is optional. An admin then
// reviews every application by hand.
package verify

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/store"
)

// Councils maps each registered profession to the council it is checked
// against. Students have none: a doctor recommends them.
var Councils = map[string]string{
	store.RoleDoctor:     "Nepal Medical Council",
	store.RolePharmacist: "Nepal Pharmacy Council",
	store.RoleNurse:      "Nepal Nursing Council",
	store.RoleParamedic:  "Nepal Health Professional Council",
}

var (
	phoneRE    = regexp.MustCompile(`^\+?[0-9][0-9 -]{6,18}$`)
	idNumberRE = regexp.MustCompile(`^[0-9A-Za-z][0-9A-Za-z /-]{0,39}$`)
)

// Error is a refused application. Its message is safe to show the applicant.
type Error struct{ Message string }

func (e *Error) Error() string { return e.Message }

func refuse(format string, args ...any) error { return &Error{fmt.Sprintf(format, args...)} }

// Upload is a file from the form, read up to the size limit plus one byte.
type Upload struct {
	Filename string
	Data     []byte
}

// Provided reports whether a file was chosen. Browsers send an empty
// filename for an empty file input.
func (u *Upload) Provided() bool { return u != nil && u.Filename != "" }

// Form is a submitted application, as typed in.
type Form struct {
	Role                string
	FullName            string
	Phone               string
	CitizenshipNumber   string
	CitizenshipDistrict string
	CouncilNumber       string
	Institution         string
	RecommenderName     string
	RecommenderNMC      string
	// By field name: citizenship_front, citizenship_back,
	// council_certificate, recommendation_letter, selfie.
	Files map[string]*Upload
}

// Check validates a form (consent and "already verified" are checked by the
// caller first) and returns what to store. Errors are *Error.
func Check(f Form, maxBytes int64) (store.ApplicationFields, []store.NewDocument, error) {
	fields := store.ApplicationFields{
		Role:                f.Role,
		FullName:            strings.TrimSpace(f.FullName),
		Phone:               strings.TrimSpace(f.Phone),
		CitizenshipNumber:   strings.TrimSpace(f.CitizenshipNumber),
		CitizenshipDistrict: strings.TrimSpace(f.CitizenshipDistrict),
	}
	if utf8.RuneCountInString(fields.FullName) < 2 {
		return fields, nil, refuse("Please enter your full name as on your citizenship.")
	}
	if !phoneRE.MatchString(fields.Phone) {
		return fields, nil, refuse("Please enter a valid phone number.")
	}
	if !idNumberRE.MatchString(fields.CitizenshipNumber) {
		return fields, nil, refuse("Please enter your citizenship number.")
	}
	if fields.CitizenshipDistrict == "" {
		return fields, nil, refuse("Please enter the district that issued your citizenship.")
	}
	front, back := f.Files["citizenship_front"], f.Files["citizenship_back"]
	if !front.Provided() || !back.Provided() {
		return fields, nil, refuse("Please upload both sides of your citizenship certificate.")
	}

	var docs []store.NewDocument
	add := func(u *Upload, kind, label string, allowPDF bool) error {
		doc, err := CheckUpload(u, kind, label, maxBytes, allowPDF)
		if err == nil {
			docs = append(docs, doc)
		}
		return err
	}
	if err := add(front, "citizenship_front", "Citizenship (front)", true); err != nil {
		return fields, nil, err
	}
	if err := add(back, "citizenship_back", "Citizenship (back)", true); err != nil {
		return fields, nil, err
	}

	if council, ok := Councils[f.Role]; ok {
		number := strings.TrimSpace(f.CouncilNumber)
		if !idNumberRE.MatchString(number) {
			return fields, nil, refuse("Please enter your %s registration number.", council)
		}
		fields.CouncilNumber = &number
		certificate := f.Files["council_certificate"]
		if !certificate.Provided() {
			return fields, nil, refuse("Please upload your %s certificate.", council)
		}
		if err := add(certificate, "council_certificate", council+" certificate", true); err != nil {
			return fields, nil, err
		}
	} else { // MBBS student
		fields.Institution = optional(f.Institution)
		fields.RecommenderName = optional(f.RecommenderName)
		fields.RecommenderNMC = optional(f.RecommenderNMC)
		if fields.Institution == nil {
			return fields, nil, refuse("Please enter your medical college.")
		}
		if fields.RecommenderName == nil {
			return fields, nil, refuse("Please enter the name of the doctor recommending you.")
		}
		if fields.RecommenderNMC == nil || !idNumberRE.MatchString(*fields.RecommenderNMC) {
			return fields, nil, refuse("Please enter the recommending doctor's NMC number.")
		}
		letter := f.Files["recommendation_letter"]
		if !letter.Provided() {
			return fields, nil, refuse("Please upload the doctor's letter of recommendation.")
		}
		if err := add(letter, "recommendation_letter", "Letter of recommendation", true); err != nil {
			return fields, nil, err
		}
	}

	if selfie := f.Files["selfie"]; selfie.Provided() {
		if err := add(selfie, "selfie", "Selfie with citizenship", false); err != nil {
			return fields, nil, err
		}
	}
	return fields, docs, nil
}

func optional(value string) *string {
	if value = strings.TrimSpace(value); value == "" {
		return nil
	}
	return &value
}

// Content types a document may have.
const (
	TypeJPEG = "image/jpeg"
	TypePNG  = "image/png"
	TypeWebP = "image/webp"
	TypePDF  = "application/pdf"
)

// SniffType decides a file's type from its first bytes, never from its name
// or the browser's claim, so a script renamed to .jpg is refused and
// documents are always served back with a type that can't run code.
func SniffType(data []byte) string {
	switch {
	case bytes.HasPrefix(data, []byte("\xff\xd8\xff")):
		return TypeJPEG
	case bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")):
		return TypePNG
	case len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP":
		return TypeWebP
	case bytes.HasPrefix(data, []byte("%PDF-")):
		return TypePDF
	}
	return ""
}

// CheckUpload refuses empty, oversized and unsupported files.
func CheckUpload(u *Upload, kind, label string, maxBytes int64, allowPDF bool) (store.NewDocument, error) {
	if len(u.Data) == 0 {
		return store.NewDocument{}, refuse("%s: the file is empty.", label)
	}
	if int64(len(u.Data)) > maxBytes {
		return store.NewDocument{}, refuse("%s: the file is too large (max %d MB).", label, maxBytes/(1024*1024))
	}
	contentType := SniffType(u.Data)
	allowed := contentType == TypeJPEG || contentType == TypePNG || contentType == TypeWebP ||
		(allowPDF && contentType == TypePDF)
	if !allowed {
		kinds := "a JPEG, PNG or WebP photo"
		if allowPDF {
			kinds += " or a PDF"
		}
		return store.NewDocument{}, refuse("%s: please upload %s.", label, kinds)
	}
	return store.NewDocument{Kind: kind, ContentType: contentType, Data: u.Data}, nil
}
