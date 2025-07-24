package limiter

import (
	"context"
	"math"
	"sync"
	"time"
)

type bucketState struct {
	mu             sync.Mutex
	tokens         float64
	lastRefillTime time.Time
}

type InMemoryTokenBucketLimiter struct {
	Capacity   float64
	RefillRate time.Duration

	users           map[string]*bucketState
	mu              sync.RWMutex
	simulatedWindow time.Duration
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
}

func NewInMemoryTokenBucketLimiter(capacity float64, rate int) *InMemoryTokenBucketLimiter {
	if rate <= 0 {
		rate = 1
	}

	refillRate := time.Duration(float64(time.Second) / float64(rate))

	l := &InMemoryTokenBucketLimiter{
		Capacity:        float64(capacity),
		RefillRate:      refillRate,
		users:           make(map[string]*bucketState),
		simulatedWindow: time.Duration(float64(capacity) * float64(refillRate)),
		cleanupInterval: refillRate * time.Duration(capacity) * 2,
		stopCleanup:     make(chan struct{}),
	}
	return l

}

func (l *InMemoryTokenBucketLimiter) Allow(ctx context.Context, userID string) (bool, error) {
	l.mu.RLock() // Khóa đọc để tìm user
	userBucket, exists := l.users[userID]
	l.mu.RUnlock()

	if !exists {
		l.mu.Lock()
		if _, exists = l.users[userID]; !exists {
			userBucket = &bucketState{
				tokens:         l.Capacity,
				lastRefillTime: time.Now(),
			}
			l.users[userID] = userBucket
		}
		l.mu.Unlock()
	}

	userBucket.mu.Lock()
	defer userBucket.mu.Unlock()

	now := time.Now()
	timeElapsed := now.Sub(userBucket.lastRefillTime)
	tokensToAdd := float64(timeElapsed) / float64(l.RefillRate)

	userBucket.tokens = math.Min(l.Capacity, userBucket.tokens+tokensToAdd)
	userBucket.lastRefillTime = now

	if userBucket.tokens >= 1.0 {
		userBucket.tokens -= 1.0
		return true, nil
	}

	return false, nil
}

func (l *InMemoryTokenBucketLimiter) GetWindowDuration() time.Duration {
	return l.simulatedWindow
}

func (l *InMemoryTokenBucketLimiter) StartCleanup() {
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

func (l *InMemoryTokenBucketLimiter) cleanUpOldEntries() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	threshold := now.Add(-l.simulatedWindow * 2)

	for userID, tsData := range l.users {
		tsData.mu.Lock()
		if tsData.lastRefillTime.Before(threshold) && tsData.tokens >= l.Capacity {
			delete(l.users, userID)
		}
		tsData.mu.Unlock()
	}
}

func (l *InMemoryTokenBucketLimiter) StopCleanup() {
	close(l.stopCleanup)
}
