package middleware

import (
	"log/slog"
	"net/http"
)

func Panic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Info("recovered", "request_id", r.Context().Value("requestID"), "error", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
