package limiter

import (
	"context"
	"sync"
	"time"
)

type requestTimestamps struct {
	mu         sync.Mutex
	timestamps []time.Time
}

type InMemorySlidingWindowLimiter struct {
	Limit  int
	Window time.Duration

	users map[string]*requestTimestamps
	mu    sync.RWMutex

	cleanupInterval time.Duration
	stopCleanup     chan struct{}
}

func NewInMemorySlidingWindowLimiter(limit int, window time.Duration) *InMemorySlidingWindowLimiter {
	limiter := &InMemorySlidingWindowLimiter{
		Limit:           limit,
		Window:          window,
		users:           make(map[string]*requestTimestamps),
		cleanupInterval: window * 2,
		stopCleanup:     make(chan struct{}),
	}
	return limiter
}

func (l *InMemorySlidingWindowLimiter) Allow(ctx context.Context, userID string) (bool, error) {
	l.mu.RLock() 
	userTimestamps, exists := l.users[userID]
	l.mu.RUnlock()

	if !exists {
		l.mu.Lock() 
		if _, exists = l.users[userID]; !exists {
			userTimestamps = &requestTimestamps{
				timestamps: make([]time.Time, 0, l.Limit+10), 
			}
			l.users[userID] = userTimestamps
		}
		l.mu.Unlock()
	}

	userTimestamps.mu.Lock()
	defer userTimestamps.mu.Unlock()

	now := time.Now()
	newTimestamps := make([]time.Time, 0, len(userTimestamps.timestamps))
	windowStart := now.Add(-l.Window)

	for _, ts := range userTimestamps.timestamps {
		if ts.After(windowStart) {
			newTimestamps = append(newTimestamps, ts)
		}
	}
	userTimestamps.timestamps = newTimestamps

	if len(userTimestamps.timestamps) < l.Limit {
		userTimestamps.timestamps = append(userTimestamps.timestamps, now)
		return true, nil
	}

	return false, nil 
}

func (l *InMemorySlidingWindowLimiter) GetWindowDuration() time.Duration {
	return l.Window
}

func (l *InMemorySlidingWindowLimiter) StartCleanup() {
	ticker := time.NewTicker(l.cleanupInterval)
	defer ticker.Stop() 

	for {
		select {
		case <-ticker.C: 
			l.cleanUpOldEntries()
		case <-l.stopCleanup:
			return
		}
	}
}

func (l *InMemorySlidingWindowLimiter) cleanUpOldEntries() {
	l.mu.Lock() 
	defer l.mu.Unlock()

	now := time.Now()
	threshold := now.Add(-l.Window * 2)

	for userID, tsData := range l.users {
		tsData.mu.Lock() 
		if len(tsData.timestamps) == 0 || tsData.timestamps[len(tsData.timestamps)-1].Before(threshold) {
			delete(l.users, userID)
		}
		tsData.mu.Unlock()
	}
}

func (l *InMemorySlidingWindowLimiter) StopCleanup() {
	close(l.stopCleanup)
}

