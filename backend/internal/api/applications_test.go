package api

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/config"
)

var (
	pngFile  = append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 64)...)
	jpegFile = append([]byte("\xff\xd8\xff\xe0"), make([]byte, 64)...)
	pdfFile  = append([]byte("%PDF-1.7\n"), make([]byte, 64)...)
	script   = []byte("<script>alert(1)</script>")
)

type upload struct {
	filename string
	data     []byte
}

type form struct {
	data  map[string]string
	files map[string]upload
}

func doctorForm(overrides map[string]string) form {
	data := map[string]string{
		"role": "doctor", "full_name": "Dr. Anita Karki", "phone": "+977 9841234567",
		"citizenship_number": "27-01-71-12345", "citizenship_district": "Kathmandu",
		"council_number": "12345", "consent": "true",
	}
	for k, v := range overrides {
		data[k] = v
	}
	return form{data: data, files: map[string]upload{
		"citizenship_front":   {"front.png", pngFile},
		"citizenship_back":    {"back.jpg", jpegFile},
		"council_certificate": {"nmc.pdf", pdfFile},
	}}
}

func studentForm() form {
	return form{
		data: map[string]string{
			"role": "student", "full_name": "Bikash Thapa", "phone": "9800000000",
			"citizenship_number": "45-02-75-00001", "citizenship_district": "Kaski",
			"institution": "Manipal College of Medical Sciences", "recommender_name": "Dr. Anita Karki",
			"recommender_nmc": "12345", "consent": "true",
		},
		files: map[string]upload{
			"citizenship_front":     {"front.png", pngFile},
			"citizenship_back":      {"back.png", pngFile},
			"recommendation_letter": {"letter.pdf", pdfFile},
		},
	}
}

func (c *client) apply(f form) *response {
	c.e.t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	for k, v := range f.data {
		_ = w.WriteField(k, v)
	}
	for field, file := range f.files {
		part, _ := w.CreateFormFile(field, file.filename)
		_, _ = part.Write(file.data)
	}
	_ = w.Close()
	return c.request(http.MethodPost, "/api/v1/applications", &body, w.FormDataContentType())
}

func (c *client) asAdmin() {
	c.clearCookies()
	c.signIn(adminEmail, "Admin")
}

func actions(card map[string]any) string {
	var out []string
	for _, h := range card["history"].([]any) {
		out = append(out, h.(map[string]any)["action"].(string))
	}
	return strings.Join(out, ",")
}

func TestMustBeSignedInToApply(t *testing.T) {
	if r := newEnv(t).client().apply(doctorForm(nil)); r.status != http.StatusUnauthorized {
		t.Fatalf("%d", r.status)
	}
}

func TestDoctorAppliesAndSeesPending(t *testing.T) {
	c := newEnv(t).client()
	c.signIn("anita@example.com", "Anita")
	if r := c.get("/api/v1/applications/me"); r.status != http.StatusOK || string(bytes.TrimSpace(r.body)) != "null" {
		t.Fatalf("before applying: %d %s", r.status, r.body)
	}
	r := c.apply(doctorForm(nil))
	if r.status != http.StatusCreated {
		t.Fatalf("%d %s", r.status, r.body)
	}
	body := r.m()
	if body["status"] != "pending" || body["council_number"] != "12345" || body["reviewed_at"] != nil ||
		body["institution"] != nil {
		t.Fatalf("%v", body)
	}
	var kinds []string
	byKind := map[string]string{}
	for _, d := range body["documents"].([]any) {
		doc := d.(map[string]any)
		kinds = append(kinds, doc["kind"].(string))
		byKind[doc["kind"].(string)] = doc["content_type"].(string)
	}
	sort.Strings(kinds)
	if strings.Join(kinds, ",") != "citizenship_back,citizenship_front,council_certificate" {
		t.Fatalf("%v", kinds)
	}
	// Types come from the bytes.
	if byKind["council_certificate"] != "application/pdf" || byKind["citizenship_back"] != "image/jpeg" {
		t.Fatalf("%v", byKind)
	}

	me := c.get("/api/v1/auth/me").m()["user"].(map[string]any)
	if me["role"] != "doctor" || me["verification"].(map[string]any)["status"] != "pending" {
		t.Fatalf("%v", me)
	}
	if mine := c.get("/api/v1/applications/me").m(); mine["id"] != body["id"] {
		t.Fatalf("%v", mine)
	}
}

func TestRegisteredProfessionsNeedTheirCouncilNumberAndCertificate(t *testing.T) {
	for role, council := range map[string]string{
		"pharmacist": "Nepal Pharmacy Council",
		"nurse":      "Nepal Nursing Council",
		"paramedic":  "Nepal Health Professional Council",
	} {
		c := newEnv(t).client()
		c.signIn(role+"@example.com", "Test User")
		missingNumber := c.apply(doctorForm(map[string]string{"role": role, "council_number": ""}))
		if missingNumber.status != http.StatusUnprocessableEntity || !strings.Contains(missingNumber.detail(), council) {
			t.Fatalf("%s: %d %s", role, missingNumber.status, missingNumber.body)
		}
		f := doctorForm(map[string]string{"role": role})
		delete(f.files, "council_certificate")
		missingCertificate := c.apply(f)
		if missingCertificate.status != http.StatusUnprocessableEntity || !strings.Contains(missingCertificate.detail(), "certificate") {
			t.Fatalf("%s: %d %s", role, missingCertificate.status, missingCertificate.body)
		}
		submitted := c.apply(doctorForm(map[string]string{"role": role}))
		if submitted.status != http.StatusCreated || submitted.m()["role"] != role || submitted.m()["council_number"] == nil {
			t.Fatalf("%s: %d %s", role, submitted.status, submitted.body)
		}
	}
}

func TestStudentNeedsARecommendation(t *testing.T) {
	c := newEnv(t).client()
	c.signIn("student@example.com", "Student")
	f := studentForm()
	delete(f.files, "recommendation_letter")
	if r := c.apply(f); !strings.Contains(r.detail(), "recommendation") {
		t.Fatalf("%d %s", r.status, r.body)
	}
	r := c.apply(studentForm())
	if r.status != http.StatusCreated || r.m()["recommender_nmc"] != "12345" || r.m()["council_number"] != nil {
		t.Fatalf("%d %s", r.status, r.body)
	}
}

func TestFieldValidation(t *testing.T) {
	cases := []struct{ field, value, message string }{
		{"phone", "abc", "phone"},
		{"citizenship_number", "", "citizenship number"},
		{"consent", "false", "agree"},
		{"consent", "maybe", "agree"},
		{"role", "admin", "profession"},
		{"full_name", strings.Repeat("n", 161), "160 characters"},
	}
	for _, tc := range cases {
		c := newEnv(t).client()
		c.signIn("x@example.com", "X")
		r := c.apply(doctorForm(map[string]string{tc.field: tc.value}))
		if r.status != http.StatusUnprocessableEntity || !strings.Contains(r.detail(), tc.message) {
			t.Errorf("%s=%q: %d %s", tc.field, tc.value, r.status, r.body)
		}
	}
}

func TestADisguisedFileIsRefused(t *testing.T) {
	c := newEnv(t).client()
	c.signIn("x@example.com", "X")
	f := doctorForm(nil)
	f.files["citizenship_front"] = upload{"front.jpg", script}
	r := c.apply(f)
	if r.status != http.StatusUnprocessableEntity || !strings.Contains(r.detail(), "Citizenship (front)") {
		t.Fatalf("%d %s", r.status, r.body)
	}
}

func TestAnOversizedFileIsRefused(t *testing.T) {
	c := newEnv(t, withConfig(func(cfg *config.Config) { cfg.MaxUploadBytes = 1024 * 1024 })).client()
	c.signIn("x@example.com", "X")
	f := doctorForm(nil)
	f.files["citizenship_back"] = upload{"big.png", append(append([]byte{}, pngFile...), make([]byte, 1024*1024)...)}
	if r := c.apply(f); !strings.Contains(r.detail(), "too large (max 1 MB)") {
		t.Fatalf("%d %s", r.status, r.body)
	}
}

func TestTheSelfieMustBeAPhoto(t *testing.T) {
	c := newEnv(t).client()
	c.signIn("x@example.com", "X")
	f := doctorForm(nil)
	f.files["selfie"] = upload{"selfie.pdf", pdfFile}
	if r := c.apply(f); !strings.Contains(r.detail(), "photo") {
		t.Fatalf("%d %s", r.status, r.body)
	}
	f.files["selfie"] = upload{"selfie.png", pngFile}
	if r := c.apply(f); r.status != http.StatusCreated || len(r.m()["documents"].([]any)) != 4 {
		t.Fatalf("%d %s", r.status, r.body)
	}
}

func TestBothCitizenshipSidesAreRequired(t *testing.T) {
	c := newEnv(t).client()
	c.signIn("x@example.com", "X")
	f := doctorForm(nil)
	delete(f.files, "citizenship_back")
	r := c.apply(f)
	if r.status != http.StatusUnprocessableEntity || !strings.Contains(r.detail(), "both sides") {
		t.Fatalf("%d %s", r.status, r.body)
	}
	// An empty file input arrives with an empty filename: not provided.
	f.files["citizenship_back"] = upload{"", nil}
	if r := c.apply(f); !strings.Contains(r.detail(), "both sides") {
		t.Fatalf("%d %s", r.status, r.body)
	}
}

func TestAdminOnly(t *testing.T) {
	c := newEnv(t).client()
	c.signIn("anita@example.com", "Anita")
	if r := c.get("/api/v1/admin/applications"); r.status != http.StatusForbidden || r.detail() != "Admins only." {
		t.Fatalf("%d %s", r.status, r.body)
	}
	c.clearCookies()
	if r := c.get("/api/v1/admin/applications"); r.status != http.StatusUnauthorized {
		t.Fatalf("%d", r.status)
	}
}

func TestAdminReviewsDocumentsAndApproves(t *testing.T) {
	c := newEnv(t).client()
	c.signIn("anita@example.com", "Anita")
	applicationID := c.apply(doctorForm(nil)).m()["id"]

	c.asAdmin()
	queue := c.get("/api/v1/admin/applications").list()
	if len(queue) != 1 || queue[0]["id"] != applicationID {
		t.Fatalf("%v", queue)
	}
	card := queue[0]
	if card["user_email"] != "anita@example.com" || card["user_name"] != "Anita" || actions(card) != "submitted" {
		t.Fatalf("%v", card)
	}

	var frontID float64
	for _, d := range card["documents"].([]any) {
		if doc := d.(map[string]any); doc["kind"] == "citizenship_front" {
			frontID = doc["id"].(float64)
		}
	}
	image := c.get(fmt.Sprintf("/api/v1/admin/documents/%d", int(frontID)))
	if !bytes.Equal(image.body, pngFile) || image.header.Get("Content-Type") != "image/png" ||
		image.header.Get("X-Content-Type-Options") != "nosniff" ||
		!strings.Contains(image.header.Get("Cache-Control"), "no-store") {
		t.Fatalf("%d %v", image.status, image.header)
	}
	if r := c.get("/api/v1/admin/documents/99999"); r.status != http.StatusNotFound || r.detail() != "Document not found." {
		t.Fatalf("%d %s", r.status, r.body)
	}
	if r := c.get("/api/v1/admin/documents/abc"); r.status != http.StatusUnprocessableEntity {
		t.Fatalf("%d", r.status)
	}

	id := int(applicationID.(float64))
	approved := c.post(fmt.Sprintf("/api/v1/admin/applications/%d/approve", id), nil).m()
	if approved["status"] != "approved" || actions(approved) != "submitted,approved" || approved["reviewed_at"] == nil {
		t.Fatalf("%v", approved)
	}
	if r := c.post(fmt.Sprintf("/api/v1/admin/applications/%d/approve", id), nil); r.status != http.StatusConflict {
		t.Fatalf("%d", r.status)
	}
	if pending := c.get("/api/v1/admin/applications").list(); len(pending) != 0 {
		t.Fatalf("%v", pending)
	}
	approvedList := c.get("/api/v1/admin/applications?status=approved").list()
	if len(approvedList) != 1 || approvedList[0]["id"] != applicationID {
		t.Fatalf("%v", approvedList)
	}
	if all := c.get("/api/v1/admin/applications?status=all").list(); len(all) != 1 {
		t.Fatalf("%v", all)
	}
	if r := c.get("/api/v1/admin/applications?status=bogus"); r.status != http.StatusUnprocessableEntity {
		t.Fatalf("%d", r.status)
	}
	if r := c.post("/api/v1/admin/applications/99999/approve", nil); r.status != http.StatusNotFound || r.detail() != "Application not found." {
		t.Fatalf("%d %s", r.status, r.body)
	}
}

func TestApplicantCannotReadDocuments(t *testing.T) {
	c := newEnv(t).client()
	c.signIn("anita@example.com", "Anita")
	docID := c.apply(doctorForm(nil)).m()["documents"].([]any)[0].(map[string]any)["id"].(float64)
	if r := c.get(fmt.Sprintf("/api/v1/admin/documents/%d", int(docID))); r.status != http.StatusForbidden {
		t.Fatalf("%d", r.status)
	}
}

func TestRejectedApplicantFixesAndResubmits(t *testing.T) {
	c := newEnv(t).client()
	c.signIn("anita@example.com", "Anita")
	applicationID := int(c.apply(doctorForm(nil)).m()["id"].(float64))

	c.asAdmin()
	rejected := c.post(fmt.Sprintf("/api/v1/admin/applications/%d/reject", applicationID), obj{"reason": " Citizenship photo is blurry. "})
	if rejected.m()["status"] != "rejected" || rejected.m()["rejection_reason"] != "Citizenship photo is blurry." {
		t.Fatalf("%s", rejected.body)
	}
	if again := c.post(fmt.Sprintf("/api/v1/admin/applications/%d/reject", applicationID), obj{"reason": "Again."}); again.status != http.StatusConflict {
		t.Fatalf("%d", again.status)
	}

	c.clearCookies()
	c.signIn("anita@example.com", "Anita")
	me := c.get("/api/v1/auth/me").m()["user"].(map[string]any)
	if me["verification"].(map[string]any)["rejection_reason"] != "Citizenship photo is blurry." {
		t.Fatalf("%v", me)
	}
	again := c.apply(doctorForm(map[string]string{"citizenship_district": "Lalitpur"}))
	body := again.m()
	if again.status != http.StatusCreated || int(body["id"].(float64)) != applicationID || body["status"] != "pending" ||
		body["rejection_reason"] != nil || len(body["documents"].([]any)) != 3 || body["citizenship_district"] != "Lalitpur" {
		t.Fatalf("%d %v", again.status, body)
	}

	c.asAdmin()
	card := c.get("/api/v1/admin/applications").list()[0]
	if actions(card) != "submitted,rejected,resubmitted" {
		t.Fatalf("%v", card["history"])
	}
	history := card["history"].([]any)
	if reason := history[1].(map[string]any)["reason"]; reason != "Citizenship photo is blurry." {
		t.Fatalf("%v", reason)
	}
}

func TestRejectNeedsAReason(t *testing.T) {
	c := newEnv(t).client()
	c.signIn("anita@example.com", "Anita")
	applicationID := int(c.apply(doctorForm(nil)).m()["id"].(float64))
	c.asAdmin()
	for _, reason := range []string{"", "  ", "no", strings.Repeat("r", 501)} {
		r := c.post(fmt.Sprintf("/api/v1/admin/applications/%d/reject", applicationID), obj{"reason": reason})
		if r.status != http.StatusUnprocessableEntity {
			t.Errorf("%q: %d", reason, r.status)
		}
	}
}

func TestApprovedCannotResubmitOrSwitchRole(t *testing.T) {
	c := newEnv(t).client()
	c.signIn("anita@example.com", "Anita")
	applicationID := int(c.apply(doctorForm(nil)).m()["id"].(float64))
	c.asAdmin()
	c.post(fmt.Sprintf("/api/v1/admin/applications/%d/approve", applicationID), nil)

	c.clearCookies()
	c.signIn("anita@example.com", "Anita")
	if r := c.apply(doctorForm(nil)); r.status != http.StatusConflict || r.detail() != "You are already verified." {
		t.Fatalf("%d %s", r.status, r.body)
	}
	if r := c.post("/api/v1/auth/role", obj{"role": "pharmacist"}); r.status != http.StatusConflict {
		t.Fatalf("%d", r.status)
	}
	if r := c.post("/api/v1/auth/role", obj{"role": "doctor"}); r.status != http.StatusOK {
		t.Fatalf("keeping the verified role is fine: %d", r.status)
	}
	verification := c.get("/api/v1/auth/me").m()["user"].(map[string]any)["verification"]
	if !reflect.DeepEqual(verification, map[string]any{
		"role": "doctor", "status": "approved", "rejection_reason": nil, "full_name": "Dr. Anita Karki",
	}) {
		t.Fatalf("%v", verification)
	}
}

func TestAdminCanRevokeAnApproval(t *testing.T) {
	c := newEnv(t).client()
	c.signIn("anita@example.com", "Anita")
	applicationID := int(c.apply(doctorForm(nil)).m()["id"].(float64))
	c.asAdmin()
	c.post(fmt.Sprintf("/api/v1/admin/applications/%d/approve", applicationID), nil)
	revoked := c.post(fmt.Sprintf("/api/v1/admin/applications/%d/reject", applicationID), obj{"reason": "NMC registration not found."})
	if revoked.m()["status"] != "rejected" {
		t.Fatalf("%s", revoked.body)
	}
}
