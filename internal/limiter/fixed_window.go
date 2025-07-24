package limiter

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type rateLimitState struct {
	count     int
	windowEnd int64
}

type FixedWindowLimiter struct {
	Limit  int
	Window time.Duration

	counts map[string]int
	mu     sync.RWMutex

	cleanupInterval time.Duration
	stopCleanup     chan struct{}
}

func NewInMemoryFixedWindowLimiter(limit int, window time.Duration) *FixedWindowLimiter {
	limiter := &FixedWindowLimiter{
		Limit:           limit,
		Window:          window,
		counts:          make(map[string]int),
		cleanupInterval: window * 2,
		stopCleanup:     make(chan struct{}),
	}
	return limiter
}

func (l *FixedWindowLimiter) Allow(ctx context.Context, userID string) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	windowStart := now.Truncate(l.Window).Unix()
	key := fmt.Sprintf("%s_%d", userID, windowStart)

	currentCount := l.counts[key]

	if currentCount < l.Limit {
		l.counts[key] = currentCount + 1
		return true, nil
	}

	return false, nil
}

func (l *FixedWindowLimiter) GetWindowDuration() time.Duration {
	return l.Window
}

func (l *FixedWindowLimiter) cleanUpOldEntries() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	threshold := now.Add(-l.Window * 2).Unix()

	for key := range l.counts {
		var userID string
		var windowStart int64
		_, err := fmt.Sscanf(key, "%s_%d", &userID, &windowStart)
		if err != nil {
			delete(l.counts, key)
			continue
		}

		if windowStart < threshold {
			delete(l.counts, key)
		}
	}
}

func (l *FixedWindowLimiter) StopCleanup() {
	close(l.stopCleanup)
}
