package middleware

import (
	"log/slog"
	"net/http"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func RequestLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()
			writer := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				status := writer.Status()
				if status == 0 {
					status = http.StatusOK
				}

				log.Info("HTTP request",
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Int("status", status),
					slog.Duration("duration", time.Since(startedAt)),
					slog.String("request_id", chimiddleware.GetReqID(r.Context())),
				)
			}()

			next.ServeHTTP(writer, r)
		})
	}
}
