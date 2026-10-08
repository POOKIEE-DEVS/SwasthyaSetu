package api

import (
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/store"
)

// The same client acts as patient (no login needed) and as a verified
// professional (signed in), as in the Python tests.

func (c *client) requestProfessional(name string) map[string]any {
	c.e.t.Helper()
	r := c.post("/api/v1/consultations", obj{"patient_name": name, "summary": "Fever for two days"})
	if r.status != http.StatusCreated {
		c.e.t.Fatalf("request: %d %s", r.status, r.body)
	}
	return r.m()
}

func consultationID(ticket map[string]any) string {
	return ticket["consultation"].(map[string]any)["id"].(string)
}

func (c *client) acceptCall(ticket map[string]any) map[string]any {
	c.e.t.Helper()
	r := c.post("/api/v1/consultations/"+consultationID(ticket)+"/accept", nil)
	if r.status != http.StatusOK {
		c.e.t.Fatalf("accept: %d %s", r.status, r.body)
	}
	return r.m()
}

func (c *client) endCall(ticket map[string]any) {
	c.e.t.Helper()
	r := c.post("/api/v1/consultations/"+consultationID(ticket)+"/end", obj{"token": ticket["token"]})
	if r.status != http.StatusNoContent {
		c.e.t.Fatalf("end: %d %s", r.status, r.body)
	}
}

func doctorClient(t *testing.T) *client {
	c := newEnv(t).client()
	c.makeDoctor()
	return c
}

func TestPatientRequestReturnsATicket(t *testing.T) {
	ticket := doctorClient(t).requestProfessional("Sita")
	consultation := ticket["consultation"].(map[string]any)
	servers := ticket["ice_servers"].([]any)
	firstURL := servers[0].(map[string]any)["urls"].([]any)[0].(string)
	if ticket["role"] != "patient" || ticket["token"] == "" || consultation["status"] != "waiting" ||
		consultation["professional"] != nil || !strings.HasPrefix(firstURL, "stun:") {
		t.Fatalf("%v", ticket)
	}
}

func TestWaitingListNeverExposesTokens(t *testing.T) {
	c := doctorClient(t)
	c.requestProfessional("Sita")
	waiting := c.get("/api/v1/consultations").list()
	if len(waiting) != 1 || waiting[0]["patient_name"] != "Sita" || waiting[0]["summary"] != "Fever for two days" {
		t.Fatalf("%v", waiting)
	}
	for key := range waiting[0] {
		if strings.Contains(key, "token") {
			t.Fatalf("token exposed: %v", waiting[0])
		}
	}
}

func TestDoctorAccepts(t *testing.T) {
	c := doctorClient(t)
	ticket := c.requestProfessional("Sita")
	body := c.acceptCall(ticket)
	consultation := body["consultation"].(map[string]any)
	if body["role"] != "doctor" || body["token"] == ticket["token"] || consultation["summary"] != "Fever for two days" ||
		consultation["status"] != "active" {
		t.Fatalf("%v", body)
	}
	// The badge the patient will see.
	if !reflect.DeepEqual(consultation["professional"], map[string]any{"name": doctorName, "role": "doctor"}) {
		t.Fatalf("%v", consultation["professional"])
	}
	if waiting := c.get("/api/v1/consultations").list(); len(waiting) != 0 {
		t.Fatalf("%v", waiting)
	}
}

func TestSecondAcceptConflicts(t *testing.T) {
	c := doctorClient(t)
	ticket := c.requestProfessional("Sita")
	c.acceptCall(ticket)
	r := c.post("/api/v1/consultations/"+consultationID(ticket)+"/accept", nil)
	if r.status != http.StatusConflict || r.detail() != "Another doctor has already taken this request." {
		t.Fatalf("%d %s", r.status, r.body)
	}
}

func TestAcceptUnknownIs404(t *testing.T) {
	r := doctorClient(t).post("/api/v1/consultations/nope/accept", nil)
	if r.status != http.StatusNotFound || r.detail() != "This request no longer exists." {
		t.Fatalf("%d %s", r.status, r.body)
	}
}

func TestEndRequiresAParticipantToken(t *testing.T) {
	c := doctorClient(t)
	ticket := c.requestProfessional("Sita")
	id := consultationID(ticket)
	if r := c.post("/api/v1/consultations/"+id+"/end", obj{"token": "x"}); r.status != http.StatusNotFound {
		t.Fatalf("wrong token: %d", r.status)
	}
	if r := c.post("/api/v1/consultations/"+id+"/end", obj{}); r.status != http.StatusUnprocessableEntity {
		t.Fatalf("no token: %d", r.status)
	}
	c.endCall(ticket)
	if waiting := c.get("/api/v1/consultations").list(); len(waiting) != 0 {
		t.Fatalf("%v", waiting)
	}
}

func TestPatientNeedsNoLogin(t *testing.T) {
	c := doctorClient(t)
	c.clearCookies()
	if c.requestProfessional("Sita")["role"] != "patient" {
		t.Fatal("not a patient ticket")
	}
}

func TestBadPatientRequests(t *testing.T) {
	c := newEnv(t).client()
	for _, body := range []obj{
		{"patient_name": ""}, {"patient_name": "   "}, {},
		{"patient_name": strings.Repeat("n", 61)},
		{"patient_name": "Ram", "summary": strings.Repeat("s", 6001)},
	} {
		if r := c.post("/api/v1/consultations", body); r.status != http.StatusUnprocessableEntity {
			t.Errorf("%v: %d", body, r.status)
		}
	}
	if r := c.post("/api/v1/consultations", obj{"patient_name": strings.Repeat("न", 60), "summary": nil}); r.status != http.StatusCreated {
		t.Fatalf("%d %s", r.status, r.body)
	}
}

func TestSignedOutCannotSeeOrAccept(t *testing.T) {
	c := doctorClient(t)
	ticket := c.requestProfessional("Sita")
	c.clearCookies()
	if r := c.get("/api/v1/consultations"); r.status != http.StatusUnauthorized {
		t.Fatalf("%d", r.status)
	}
	if r := c.post("/api/v1/consultations/"+consultationID(ticket)+"/accept", nil); r.status != http.StatusUnauthorized {
		t.Fatalf("%d", r.status)
	}
}

func TestPatientAccountCannotAccept(t *testing.T) {
	c := doctorClient(t)
	ticket := c.requestProfessional("Sita")
	c.clearCookies()
	c.signIn("patient@example.com", "Patient")
	c.post("/api/v1/auth/role", obj{"role": "patient"})
	if r := c.get("/api/v1/consultations"); r.status != http.StatusForbidden {
		t.Fatalf("%d", r.status)
	}
	if r := c.post("/api/v1/consultations/"+consultationID(ticket)+"/accept", nil); r.status != http.StatusForbidden {
		t.Fatalf("%d", r.status)
	}
}

func TestUnverifiedProfessionalCannotAccept(t *testing.T) {
	for _, status := range []string{store.StatusPending, store.StatusRejected} {
		c := doctorClient(t)
		ticket := c.requestProfessional("Sita")
		c.clearCookies()
		c.makeProfessional("new@example.com", "New Doctor", "doctor", status)
		r := c.post("/api/v1/consultations/"+consultationID(ticket)+"/accept", nil)
		if r.status != http.StatusForbidden || !strings.Contains(r.detail(), "verified") {
			t.Fatalf("%s: %d %s", status, r.status, r.body)
		}
	}
}

func TestARoleSwitchAwayLosesAccess(t *testing.T) {
	c := doctorClient(t)
	// A verified doctor whose account role no longer matches is not verified.
	if _, err := c.e.db.ExecContext(t.Context(), `UPDATE users SET role = 'patient'`); err != nil {
		t.Fatal(err)
	}
	if r := c.get("/api/v1/consultations"); r.status != http.StatusForbidden {
		t.Fatalf("%d", r.status)
	}
}

func TestEveryVerifiedProfessionCanAccept(t *testing.T) {
	for _, role := range []string{"pharmacist", "nurse", "paramedic", "student"} {
		c := doctorClient(t)
		ticket := c.requestProfessional("Sita")
		c.clearCookies()
		c.makeProfessional(role+"@example.com", "Hari", role, store.StatusApproved)
		accepted := c.acceptCall(ticket)
		badge := accepted["consultation"].(map[string]any)["professional"]
		if !reflect.DeepEqual(badge, map[string]any{"name": "Hari", "role": role}) {
			t.Fatalf("%s: %v", role, badge)
		}
	}
}

// --- Help record --------------------------------------------------------------

const helped = "/api/v1/consultations/helped"

func TestAFinishedCallIsCountedWithThePatientsName(t *testing.T) {
	c := doctorClient(t)
	if got := c.get(helped).m(); !reflect.DeepEqual(got, obj{"count": 0.0, "people": []any{}}) {
		t.Fatalf("%v", got)
	}
	doctor := c.acceptCall(c.requestProfessional("Sita"))
	c.endCall(doctor)

	record := c.get(helped).m()
	people := record["people"].([]any)
	if record["count"] != 1.0 || len(people) != 1 {
		t.Fatalf("%v", record)
	}
	person := people[0].(map[string]any)
	if person["patient_name"] != "Sita" || person["duration_seconds"].(float64) < 0 || person["started_at"].(float64) <= 0 {
		t.Fatalf("%v", person)
	}
}

func TestBothSidesEndingCountsOnce(t *testing.T) {
	c := doctorClient(t)
	patient := c.requestProfessional("Sita")
	doctor := c.acceptCall(patient)
	c.endCall(doctor)
	c.endCall(patient) // the other side reports the end too
	if count := c.get(helped).m()["count"]; count != 1.0 {
		t.Fatalf("%v", count)
	}
}

func TestAPatientWhoGivesUpWaitingIsNotCounted(t *testing.T) {
	c := doctorClient(t)
	c.endCall(c.requestProfessional("Sita"))
	if count := c.get(helped).m()["count"]; count != 0.0 {
		t.Fatalf("%v", count)
	}
}

func TestNewestFirstAndCountedPerCall(t *testing.T) {
	c := doctorClient(t)
	for _, name := range []string{"Sita", "Ram", "Gita"} {
		c.endCall(c.acceptCall(c.requestProfessional(name)))
	}
	record := c.get(helped).m()
	var names []string
	for _, p := range record["people"].([]any) {
		names = append(names, p.(map[string]any)["patient_name"].(string))
	}
	if record["count"] != 3.0 || strings.Join(names, ",") != "Gita,Ram,Sita" {
		t.Fatalf("%v", record)
	}
}

func TestEachProfessionalSeesOnlyTheirOwn(t *testing.T) {
	c := doctorClient(t)
	c.endCall(c.acceptCall(c.requestProfessional("Sita")))
	c.clearCookies()
	c.makeProfessional("nurse@example.com", "Maya", "nurse", store.StatusApproved)
	if got := c.get(helped).m(); !reflect.DeepEqual(got, obj{"count": 0.0, "people": []any{}}) {
		t.Fatalf("%v", got)
	}
}

func TestOnlyVerifiedProfessionalsHaveARecord(t *testing.T) {
	c := newEnv(t).client()
	if r := c.get(helped); r.status != http.StatusUnauthorized {
		t.Fatalf("%d", r.status)
	}
	c.signIn("patient@example.com", "Patient")
	c.post("/api/v1/auth/role", obj{"role": "patient"})
	if r := c.get(helped); r.status != http.StatusForbidden {
		t.Fatalf("%d", r.status)
	}
	c.clearCookies()
	c.makeProfessional("new@example.com", "New", "doctor", store.StatusPending)
	if r := c.get(helped); r.status != http.StatusForbidden {
		t.Fatalf("%d", r.status)
	}
}

func TestHangingUpStillWorksWhenTheDatabaseIsDown(t *testing.T) {
	c := doctorClient(t)
	doctor := c.acceptCall(c.requestProfessional("Sita"))
	c.e.db.SetReady(false)
	c.endCall(doctor) // still 204
	c.e.db.SetReady(true)
	if count := c.get(helped).m()["count"]; count != 0.0 {
		t.Fatalf("%v", count)
	}
}
