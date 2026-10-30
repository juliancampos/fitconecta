package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// Logger retorna um middleware chi que registra as requisicoes usando log/slog estruturado.
func Logger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()

			defer func() {
				latency := time.Since(start)
				reqID := middleware.GetReqID(r.Context())

				logger.Info("HTTP Request",
					"request_id", reqID,
					"method", r.Method,
					"path", r.URL.Path,
					"status", ww.Status(),
					"bytes", ww.BytesWritten(),
					"latency_ms", latency.Milliseconds(),
					"remote_ip", r.RemoteAddr,
				)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}
