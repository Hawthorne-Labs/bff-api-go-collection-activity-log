package middleware

import (
	"sync"
	"time"
)

type RateLimitStore interface {
	Allow(key string) bool
}

type memoryRateLimitStore struct {
	limit   int
	window  time.Duration
	mu      sync.Mutex
	entries map[string][]time.Time
}

func NewMemoryRateLimitStore(limit, windowSeconds int) RateLimitStore {
	if limit <= 0 {
		limit = 60
	}
	if windowSeconds <= 0 {
		windowSeconds = 60
	}
	return &memoryRateLimitStore{
		limit:   limit,
		window:  time.Duration(windowSeconds) * time.Second,
		entries: make(map[string][]time.Time),
	}
}

func (s *memoryRateLimitStore) Allow(key string) bool {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	active := s.entries[key][:0]
	for _, ts := range s.entries[key] {
		if now.Sub(ts) < s.window {
			active = append(active, ts)
		}
	}
	if len(active) >= s.limit {
		s.entries[key] = active
		return false
	}
	active = append(active, now)
	s.entries[key] = active
	return true
}
