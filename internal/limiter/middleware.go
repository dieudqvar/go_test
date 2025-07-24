package limiter

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type RateLimiter interface {
	Allow(ctx context.Context, userID string) (bool, error)
	GetWindowDuration() time.Duration
	StartCleanup()
	StopCleanup()
}

func RateLimitMiddleware(limiter RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := r.Header.Get("X-User-ID")
			if userID == "" {
				userID = r.RemoteAddr
			}

			allowed, err := limiter.Allow(r.Context(), userID)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				fmt.Printf("Rate limiter error: %v\n", err)
				return
			}

			if !allowed {
				retryAfterSeconds := int(limiter.GetWindowDuration().Seconds())
				if retryAfterSeconds == 0 {
					retryAfterSeconds = 1
				}
				w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfterSeconds))
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
