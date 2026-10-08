package api

import (
	"reflect"
	"sync"
	"testing"

	"github.com/coder/websocket"

	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/realtime"
)

var badge = map[string]any{"name": doctorName, "role": "doctor"}

// startCall has a patient request help and the signed-in doctor accept.
func startCall(c *client) (id, patientToken, doctorToken string) {
	patient := c.post("/api/v1/consultations", obj{"patient_name": "Ram"}).m()
	doctor := c.acceptCall(patient)
	return consultationID(patient), patient["token"].(string), doctor["token"].(string)
}

func TestQueueSocketRequiresVerification(t *testing.T) {
	c := newEnv(t).client()
	if code := closeCode(t, c.mustDial("/ws/doctors")); code != realtime.CloseNotVerified {
		t.Fatalf("signed out: close code %d", code)
	}
	c.makeProfessional("new@example.com", "New", "doctor", "pending")
	if code := closeCode(t, c.mustDial("/ws/doctors")); code != realtime.CloseNotVerified {
		t.Fatalf("pending: close code %d", code)
	}
}

func TestQueueSnapshotAndUpdates(t *testing.T) {
	c := doctorClient(t)
	queue := c.mustDial("/ws/doctors")
	if got := receive(t, queue); !reflect.DeepEqual(got, obj{"type": "queue", "consultations": []any{}}) {
		t.Fatalf("%v", got)
	}
	ticket := c.post("/api/v1/consultations", obj{"patient_name": "Gita"}).m()
	update := receive(t, queue)
	waiting := update["consultations"].([]any)
	if update["type"] != "queue" || len(waiting) != 1 || waiting[0].(map[string]any)["patient_name"] != "Gita" {
		t.Fatalf("%v", update)
	}
	c.acceptCall(ticket)
	if after := receive(t, queue); len(after["consultations"].([]any)) != 0 {
		t.Fatalf("an accepted call leaves everyone's queue: %v", after)
	}
}

func TestQueueUpdatesArriveInOrder(t *testing.T) {
	c := doctorClient(t)
	queue := c.mustDial("/ws/doctors")
	receive(t, queue)
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.e.client().post("/api/v1/consultations", obj{"patient_name": "P"})
		}()
	}
	wg.Wait()
	// Each snapshot is at least as long as the one before.
	last := 0
	for range 10 {
		n := len(receive(t, queue)["consultations"].([]any))
		if n < last {
			t.Fatalf("a stale snapshot arrived: %d after %d", n, last)
		}
		last = n
	}
	if last != 10 {
		t.Fatalf("final snapshot has %d", last)
	}
}

func TestBadTokenIsRejected(t *testing.T) {
	c := doctorClient(t)
	id, _, _ := startCall(c)
	if code := closeCode(t, c.mustDial("/ws/consultations/"+id+"?token=wrong")); code != realtime.CloseUnauthorized {
		t.Fatalf("close code %d", code)
	}
	if code := closeCode(t, c.mustDial("/ws/consultations/"+id)); code != realtime.CloseUnauthorized {
		t.Fatalf("no token: close code %d", code)
	}
}

func TestUnknownConsultationIsRejected(t *testing.T) {
	c := newEnv(t).client()
	if code := closeCode(t, c.mustDial("/ws/consultations/missing?token=x")); code != realtime.CloseUnauthorized {
		t.Fatalf("close code %d", code)
	}
}

func TestSignalsRelayBetweenPatientAndDoctor(t *testing.T) {
	c := doctorClient(t)
	id, patientToken, doctorToken := startCall(c)

	p := c.mustDial("/ws/consultations/" + id + "?token=" + patientToken)
	want := obj{"type": "joined", "role": "patient", "peer_present": false, "professional": badge}
	if got := receive(t, p); !reflect.DeepEqual(got, want) {
		t.Fatalf("%v", got)
	}

	d := c.mustDial("/ws/consultations/" + id + "?token=" + doctorToken)
	if got := receive(t, d); got["peer_present"] != true || got["role"] != "doctor" {
		t.Fatalf("%v", got)
	}
	// The one already in the room is told, and learns who accepted, for the
	// "Verified Doctor" badge. They make the offer.
	if got := receive(t, p); !reflect.DeepEqual(got, obj{"type": "peer-joined", "role": "doctor", "professional": badge}) {
		t.Fatalf("%v", got)
	}

	send(t, p, obj{"type": "offer", "payload": obj{"sdp": "v=0", "type": "offer"}})
	offer := receive(t, d)
	if offer["type"] != "offer" || offer["from"] != "patient" || offer["payload"].(map[string]any)["sdp"] != "v=0" {
		t.Fatalf("%v", offer)
	}
	send(t, d, obj{"type": "answer", "payload": obj{"sdp": "v=0", "type": "answer"}})
	if got := receive(t, p); got["type"] != "answer" || got["from"] != "doctor" {
		t.Fatalf("%v", got)
	}
	send(t, d, obj{"type": "ice-candidate", "payload": obj{"candidate": "c1"}})
	if got := receive(t, p); got["type"] != "ice-candidate" {
		t.Fatalf("%v", got)
	}

	_ = d.Close(websocket.StatusNormalClosure, "")
	if got := receive(t, p); !reflect.DeepEqual(got, obj{"type": "peer-left", "role": "doctor"}) {
		t.Fatalf("%v", got)
	}
}

func TestUnknownSignalIsNotRelayed(t *testing.T) {
	c := doctorClient(t)
	id, patientToken, doctorToken := startCall(c)
	p := c.mustDial("/ws/consultations/" + id + "?token=" + patientToken)
	receive(t, p) // joined
	d := c.mustDial("/ws/consultations/" + id + "?token=" + doctorToken)
	receive(t, d) // joined
	receive(t, p) // peer-joined

	unsupported := obj{"type": "error", "detail": "unsupported signal"}
	for _, bad := range []any{
		obj{"type": "rm -rf"},
		obj{"type": "offer", "payload": "not an object"},
		[]int{1, 2},
	} {
		send(t, p, bad)
		if got := receive(t, p); !reflect.DeepEqual(got, unsupported) {
			t.Fatalf("%v: %v", bad, got)
		}
	}
	send(t, p, obj{"type": "hangup"})
	if got := receive(t, d); !reflect.DeepEqual(got, obj{"type": "hangup", "payload": nil, "from": "patient"}) {
		t.Fatalf("%v", got)
	}
}

// A page refresh must not lock the participant out of their own call.
func TestReconnectReplacesTheOldSocket(t *testing.T) {
	c := doctorClient(t)
	id, patientToken, doctorToken := startCall(c)
	d := c.mustDial("/ws/consultations/" + id + "?token=" + doctorToken)
	receive(t, d) // joined
	url := "/ws/consultations/" + id + "?token=" + patientToken

	first := c.mustDial(url)
	receive(t, first)
	if got := receive(t, d); got["type"] != "peer-joined" {
		t.Fatalf("%v", got)
	}
	second := c.mustDial(url)
	if got := receive(t, second); got["type"] != "joined" || got["peer_present"] != true {
		t.Fatalf("%v", got)
	}
	if code := closeCode(t, first); code != realtime.CloseReplaced {
		t.Fatalf("close code %d", code)
	}
	// The doctor hears the patient came back (and offers again), never that
	// they left.
	if got := receive(t, d); got["type"] != "peer-joined" || got["role"] != "patient" {
		t.Fatalf("%v", got)
	}
	send(t, second, obj{"type": "hangup"})
	if got := receive(t, d); got["type"] != "hangup" {
		t.Fatalf("the replaced socket's leaving must not announce peer-left: %v", got)
	}
}

func TestAnEndedCallCannotBeRejoined(t *testing.T) {
	c := doctorClient(t)
	id, patientToken, _ := startCall(c)
	c.post("/api/v1/consultations/"+id+"/end", obj{"token": patientToken})
	if code := closeCode(t, c.mustDial("/ws/consultations/"+id+"?token="+patientToken)); code != realtime.CloseUnauthorized {
		t.Fatalf("close code %d", code)
	}
}

func TestShutdownTellsSocketsToReconnect(t *testing.T) {
	c := doctorClient(t)
	queue := c.mustDial("/ws/doctors")
	receive(t, queue)
	c.e.server.Shutdown()
	if code := closeCode(t, queue); code != websocket.StatusGoingAway {
		t.Fatalf("close code %d", code)
	}
}
