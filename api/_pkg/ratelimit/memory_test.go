package ratelimit

import (
	"sync"
	"testing"
	"time"
)

func TestAllow(t *testing.T) {
	limiter := NewIPLimiter(3, time.Second)

	for i := 0; i < 3; i++ {
		if !limiter.Allow("1.2.3.4") {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}

	if limiter.Allow("1.2.3.4") {
		t.Fatal("4th request should be denied")
	}
}

func TestAllowDifferentIPs(t *testing.T) {
	limiter := NewIPLimiter(1, time.Second)

	if !limiter.Allow("1.1.1.1") {
		t.Fatal("first IP should be allowed")
	}
	if !limiter.Allow("2.2.2.2") {
		t.Fatal("second IP should be allowed")
	}
	if limiter.Allow("1.1.1.1") {
		t.Fatal("first IP should be denied on second request")
	}
}

func TestWindowExpiry(t *testing.T) {
	limiter := NewIPLimiter(1, 50*time.Millisecond)

	if !limiter.Allow("1.2.3.4") {
		t.Fatal("first request should be allowed")
	}
	if limiter.Allow("1.2.3.4") {
		t.Fatal("second request should be denied")
	}

	time.Sleep(60 * time.Millisecond)

	if !limiter.Allow("1.2.3.4") {
		t.Fatal("request after window should be allowed")
	}
}

func TestConcurrency(t *testing.T) {
	limiter := NewIPLimiter(100, time.Second)
	var wg sync.WaitGroup
	allowed := make(chan bool, 200)

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			allowed <- limiter.Allow("1.2.3.4")
		}()
	}

	wg.Wait()
	close(allowed)

	count := 0
	for a := range allowed {
		if a {
			count++
		}
	}

	if count != 100 {
		t.Fatalf("expected exactly 100 allowed, got %d", count)
	}
}

func TestMapCap(t *testing.T) {
	limiter := NewIPLimiter(1, time.Hour)

	for i := 0; i < maxEntries+100; i++ {
		limiter.Allow("ip-" + string(rune(i)))
	}

	limiter.mu.Lock()
	size := len(limiter.requests)
	limiter.mu.Unlock()

	if size > maxEntries+1 {
		t.Fatalf("map size %d exceeds cap %d by too much", size, maxEntries)
	}
}
