package middleware

import (
	"log"
	"net/http"
	"time"
)

// Logging wraps an http.Handler and logs the request method, path, and duration.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		log.Printf("START %s %s", r.Method, r.URL.Path)

		next.ServeHTTP(w, r)

		duration := time.Since(start)
		log.Printf("END %s %s - %v", r.Method, r.URL.Path, duration)
	})
}
