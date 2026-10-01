package main

import (
	"strings"
	"testing"
)

// TestDatabaseHost covers both DSN dialects the project accepts. The keyword
// form is what libpq also understands, and operators do paste those in by
// hand, so a regression here would silently re-enable the production guard
// against a public host.
func TestDatabaseHost(t *testing.T) {
	tests := []struct {
		name string
		dsn  string
		want string
	}{
		{
			name: "url form with credentials and port",
			dsn:  "postgres://mcp:secret@db.example.com:5432/mcp_tracker?sslmode=require",
			want: "db.example.com",
		},
		{
			name: "url form without credentials",
			dsn:  "postgres://localhost/mcp_tracker",
			want: "localhost",
		},
		{
			name: "postgresql scheme",
			dsn:  "postgresql://user:pw@10.0.0.5:5432/db",
			want: "10.0.0.5",
		},
		{
			name: "ipv6 literal",
			dsn:  "postgres://user:pw@[fd00::1]:5432/db",
			want: "fd00::1",
		},
		{
			name: "keyword value form",
			dsn:  "host=db.internal port=5432 user=mcp dbname=mcp_tracker sslmode=disable",
			want: "db.internal",
		},
		{
			name: "keyword form with quoted host",
			dsn:  "host='rds.example.com' user=mcp",
			want: "rds.example.com",
		},
		{
			name: "empty dsn",
			dsn:  "",
			want: "",
		},
		{
			name: "unparseable input yields empty rather than guessing",
			dsn:  "://///",
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := databaseHost(tc.dsn); got != tc.want {
				t.Errorf("databaseHost(%q) = %q, want %q", tc.dsn, got, tc.want)
			}
		})
	}
}

// TestIsPrivateHost pins the exact boundary the production TLS guard depends
// on. A false negative (private misclassified as public) breaks the documented
// docker-compose deployment; a false positive (public misclassified as
// private) silently ships credentials in cleartext, which is far worse.
func TestIsPrivateHost(t *testing.T) {
	tests := []struct {
		name string
		host string
		want bool
	}{
		{"loopback name", "localhost", true},
		{"loopback ipv4", "127.0.0.1", true},
		{"docker compose service name", "postgres", true},
		{"single label host", "db", true},
		{"rfc1918 10/8", "10.1.2.3", true},
		{"rfc1918 172.16/12", "172.17.0.2", true},
		{"rfc1918 192.168/16", "192.168.1.10", true},
		{"unique local ipv6", "fd00::1", true},
		{"link local ipv6 with zone", "fe80::1%eth0", true},
		{"mdns suffix", "db.local", true},
		{"internal suffix", "db.internal", true},

		// The cases that must be treated as PUBLIC — these are what the guard
		// exists to catch.
		{"public dns name", "rds.amazonaws.com", false},
		{"public dns name with subdomain chain", "db.prod.example.com", false},
		{"public ipv4", "52.1.2.3", false},
		{"public ipv6", "2606:4700::1111", false},

		// Unparseable must fail closed, not open.
		{"empty host", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isPrivateHost(tc.host); got != tc.want {
				t.Errorf("isPrivateHost(%q) = %v, want %v", tc.host, got, tc.want)
			}
		})
	}
}

// TestProductionGuardDecision encodes the actual boot decision as a table so
// the compose file's documented topology is covered by a test rather than by
// a comment.
func TestProductionGuardDecision(t *testing.T) {
	const (
		envProduction = "production"
		insecure      = "sslmode=disable"
	)

	tests := []struct {
		name       string
		env        string
		dsn        string
		wantRefuse bool
	}{
		{
			name:       "compose topology: private bridge network is allowed",
			env:        envProduction,
			dsn:        "postgres://mcp:secret@postgres:5432/mcp_tracker?" + insecure,
			wantRefuse: false,
		},
		{
			name:       "managed rds over the public internet is refused",
			env:        envProduction,
			dsn:        "postgres://u:p@db.abc.us-east-1.rds.amazonaws.com:5432/app?" + insecure,
			wantRefuse: true,
		},
		{
			name:       "tls enabled is always fine",
			env:        envProduction,
			dsn:        "postgres://u:p@db.abc.us-east-1.rds.amazonaws.com:5432/app?sslmode=verify-full",
			wantRefuse: false,
		},
		{
			name:       "guard only applies in production",
			env:        "development",
			dsn:        "postgres://u:p@db.example.com:5432/app?" + insecure,
			wantRefuse: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			refuse := tc.env == envProduction &&
				strings.Contains(tc.dsn, insecure) &&
				!isPrivateHost(databaseHost(tc.dsn))
			if refuse != tc.wantRefuse {
				t.Errorf("refuse = %v, want %v (env=%q dsn=%q)", refuse, tc.wantRefuse, tc.env, tc.dsn)
			}
		})
	}
}
