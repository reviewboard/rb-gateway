package api

import (
	"log/slog"
	"net/http"
	"time"
)

// A specialized `http.ResponseWriter` for logging.
type loggingResponseWriter struct {
	http.ResponseWriter
	status     int
	contentLen int
}

// Write the header for the given status code.
func (l *loggingResponseWriter) WriteHeader(status int) {
	l.status = status
	l.ResponseWriter.WriteHeader(status)
}

// Write the given content to the client.
func (l *loggingResponseWriter) Write(content []byte) (int, error) {
	l.contentLen += len(content)
	return l.ResponseWriter.Write(content)
}

// A middleware that provides structured logging for each HTTP request.
func loggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lw := loggingResponseWriter{
			ResponseWriter: w,
			status:         200,
			contentLen:     0,
		}
		next.ServeHTTP(&lw, r)
		logger.Info("request",
			"remote", r.RemoteAddr,
			"method", r.Method,
			"path", r.URL.String(),
			"proto", r.Proto,
			"status", lw.status,
			"bytes", lw.contentLen,
			"duration", time.Since(start),
		)
	})
}
