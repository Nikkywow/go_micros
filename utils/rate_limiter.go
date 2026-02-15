package utils

import (
	"net/http"

	"golang.org/x/time/rate"
)

func NewRateLimitMiddleware(r rate.Limit, burst int) func(http.Handler) http.Handler {
	limiter := rate.NewLimiter(r, burst)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if !limiter.Allow() {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, req)
		})
	}
}
