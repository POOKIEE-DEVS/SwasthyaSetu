package ratelimit

import (
	"testing"
	"time"
)

func TestSlidingWindow(t *testing.T) {
	now := time.Now()
	l := New(2, time.Minute)
	l.now = func() time.Time { return now }

	for i, want := range []bool{true, true, false} {
		if got := l.Allow("a"); got != want {
			t.Fatalf("hit %d: allowed %v, want %v", i+1, got, want)
		}
	}
	if !l.Allow("b") {
		t.Fatal("keys are independent")
	}
	now = now.Add(30 * time.Second)
	if l.Allow("a") {
		t.Fatal("still inside the window")
	}
	now = now.Add(31 * time.Second)
	if !l.Allow("a") {
		t.Fatal("the first hits have left the window")
	}
}

func TestIdleKeysAreSwept(t *testing.T) {
	now := time.Now()
	l := New(5, time.Minute)
	l.now = func() time.Time { return now }
	for _, ip := range []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"} {
		l.Allow(ip)
	}
	now = now.Add(2 * time.Minute)
	l.Allow("4.4.4.4")
	if len(l.hits) != 1 {
		t.Fatalf("%d keys kept", len(l.hits))
	}
	l.Clear()
	if len(l.hits) != 0 {
		t.Fatal("Clear should forget everything")
	}
}
