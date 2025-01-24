package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

func AccessLog(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := uuid.New().String()
		ctx := context.WithValue(r.Context(), "requestID", requestID)
		r = r.WithContext(ctx)

		logger.Info("Get new request",
			"request_id", requestID,
			"method", r.Method,
			"remote_addr", r.RemoteAddr,
			"url", r.URL.Path,
		)

		start := time.Now()

		next.ServeHTTP(w, r)

		logger.Info("New request complete",
			"request_id", requestID,
			"method", r.Method,
			"remote_addr", r.RemoteAddr,
			"url", r.URL.Path,
			"time", time.Since(start),
		)
	})
}
