// Package ratelimit is a per-client sliding-window limit, in memory.
//
// The demo is public and every chat message spends GPU time on the model
// server, so a stray script or a refresh loop must not drain it.
package ratelimit

import (
	"sync"
	"time"
)

// Limiter allows at most a number of hits per key within a window.
type Limiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	hits   map[string][]time.Time
	now    func() time.Time
	// Keys idle for a whole window are swept now and then, so memory stays
	// bounded however many clients come and go.
	nextSweep time.Time
}

// New allows limit hits per key per window.
func New(limit int, window time.Duration) *Limiter {
	return &Limiter{limit: limit, window: window, hits: map[string][]time.Time{}, now: time.Now}
}

// Allow records a hit for key and reports whether it is within the limit.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	cutoff := now.Add(-l.window)
	if now.After(l.nextSweep) {
		for k, times := range l.hits {
			if len(times) == 0 || !times[len(times)-1].After(cutoff) {
				delete(l.hits, k)
			}
		}
		l.nextSweep = now.Add(l.window)
	}
	times := l.hits[key]
	i := 0
	for i < len(times) && !times[i].After(cutoff) {
		i++
	}
	times = times[i:]
	if len(times) >= l.limit {
		l.hits[key] = times
		return false
	}
	l.hits[key] = append(times, now)
	return true
}

// Clear forgets every key.
func (l *Limiter) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.hits = map[string][]time.Time{}
}
