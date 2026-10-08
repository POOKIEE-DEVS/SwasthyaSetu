package store_test

import (
	"context"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/store"
	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/testdb"
)

func newStore(t *testing.T) *store.Store {
	return store.New(testdb.Open(t))
}

func ptr(s string) *string { return &s }

func TestGoogleSignInFindsTheDevelopmentAccountByEmail(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	dev, err := s.UpsertDevUser(ctx, " Sita@Example.com ", " Sita ")
	if err != nil {
		t.Fatal(err)
	}
	if dev.Email != "sita@example.com" || dev.Name != "Sita" || dev.GoogleSub != nil {
		t.Fatalf("dev user: %+v", dev)
	}
	again, _ := s.UpsertDevUser(ctx, "sita@example.com", "Other name")
	if again.ID != dev.ID || again.Name != "Sita" {
		t.Fatalf("the development login should find the same account: %+v", again)
	}

	google, err := s.UpsertGoogleUser(ctx, store.GoogleProfile{
		Sub: "sub-1", Email: "sita@example.com", Name: "Sita Sharma", Picture: ptr("https://p"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if google.ID != dev.ID || google.Name != "Sita Sharma" || *google.GoogleSub != "sub-1" {
		t.Fatalf("google user: %+v", google)
	}
	// Later sign-ins match on the Google id, even after an email change.
	moved, _ := s.UpsertGoogleUser(ctx, store.GoogleProfile{Sub: "sub-1", Email: "new@example.com", Name: "Sita"})
	if moved.ID != dev.ID {
		t.Fatalf("matched a different account: %+v", moved)
	}
	stored, _ := s.UserByID(ctx, dev.ID)
	if stored.Email != "new@example.com" || stored.PictureURL != nil {
		t.Fatalf("not updated: %+v", stored)
	}
}

func TestSessionsExpire(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	u, _ := s.UpsertDevUser(ctx, "ram@example.com", "Ram")
	if err := s.CreateSession(ctx, "live", u.ID, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateSession(ctx, "old", u.ID, time.Now().Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if got, err := s.UserForSession(ctx, "live"); err != nil || got == nil || got.ID != u.ID {
		t.Fatalf("live session: %+v, %v", got, err)
	}
	if got, _ := s.UserForSession(ctx, "old"); got != nil {
		t.Fatal("an expired session must not sign anyone in")
	}
	if got, _ := s.UserForSession(ctx, "unknown"); got != nil {
		t.Fatal("an unknown session must not sign anyone in")
	}
	_ = s.DeleteSession(ctx, "live")
	if got, _ := s.UserForSession(ctx, "live"); got != nil {
		t.Fatal("a deleted session must not sign anyone in")
	}
}

func TestApplicationLifecycle(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	u, _ := s.UpsertDevUser(ctx, "anita@example.com", "Anita")
	fields := store.ApplicationFields{
		Role: "doctor", FullName: "Dr. Anita Karki", Phone: "9800000000",
		CitizenshipNumber: "27-01-71-12345", CitizenshipDistrict: "Kathmandu", CouncilNumber: ptr("12345"),
	}
	docs := []store.NewDocument{
		{Kind: "citizenship_front", ContentType: "image/png", Data: []byte("front")},
		{Kind: "citizenship_back", ContentType: "image/jpeg", Data: []byte("back")},
	}
	a, err := s.SubmitApplication(ctx, u, fields, docs)
	if err != nil {
		t.Fatal(err)
	}
	if a.Status != store.StatusPending || a.ReviewedAt != nil || *u.Role != "doctor" {
		t.Fatalf("submitted: %+v", a)
	}
	if store.IsVerified(u, a) {
		t.Fatal("pending is not verified")
	}

	rejected, err := s.DecideApplication(ctx, a.ID, "admin@example.com", false, ptr("Blurry photo."))
	if err != nil || rejected.Status != store.StatusRejected || *rejected.RejectionReason != "Blurry photo." {
		t.Fatalf("rejected: %+v, %v", rejected, err)
	}
	if rejected.ReviewedAt == nil || *rejected.ReviewedBy != "admin@example.com" {
		t.Fatalf("review not recorded: %+v", rejected)
	}

	fields.CitizenshipDistrict = "Lalitpur"
	again, err := s.SubmitApplication(ctx, u, fields, docs[:1])
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != a.ID || again.Status != store.StatusPending || again.RejectionReason != nil ||
		again.ReviewedAt != nil || again.CitizenshipDistrict != "Lalitpur" {
		t.Fatalf("resubmitted: %+v", again)
	}
	if list, _ := s.Documents(ctx, a.ID); len(list) != 1 || list[0].Size != 5 {
		t.Fatalf("old documents should be replaced: %+v", list)
	}

	approved, _ := s.DecideApplication(ctx, a.ID, "admin@example.com", true, nil)
	user, _ := s.UserByID(ctx, u.ID)
	if !store.IsVerified(user, approved) || approved.RejectionReason != nil {
		t.Fatalf("approved: %+v", approved)
	}
	history, _ := s.History(ctx, a.ID)
	var actions []string
	for _, e := range history {
		actions = append(actions, e.Action)
	}
	if strings.Join(actions, ",") != "submitted,rejected,resubmitted,approved" {
		t.Fatalf("history: %v", actions)
	}
	doc, _ := s.Documents(ctx, a.ID)
	full, err := s.Document(ctx, doc[0].ID)
	if err != nil || string(full.Data) != "front" || full.ContentType != "image/png" {
		t.Fatalf("document: %+v, %v", full, err)
	}
	if missing, err := s.Document(ctx, 9999); missing != nil || err != nil {
		t.Fatalf("missing document: %+v, %v", missing, err)
	}
}

func TestListApplicationsOrder(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	var ids []int64
	for _, email := range []string{"a@example.com", "b@example.com", "c@example.com"} {
		u, _ := s.UpsertDevUser(ctx, email, "X")
		a, err := s.SubmitApplication(ctx, u, store.ApplicationFields{
			Role: "nurse", FullName: "X Y", Phone: "9800000000",
			CitizenshipNumber: "1", CitizenshipDistrict: "Kaski",
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, a.ID)
	}
	pending, _ := s.ListApplications(ctx, store.StatusPending)
	if len(pending) != 3 || pending[0].ID != ids[0] || pending[2].ID != ids[2] {
		t.Fatalf("pending should be oldest first: %v", pending)
	}
	all, _ := s.ListApplications(ctx, "")
	if len(all) != 3 || all[0].ID != ids[2] {
		t.Fatalf("all should be newest first: %v", all)
	}
	if none, _ := s.ListApplications(ctx, store.StatusApproved); len(none) != 0 {
		t.Fatalf("approved: %v", none)
	}
}

func TestChatTitle(t *testing.T) {
	cases := map[string][]store.Message{
		"New chat":       {{Role: "assistant", Content: "hi"}},
		"Burned my hand": {{Role: "user", Content: "  Burned \n my\thand "}},
		"मेरो बुबाको छाती दुख्यो": {{Role: "user", Content: "मेरो बुबाको छाती दुख्यो"}},
	}
	for want, messages := range cases {
		if got := store.ChatTitle(messages); got != want {
			t.Errorf("ChatTitle = %q, want %q", got, want)
		}
	}
	long := store.ChatTitle([]store.Message{{Role: "user", Content: strings.Repeat("शब्द ", 50)}})
	if utf8.RuneCountInString(long) != 60 || !strings.HasSuffix(long, "…") {
		t.Errorf("long title %q (%d characters)", long, utf8.RuneCountInString(long))
	}
}

func TestSaveExchangeAddsOnlyWhatIsNew(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	u, _ := s.UpsertDevUser(ctx, "sita@example.com", "Sita")
	q1 := store.Message{Role: "user", Content: "Fever"}
	id, err := s.SaveExchange(ctx, u.ID, "", []store.Message{q1}, "1. Rest.")
	if err != nil || id == "" {
		t.Fatalf("save: %q, %v", id, err)
	}
	history := []store.Message{q1, {Role: "assistant", Content: "1. Rest."}, {Role: "user", Content: "Still hot"}}
	if again, _ := s.SaveExchange(ctx, u.ID, id, history, "2. Fluids."); again != id {
		t.Fatalf("continued chat id %q, want %q", again, id)
	}
	messages, _ := s.ChatMessages(ctx, id)
	if len(messages) != 4 || messages[2].Content != "Still hot" || messages[3].Content != "2. Fluids." {
		t.Fatalf("messages: %+v", messages)
	}

	// Someone else's chat id starts a fresh chat of your own.
	other, _ := s.UpsertDevUser(ctx, "ram@example.com", "Ram")
	mine, _ := s.SaveExchange(ctx, other.ID, id, []store.Message{{Role: "user", Content: "Mine"}}, "ok")
	if mine == id {
		t.Fatal("a chat id that isn't yours must not be continued")
	}
	if c, _ := s.OwnedChat(ctx, other.ID, id); c != nil {
		t.Fatal("chats are private")
	}
	_ = s.DeleteChat(ctx, other.ID, id) // not theirs: nothing happens
	if messages, _ := s.ChatMessages(ctx, id); len(messages) != 4 {
		t.Fatal("deleting someone else's chat must do nothing")
	}
}

func TestListChatsNewestFirst(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	u, _ := s.UpsertDevUser(ctx, "sita@example.com", "Sita")
	for _, text := range []string{"Headache", "Cough", "Burn"} {
		if _, err := s.CreateChat(ctx, u.ID, []store.Message{{Role: "user", Content: text}}); err != nil {
			t.Fatal(err)
		}
	}
	chats, err := s.ListChats(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	var titles []string
	for _, c := range chats {
		titles = append(titles, c.Title)
		if c.MessageCount != 1 {
			t.Errorf("%s: %d messages", c.Title, c.MessageCount)
		}
	}
	if strings.Join(titles, ",") != "Burn,Cough,Headache" {
		t.Fatalf("order: %v", titles)
	}
	if err := s.DeleteChat(ctx, u.ID, chats[0].ID); err != nil {
		t.Fatal(err)
	}
	if left, _ := s.ListChats(ctx, u.ID); len(left) != 2 {
		t.Fatalf("after delete: %d chats", len(left))
	}
}

func TestHelpRecordsCountOncePerCall(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	pro, _ := s.UpsertDevUser(ctx, "doctor@example.com", "Doctor")
	start := time.Now().Add(-time.Hour)
	for i, name := range []string{"Sita", "Ram", "Gita"} {
		r := store.HelpRecord{
			ProfessionalID: pro.ID, ConsultationID: name, PatientName: name,
			StartedAt: start.Add(time.Duration(i) * time.Minute), EndedAt: time.Now(), DurationSeconds: 60,
		}
		if ok, err := s.RecordHelp(ctx, r); !ok || err != nil {
			t.Fatalf("record %s: %v, %v", name, ok, err)
		}
		if ok, err := s.RecordHelp(ctx, r); ok || err != nil {
			t.Fatalf("a second end of the same call must not count: %v, %v", ok, err)
		}
	}
	count, people, err := s.HelpSummary(ctx, pro.ID)
	if err != nil || count != 3 || len(people) != 3 || people[0].PatientName != "Gita" {
		t.Fatalf("summary: %d %+v %v", count, people, err)
	}
	other, _ := s.UpsertDevUser(ctx, "nurse@example.com", "Nurse")
	if count, people, _ := s.HelpSummary(ctx, other.ID); count != 0 || len(people) != 0 {
		t.Fatalf("someone else's record leaked: %d %+v", count, people)
	}
}

func TestTimesComeBackAsWritten(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	before := store.Now()
	u, _ := s.UpsertDevUser(ctx, "time@example.com", "T")
	got, _ := s.UserByID(ctx, u.ID)
	if !got.CreatedAt.Equal(u.CreatedAt) || got.CreatedAt.Before(before) {
		t.Fatalf("created_at %v, wrote %v", got.CreatedAt, u.CreatedAt)
	}
	if store.Unix(time.Unix(1759912345, 123456000)) != 1759912345.123456 {
		t.Fatal("Unix should keep microseconds")
	}
}
