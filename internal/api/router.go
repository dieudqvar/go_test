// internal/api/router.go
package api

import (
	"fmt"
	"net/http"

	"go_test/internal/limiter" 

	"github.com/gorilla/mux"
)

func NewRouter(limiterInstance limiter.RateLimiter) *mux.Router {
	r := mux.NewRouter()

	// normal router
	r.HandleFunc("/unlimited", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Bạn có thể truy cập không giới hạn!")
	}).Methods("GET")

	//router using Fixed Window Limit
	r.Handle("/init", limiter.RateLimitMiddleware(limiterInstance)(http.HandlerFunc(HelloHandler))).Methods("GET")

	return r
}
