package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExtractCredential(t *testing.T) {
	tests := []struct {
		name          string
		apiKeyHeader  string
		authzHeader   string
		query         string
		allowQueryKey bool
		want          string
	}{
		{
			name:         "x-api-key header",
			apiKeyHeader: "secret-key",
			want:         "secret-key",
		},
		{
			name:        "authorization bearer",
			authzHeader: "Bearer secret-key",
			want:        "secret-key",
		},
		{
			name:        "authorization scheme is case-insensitive",
			authzHeader: "bearer secret-key",
			want:        "secret-key",
		},
		{
			name:        "bare authorization token without a scheme",
			authzHeader: "secret-key",
			want:        "secret-key",
		},
		{
			name:         "x-api-key wins over authorization",
			apiKeyHeader: "from-x-api-key",
			authzHeader:  "Bearer from-authorization",
			want:         "from-x-api-key",
		},
		{
			name:          "query parameter when allowed",
			query:         "?api_key=secret-key",
			allowQueryKey: true,
			want:          "secret-key",
		},
		{
			// This is the case that matters: a key in a query string would be
			// written to access logs, so it must be ignored when not allowed.
			name:  "query parameter ignored when not allowed",
			query: "?api_key=secret-key",
			want:  "",
		},
		{
			name:          "query parameter is a fallback, not a priority",
			apiKeyHeader:  "from-header",
			query:         "?api_key=from-query",
			allowQueryKey: true,
			want:          "from-header",
		},
		{
			name: "no credential",
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			target := "/api/projects"
			if tc.query != "" {
				target += tc.query
			}
			r := httptest.NewRequest(http.MethodGet, target, nil)
			if tc.apiKeyHeader != "" {
				r.Header.Set(HeaderAPIKey, tc.apiKeyHeader)
			}
			if tc.authzHeader != "" {
				r.Header.Set(HeaderAuthz, tc.authzHeader)
			}
			if got := extractCredential(r, tc.allowQueryKey); got != tc.want {
				t.Errorf("extractCredential() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRequireAPIKeyDisabledIsPassThrough(t *testing.T) {
	mw, err := RequireAPIKey(AuthConfig{Enabled: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	called := false
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/projects", nil))
	if !called {
		t.Error("handler not reached; disabled auth must not intercept")
	}
}

// TestRequireAPIKeyMisconfigured pins the fail-fast contract: switching auth
// on without keys must be an error at construction, never a middleware that
// silently accepts everything.
func TestRequireAPIKeyMisconfigured(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  AuthConfig
	}{
		{name: "enabled with no keys", cfg: AuthConfig{Enabled: true}},
		{name: "enabled with only empty keys", cfg: AuthConfig{Enabled: true, Keys: []string{"", "  "}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := RequireAPIKey(tc.cfg); err == nil {
				t.Fatal("expected ErrAuthMisconfigured, got nil")
			}
		})
	}
}

func TestRequireAPIKeyEnforcement(t *testing.T) {
	const key = "s3cr3t-key-value"

	mw, err := RequireAPIKey(AuthConfig{
		Enabled:       true,
		Keys:          []string{key, "another-key"},
		AllowQueryKey: true,
		Skip:          []string{"/healthz", "/readyz", "/metrics"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tests := []struct {
		name       string
		path       string
		apiKey     string
		authz      string
		wantStatus int
	}{
		{name: "valid x-api-key", path: "/api/projects", apiKey: key, wantStatus: http.StatusOK},
		{name: "valid bearer", path: "/api/projects", authz: "Bearer " + key, wantStatus: http.StatusOK},
		{name: "second configured key also works", path: "/api/projects", apiKey: "another-key", wantStatus: http.StatusOK},
		{name: "valid key via query", path: "/ws?api_key=" + key, wantStatus: http.StatusOK},
		{name: "no credential", path: "/api/projects", wantStatus: http.StatusUnauthorized},
		{name: "wrong key", path: "/api/projects", apiKey: "nope", wantStatus: http.StatusUnauthorized},
		{name: "key is a prefix of a valid one", path: "/api/projects", apiKey: key[:len(key)-1], wantStatus: http.StatusUnauthorized},
		{name: "key is a superstring of a valid one", path: "/api/projects", apiKey: key + "x", wantStatus: http.StatusUnauthorized},
		{name: "empty bearer", path: "/api/projects", authz: "Bearer ", wantStatus: http.StatusUnauthorized},
		{name: "healthz bypasses auth", path: "/healthz", wantStatus: http.StatusOK},
		{name: "readyz bypasses auth", path: "/readyz", wantStatus: http.StatusOK},
		{name: "metrics bypasses auth", path: "/metrics", wantStatus: http.StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			if tc.apiKey != "" {
				req.Header.Set(HeaderAPIKey, tc.apiKey)
			}
			if tc.authz != "" {
				req.Header.Set(HeaderAuthz, tc.authz)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d (body: %s)", rec.Code, tc.wantStatus, rec.Body.String())
			}
		})
	}
}

// TestUnauthorizedResponseShape keeps the 401 in the same envelope every other
// error uses, so clients have one parser.
func TestUnauthorizedResponseShape(t *testing.T) {
	mw, err := RequireAPIKey(AuthConfig{Enabled: true, Keys: []string{"k"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/projects", nil))

	if got := rec.Header().Get("WWW-Authenticate"); !strings.HasPrefix(got, "Bearer") {
		t.Errorf("WWW-Authenticate = %q, want a Bearer challenge", got)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	body := rec.Body.String()
	for _, want := range []string{`"error"`, `"code":"unauthorized"`, `"request_id"`} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %s: %s", want, body)
		}
	}
}

// TestUnauthorizedDoesNotEchoCredential guards against a response that
// reflects the presented key back, which would turn any auth endpoint into an
// oracle for guessing keys one character at a time.
func TestUnauthorizedDoesNotEchoCredential(t *testing.T) {
	mw, err := RequireAPIKey(AuthConfig{Enabled: true, Keys: []string{"real-key"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	for _, guess := range []string{"real", "real-ke", "real-key-but-longer", "wrong"} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
		req.Header.Set(HeaderAPIKey, guess)
		h.ServeHTTP(rec, req)
		if strings.Contains(rec.Body.String(), guess) {
			t.Errorf("response echoed the presented credential %q: %s", guess, rec.Body.String())
		}
	}
}

// TestMatchesAnyKeyAcceptsAnyPosition confirms the constant-time comparison
// loop really is order-independent: a key late in the list still matches.
func TestMatchesAnyKeyAcceptsAnyPosition(t *testing.T) {
	keys := []string{"first", "second", "third", "fourth"}
	for _, k := range keys {
		if !matchesAnyKey(k, keys) {
			t.Errorf("key %q should match", k)
		}
	}
	if matchesAnyKey("fifth", keys) {
		t.Error("unknown key matched")
	}
	if matchesAnyKey("", keys) {
		t.Error("empty key matched")
	}
}

func TestAPIKeysFromEnv(t *testing.T) {
	t.Setenv("TEST_AGENT_SHAKER_KEYS", " a , b ,, c ")
	got := APIKeysFromEnv("TEST_AGENT_SHAKER_KEYS")
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("key[%d] = %q, want %q", i, got[i], want[i])
		}
	}

	t.Setenv("TEST_AGENT_SHAKER_KEYS", "   ")
	if got := APIKeysFromEnv("TEST_AGENT_SHAKER_KEYS"); got != nil {
		t.Errorf("blank env should yield nil, got %v", got)
	}
}

// TestDescribeAuthConfigRedacts makes sure the startup log line can never
// become the place the credential is written down.
func TestDescribeAuthConfigRedacts(t *testing.T) {
	full := "super-secret-key-material"
	desc := DescribeAuthConfig(true, []string{full})
	if strings.Contains(desc, full) {
		t.Errorf("description leaked the full key: %s", desc)
	}
	if !strings.Contains(desc, "1 key") {
		t.Errorf("description should report the key count: %s", desc)
	}
	if d := DescribeAuthConfig(false, nil); d != "disabled" {
		t.Errorf("disabled description = %q", d)
	}
	if d := DescribeAuthConfig(true, nil); !strings.Contains(d, "misconfigured") {
		t.Errorf("misconfigured description = %q", d)
	}
}

// TestRandomKeyHelper documents that keys should be high-entropy; the length
// check mirrors what a deployment script would assert.
func TestRandomKeyHelper(t *testing.T) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("rand: %v", err)
	}
	if got := len(hex.EncodeToString(b)); got != 64 {
		t.Errorf("generated key length = %d, want 64", got)
	}
}
