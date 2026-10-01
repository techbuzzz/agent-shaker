package middleware

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// ctxKey is a private type used for context values stored by this package so
// they cannot collide with keys defined by other packages.
type ctxKey int

const (
	requestIDKey ctxKey = iota
)

// RequestIDHeader is the header used both for inbound (honored when set) and
// outbound (always set) request identifiers.
const RequestIDHeader = "X-Request-ID"

// NewRequestID returns a 16-byte random hex string suitable for use as a
// per-request correlation id.
func NewRequestID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		// crypto/rand failure is extraordinarily rare; fall back to a
		// timestamp-derived value so we always have a non-empty id.
		return "req-" + time.Now().UTC().Format("20060102T150405.000000000")
	}
	return hex.EncodeToString(buf[:])
}

// RequestIDWithContext returns a derived context carrying the given request id.
func RequestIDWithContext(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestIDFromContext extracts the request id placed by RequestID. Returns ""
// when no id is present.
func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

// RequestID middleware ensures every request has an X-Request-ID. It honors
// an inbound header (useful for tracing across services) and otherwise
// generates a new id. The id is attached to the request context and echoed
// back in the response header.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(RequestIDHeader)
		if id == "" {
			id = NewRequestID()
		}
		w.Header().Set(RequestIDHeader, id)
		ctx := RequestIDWithContext(r.Context(), id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// responseWriter wraps http.ResponseWriter so middlewares can read the status
// code and the number of bytes written.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	bytes      int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.bytes += n
	return n, err
}

// Hijack delegates to the wrapped writer so WebSocket upgrades still work when
// this wrapper is in the chain. Without it the Upgrader fails with
// "response does not implement http.Hijacker".
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errNotHijacker
	}
	// A hijacked connection never emits an HTTP status; record the 101 so the
	// access log does not report every WebSocket as a 2xx.
	rw.statusCode = http.StatusSwitchingProtocols
	return hj.Hijack()
}

// Flush delegates to the wrapped writer when it supports flushing.
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap exposes the underlying writer to http.ResponseController.
func (rw *responseWriter) Unwrap() http.ResponseWriter { return rw.ResponseWriter }

var errNotHijacker = errors.New("middleware: underlying http.ResponseWriter does not implement http.Hijacker")

// Logger middleware logs each HTTP request using slog. The log line carries
// method, path, status, duration, remote addr, and the request id when present.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		slog.InfoContext(r.Context(), "http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.statusCode,
			"bytes", wrapped.bytes,
			"duration_ms", time.Since(start).Milliseconds(),
			"remote", r.RemoteAddr,
			"request_id", RequestIDFromContext(r.Context()),
		)
	})
}

// RequestSizeLimit middleware limits request body size.
func RequestSizeLimit(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

// Recovery middleware recovers from panics, logs the panic with its request id
// and stack, and returns a generic 500. Library code must not panic.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic recovered",
					"error", err,
					"path", r.URL.Path,
					"method", r.Method,
					"request_id", RequestIDFromContext(r.Context()),
				)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
