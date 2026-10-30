package router

import (
	"sync"
	"time"
)

const (
	postRequestWindow = time.Minute
)

type postRateLimiter struct {
	mu       sync.Mutex
	limit    int
	requests []time.Time
}

func newPostRateLimiter(limit int) *postRateLimiter {
	return &postRateLimiter{limit: limit}
}

// allow enforces a rolling one-minute limit shared by all POST requests.
func (l *postRateLimiter) allow() (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-postRequestWindow)
	l.requests = retainRecentRequests(l.requests, cutoff)
	if len(l.requests) >= l.limit {
		return false, l.requests[0].Add(postRequestWindow).Sub(now)
	}

	l.requests = append(l.requests, now)
	return true, 0
}

func retainRecentRequests(requests []time.Time, cutoff time.Time) []time.Time {
	firstRecent := 0
	for firstRecent < len(requests) && !requests[firstRecent].After(cutoff) {
		firstRecent++
	}
	return requests[firstRecent:]
}
