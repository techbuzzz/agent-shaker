package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// captureLog runs fn with slog writing into a buffer and returns what was
// written.
func captureLog(t *testing.T, fn func()) string {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	fn()
	return buf.String()
}

// TestLoggerClientIPReportsTheClient is the point of LoggerClientIP. Behind a
// proxy, logging RemoteAddr writes the same line for every caller, which is the
// one thing an access log cannot afford to do.
func TestLoggerClientIPReportsTheClient(t *testing.T) {
	// No t.Parallel: captureLog swaps the process-wide slog default, and two
	// parallel tests doing that clobber each other's output.
	log := captureLog(t, func() {
		req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
		req.RemoteAddr = "172.20.0.5:46766"
		req.Header.Set("X-Forwarded-For", "203.0.113.77, 198.51.100.12")
		rec := httptest.NewRecorder()
		LoggerClientIP(true)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {})).
			ServeHTTP(rec, req)
	})

	// The header is "<forged>, <real>" — what a proxy appends. The logged
	// address must be the RIGHTMOST one, i.e. the peer the proxy actually saw.
	// Expecting the leftmost here would be the leftmost-trust bug this whole
	// change exists to avoid.
	if !strings.Contains(log, `"remote":"198.51.100.12"`) {
		t.Errorf("log does not report the client address; got %s", log)
	}
	if strings.Contains(log, `"remote":"203.0.113.77"`) {
		t.Errorf("log trusted the caller-supplied prefix; got %s", log)
	}
	// The socket peer is kept too, so a misconfiguration is diagnosable: if
	// `remote` ever stops varying while `peer` does, the header is not arriving.
	if !strings.Contains(log, `"peer":"172.20.0.5:46766"`) {
		t.Errorf("log does not retain the socket peer; got %s", log)
	}
}

// TestLoggerIgnoresForwardedForWhenUntrusted: the default must not record a
// caller-supplied value, or an attacker can poison the operator's log.
func TestLoggerIgnoresForwardedForWhenUntrusted(t *testing.T) {
	log := captureLog(t, func() {
		req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
		req.RemoteAddr = "203.0.113.9:5555"
		req.Header.Set("X-Forwarded-For", "1.2.3.4")
		rec := httptest.NewRecorder()
		LoggerClientIP(false)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {})).
			ServeHTTP(rec, req)
	})

	if strings.Contains(log, "1.2.3.4") {
		t.Errorf("untrusted X-Forwarded-For leaked into the log; got %s", log)
	}
	if !strings.Contains(log, `"remote":"203.0.113.9"`) {
		t.Errorf("log should fall back to the socket peer; got %s", log)
	}
}

// TestLoggerAgreesWithRateLimiter is the property that makes the log useful at
// all: correlating a 429 with the request behind it joins on the client
// address, so the two must resolve the same value for the same request.
func TestLoggerAgreesWithRateLimiter(t *testing.T) {
	const xff = "203.0.113.50, 198.51.100.3"
	req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	req.RemoteAddr = "172.20.0.5:46766"
	req.Header.Set("X-Forwarded-For", xff)

	fromLimiter := clientIP(req, true)

	log := captureLog(t, func() {
		rec := httptest.NewRecorder()
		LoggerClientIP(true)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {})).
			ServeHTTP(rec, req)
	})

	if !strings.Contains(log, `"remote":"`+fromLimiter+`"`) {
		t.Errorf("logger reported something other than the rate limiter's key %q; log: %s", fromLimiter, log)
	}
}

// TestLoggerPreservesStatusAndRequestID keeps the other fields intact: the log
// is only worth reading if it still says what happened.
func TestLoggerPreservesStatusAndRequestID(t *testing.T) {
	log := captureLog(t, func() {
		req := httptest.NewRequest(http.MethodPost, "/api/tasks", nil)
		req.RemoteAddr = "203.0.113.9:5555"
		req = req.WithContext(RequestIDWithContext(req.Context(), "req-abc123"))
		rec := httptest.NewRecorder()
		LoggerClientIP(false)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
		})).ServeHTTP(rec, req)
	})

	for _, want := range []string{
		`"method":"POST"`,
		`"path":"/api/tasks"`,
		`"status":201`,
		`"request_id":"req-abc123"`,
	} {
		if !strings.Contains(log, want) {
			t.Errorf("log missing %s; got %s", want, log)
		}
	}
}
