package ratelimit

import (
	"sync"
	"time"

	"rmpc-server/api/_pkg/config"
)

const maxEntries = 10000

type IPLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

func NewIPLimiter(limit int, window time.Duration) *IPLimiter {
	limiter := &IPLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
	go limiter.cleanup()
	return limiter
}

func (l *IPLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-l.window)

	// Filter to only requests within window
	var recent []time.Time
	for _, t := range l.requests[ip] {
		if t.After(windowStart) {
			recent = append(recent, t)
		}
	}

	if len(recent) >= l.limit {
		l.requests[ip] = recent
		return false
	}

	l.requests[ip] = append(recent, now)

	// Evict oldest entries if map is too large
	if len(l.requests) > maxEntries {
		l.evictOldest()
	}

	return true
}

func (l *IPLimiter) evictOldest() {
	var oldestIP string
	var oldestTime time.Time
	first := true

	for ip, times := range l.requests {
		if len(times) == 0 {
			delete(l.requests, ip)
			continue
		}
		lastAccess := times[len(times)-1]
		if first || lastAccess.Before(oldestTime) {
			oldestIP = ip
			oldestTime = lastAccess
			first = false
		}
	}

	if oldestIP != "" {
		delete(l.requests, oldestIP)
	}
}

func (l *IPLimiter) cleanup() {
	ticker := time.NewTicker(l.window)
	for range ticker.C {
		l.mu.Lock()
		now := time.Now()
		windowStart := now.Add(-l.window)
		for ip, times := range l.requests {
			var recent []time.Time
			for _, t := range times {
				if t.After(windowStart) {
					recent = append(recent, t)
				}
			}
			if len(recent) == 0 {
				delete(l.requests, ip)
			} else {
				l.requests[ip] = recent
			}
		}
		l.mu.Unlock()
	}
}

var (
	authLimiter     *IPLimiter
	authLimiterOnce sync.Once
)

func AuthLimiter() *IPLimiter {
	authLimiterOnce.Do(func() {
		authLimiter = NewIPLimiter(config.Env.AuthRateLimit, time.Minute)
	})
	return authLimiter
}
