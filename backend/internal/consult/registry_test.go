package consult

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func text(s string) *string { return &s }

func TestCreateTrimsAndQueues(t *testing.T) {
	r := NewRegistry(time.Hour)
	c := r.Create("  Sita ", text("  Fever for two days  "))
	if c.PatientName != "Sita" || *c.Summary != "Fever for two days" || c.Status != Waiting {
		t.Fatalf("created %+v", c)
	}
	if blank := r.Create("Ram", text("   ")); blank.Summary != nil {
		t.Fatal("a blank summary should be dropped")
	}
	if len(c.ID) != 11 || len(c.PatientToken) != 32 || c.DoctorToken != "" {
		t.Fatalf("id %q token %q", c.ID, c.PatientToken)
	}
	waiting := r.Waiting()
	if len(waiting) != 2 || waiting[0].ID != c.ID {
		t.Fatalf("queue %+v", waiting)
	}
}

func TestAcceptAndEnd(t *testing.T) {
	r := NewRegistry(time.Hour)
	c := r.Create("Sita", nil)
	badge := Badge{Name: "Dr. Anita Karki", Role: "doctor"}

	accepted, err := r.Accept(c.ID, badge, 7)
	if err != nil || accepted.Status != Active || *accepted.Professional != badge || accepted.DoctorToken == "" {
		t.Fatalf("accepted %+v, %v", accepted, err)
	}
	if _, err := r.Accept(c.ID, badge, 8); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("second accept: %v", err)
	}
	if _, err := r.Accept("nope", badge, 8); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown accept: %v", err)
	}
	if len(r.Waiting()) != 0 {
		t.Fatal("an accepted call leaves the queue")
	}
	if accepted.RoleFor(c.PatientToken) != Patient || accepted.RoleFor(accepted.DoctorToken) != Doctor ||
		accepted.RoleFor("x") != "" || accepted.RoleFor("") != "" {
		t.Fatal("RoleFor is wrong")
	}

	if _, _, err := r.End(c.ID, "wrong"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("end with a wrong token: %v", err)
	}
	ended, completed, err := r.End(c.ID, accepted.DoctorToken)
	if err != nil || !completed || ended.Status != Ended || ended.ProfessionalUserID != 7 {
		t.Fatalf("end: %+v %v %v", ended, completed, err)
	}
	if _, completed, _ := r.End(c.ID, c.PatientToken); completed {
		t.Fatal("only the first end of an accepted call completes it")
	}
}

func TestGivingUpWhileWaitingIsNotACompletedCall(t *testing.T) {
	r := NewRegistry(time.Hour)
	c := r.Create("Sita", nil)
	if _, completed, err := r.End(c.ID, c.PatientToken); err != nil || completed {
		t.Fatalf("completed %v, err %v", completed, err)
	}
}

func TestOldEntriesExpire(t *testing.T) {
	r := NewRegistry(time.Hour)
	now := time.Now()
	r.now = func() time.Time { return now }
	old := r.Create("Old", nil)
	now = now.Add(61 * time.Minute)
	fresh := r.Create("Fresh", nil)
	if _, err := r.Get(old.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal("an hour-old request should be gone")
	}
	if waiting := r.Waiting(); len(waiting) != 1 || waiting[0].ID != fresh.ID {
		t.Fatalf("queue %+v", waiting)
	}
}

func TestOnlyOneOfManySimultaneousAcceptsWins(t *testing.T) {
	r := NewRegistry(time.Hour)
	c := r.Create("Sita", nil)
	var wins atomic.Int32
	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := r.Accept(c.ID, Badge{Name: "Dr", Role: "doctor"}, int64(i)); err == nil {
				wins.Add(1)
			}
		}()
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatalf("%d accepts won", wins.Load())
	}
}

func TestPublicNeverHasTokens(t *testing.T) {
	r := NewRegistry(time.Hour)
	c := r.Create("Sita", nil)
	p := c.Public()
	if p.ID != c.ID || p.PatientName != "Sita" || p.Professional != nil || p.CreatedAt <= 0 {
		t.Fatalf("public %+v", p)
	}
}
