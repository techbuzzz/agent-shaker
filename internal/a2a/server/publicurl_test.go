package server

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolvePublicBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		configured string
		target     string
		tls        bool
		headers    map[string]string
		want       string
	}{
		{
			name: "configured base url wins over everything",
			// An operator who sets BASE_URL is stating the answer, even when the
			// request carries forwarding headers that disagree.
			configured: "https://api.example.com/",
			target:     "http://mcp-server:8080/.well-known/agent-card.json",
			headers:    map[string]string{"X-Forwarded-Host": "edge.example.com", "X-Forwarded-Proto": "https"},
			want:       "https://api.example.com",
		},
		{
			name: "configured value is trimmed of trailing slash",
			// Callers append "/a2a/v1" directly; a double slash would be a
			// different (404) path on some servers.
			configured: "https://api.example.com///",
			target:     "/",
			want:       "https://api.example.com",
		},
		{
			name:    "forwarded proto and host from a tls terminating edge",
			target:  "/.well-known/agent-card.json",
			headers: map[string]string{"X-Forwarded-Proto": "https", "X-Forwarded-Host": "shaker.example.com"},
			want:    "https://shaker.example.com",
		},
		{
			name:   "direct request without any proxy",
			target: "http://mcp-server:8080/",
			want:   "http://mcp-server:8080",
		},
		{
			name:   "direct tls request",
			target: "https://shaker.example.com/",
			tls:    true,
			want:   "https://shaker.example.com",
		},
		{
			name:    "first entry of a forwarded chain is the closest proxy",
			target:  "/",
			headers: map[string]string{"X-Forwarded-Host": "edge.example.com, inner.example.com", "X-Forwarded-Proto": "https, http"},
			want:    "https://edge.example.com",
		},
		{
			name:    "unrecognised forwarded proto falls back to the visible hop",
			target:  "/",
			headers: map[string]string{"X-Forwarded-Proto": "gopher", "X-Forwarded-Host": "shaker.example.com"},
			want:    "http://shaker.example.com",
		},
		{
			name:    "ipv6 literal host",
			target:  "/",
			headers: map[string]string{"X-Forwarded-Host": "[2001:db8::1]:8443", "X-Forwarded-Proto": "https"},
			want:    "https://[2001:db8::1]:8443",
		},
		{
			// Authority injection: a Host carrying a path would otherwise
			// escape the authority it is spliced into.
			name:    "host with a path is refused",
			target:  "/",
			headers: map[string]string{"X-Forwarded-Host": "evil.example/x"},
			want:    "",
		},
		{
			name:    "host with userinfo is refused",
			target:  "/",
			headers: map[string]string{"X-Forwarded-Host": "user@evil.example"},
			want:    "",
		},
		{
			// An empty X-Forwarded-Host must not blank the result: the
			// fallback to Host is what keeps the card correct when a proxy
			// forwards the proto but not the host.
			name:    "empty forwarded host falls back to the request host",
			target:  "/",
			headers: map[string]string{"X-Forwarded-Host": ""},
			want:    "http://example.com",
		},
		{
			name: "no configured value and no request yields empty",
			want: "",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var r *http.Request
			if tc.target != "" {
				r = httptest.NewRequest(http.MethodGet, tc.target, nil)
				if tc.tls {
					r.TLS = &tls.ConnectionState{}
				}
				for k, v := range tc.headers {
					r.Header.Set(k, v)
				}
			}

			if got := resolvePublicBaseURL(tc.configured, r); got != tc.want {
				t.Errorf("resolvePublicBaseURL() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolvePublicBaseURLRefusesEmptyHost(t *testing.T) {
	t.Parallel()

	// Distinct from the table case above: there the header was empty and Host
	// supplied the answer. Here Host itself is empty, so there is no
	// trustworthy origin left and the caller must fall back to a relative URL
	// rather than emit "http://".
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Host = ""

	if got := resolvePublicBaseURL("", r); got != "" {
		t.Errorf("resolvePublicBaseURL() = %q, want empty", got)
	}
}

func TestIsValidAuthority(t *testing.T) {
	t.Parallel()

	valid := []string{
		"example.com",
		"sub.example.com:8443",
		"127.0.0.1:3000",
		"[2001:db8::1]",
		"[2001:db8::1]:443",
		"localhost",
	}
	for _, s := range valid {
		if !isValidAuthority(s) {
			t.Errorf("isValidAuthority(%q) = false, want true", s)
		}
	}

	invalid := []string{
		"",
		"exa mple.com",
		"example.com/path",
		"example.com?x=1",
		"user@evil.example",
		"example.com#frag",
		"exam\nple.com",
		"exam\tple.com",
		"exam\rple.com",
		"http://example.com",
		"example.com;rm -rf /",
		"exa*mple.com",
		"exaémple.com",
		"host\r\nX-Injected: 1",
	}
	for _, s := range invalid {
		if isValidAuthority(s) {
			t.Errorf("isValidAuthority(%q) = true, want false", s)
		}
	}
}
