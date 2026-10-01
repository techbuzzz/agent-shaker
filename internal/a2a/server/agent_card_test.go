package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/techbuzzz/agent-shaker/internal/a2a/models"
)

func TestAgentCardAdvertisesRequestOrigin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		configured string
		authReq    bool
		headers    map[string]string
		wantURL    string
		wantIcon   string
	}{
		{
			name:     "behind a tls edge",
			authReq:  true,
			headers:  map[string]string{"X-Forwarded-Proto": "https", "X-Forwarded-Host": "shaker.example.com"},
			wantURL:  "https://shaker.example.com/a2a/v1",
			wantIcon: "https://shaker.example.com/favicon.ico",
		},
		{
			name:       "operator override wins",
			configured: "https://api.example.com",
			authReq:    true,
			headers:    map[string]string{"X-Forwarded-Host": "edge.example.com"},
			wantURL:    "https://api.example.com/a2a/v1",
			wantIcon:   "https://api.example.com/favicon.ico",
		},
		{
			// No provenance for any origin: URLs stay root-relative rather than
			// asserting a scheme and host that were never established.
			name:     "no provenance yields relative urls",
			authReq:  true,
			wantURL:  "/a2a/v1",
			wantIcon: "/favicon.ico",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := NewAgentCardHandler("1.0.0", tc.configured, tc.authReq)
			r := httptest.NewRequest(http.MethodGet, "/.well-known/agent-card.json", nil)
			if len(tc.headers) == 0 {
				r.Host = "" // no origin provenance at all
			}
			for k, v := range tc.headers {
				r.Header.Set(k, v)
			}

			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, r)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}

			var card models.AgentCard
			if err := json.NewDecoder(rec.Body).Decode(&card); err != nil {
				t.Fatalf("decode card: %v", err)
			}
			if card.URL != tc.wantURL {
				t.Errorf("card.URL = %q, want %q", card.URL, tc.wantURL)
			}
			if card.IconURL != tc.wantIcon {
				t.Errorf("card.IconURL = %q, want %q", card.IconURL, tc.wantIcon)
			}
		})
	}
}

// TestAgentCardAuthSchemeTracksMiddleware guards the regression that motivated
// making this dynamic: the card used to hardcode `none`, which was a straight
// lie once API-key auth landed and left every conforming A2A client to send no
// credential and collect a 401.
func TestAgentCardAuthSchemeTracksMiddleware(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		authReq     bool
		wantScheme  string
		wantInBlock bool
	}{
		{name: "auth enabled", authReq: true, wantScheme: "apiKey"},
		{name: "auth disabled", authReq: false, wantScheme: "none"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			card := NewAgentCardHandler("1.0.0", "", tc.authReq).authSchemes()
			if len(card) != 1 {
				t.Fatalf("got %d auth schemes, want 1", len(card))
			}
			if card[0].Scheme != tc.wantScheme {
				t.Errorf("scheme = %q, want %q", card[0].Scheme, tc.wantScheme)
			}
			if card[0].Description == "" {
				t.Error("description must not be empty: it is the only place a client learns how to send the credential")
			}
		})
	}
}
