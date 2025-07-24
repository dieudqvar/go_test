package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go_test/internal/api"
	"go_test/internal/limiter"
)

func main() {

	var rateLimiter limiter.RateLimiter

	// rateLimiter = limiter.NewInMemoryFixedWindowLimiter(100, 1*time.Second)

	//rateLimiter = limiter.NewInMemorySlidingWindowLimiter(100, 1*time.Second)

	//    capacity: 100
	//    rate: 100 req/s
	rateLimiter = limiter.NewInMemoryTokenBucketLimiter(100, 100)

	if rateLimiter == nil {
		log.Fatal("Please choice")
	}

	defer rateLimiter.StopCleanup()

	router := api.NewRouter(rateLimiter)

	port := ":8081"
	server := &http.Server{
		Addr:    port,
		Handler: router,

		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("Server running on http://localhost%s\n", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Can not start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Closing server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server can not shutdown properly: %v", err)
	}
	log.Println("Server have been shutdown")
}
