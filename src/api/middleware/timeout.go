package middleware

import (
	"net/http"
	"time"
)

// Timeout wraps an http.Handler with a maximum execution time limit using http.TimeoutHandler.
func Timeout(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, timeout, "Request Timeout")
	}
}
