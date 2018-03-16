package api

import (
	"encoding/base64"
	"log"
	"net/http"
	"strings"
	"time"
)

const timeLayout = "02/Jan/2006:15:04:05 -0700"

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

// A middleware that provides logging for each HTTP request.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := loggingResponseWriter{
			ResponseWriter: w,
			status:         200,
			contentLen:     0,
		}
		next.ServeHTTP(&logger, r)
		log.Printf("%s - - [%s] \"%s %s %s\" %d %d",
			r.RemoteAddr,
			time.Now().Format(timeLayout),
			r.Method,
			r.URL,
			r.Proto,
			logger.status,
			logger.contentLen)
	})
}

func authenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		privateToken := r.Header.Get(PrivateTokenHeader)

		if privateToken == "" {
			http.Error(w, "Bad private token", http.StatusBadRequest)
			return
		}

		payload, _ := base64.StdEncoding.DecodeString(privateToken)
		pair := strings.SplitN(string(payload), ":", 2)

		if len(pair) != 2 || !validate(pair[0], pair[1]) {
			http.Error(w, "Authorization failed", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
