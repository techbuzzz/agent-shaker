package main

import (
	"net"
	"net/url"
	"strings"
)

// databaseHost extracts the hostname from a Postgres connection string.
//
// It handles both URL-style DSNs (postgres://user:pass@host:5432/db?sslmode=)
// and libpq keyword/value DSNs (host=... port=... dbname=...). Returns "" when
// the host cannot be determined — callers must treat "" as unsafe.
func databaseHost(dsn string) string {
	if dsn == "" {
		return ""
	}

	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		if u, err := url.Parse(dsn); err == nil {
			return u.Hostname()
		}
		return ""
	}

	// Keyword/value form: scan for a bare `host=` token.
	for _, field := range strings.Fields(dsn) {
		if k, v, ok := strings.Cut(field, "="); ok && strings.EqualFold(k, "host") {
			return strings.Trim(v, "'\"")
		}
	}
	return ""
}

// isPrivateHost reports whether traffic to host stays on a private or loopback
// path, where TLS provides no meaningful confidentiality guarantee anyway and
// sslmode=disable is a deliberate, defensible choice.
//
// Loopback, RFC1918 ranges, link-local, unique-local IPv6, and the ".local" /
// Docker-compose service-name forms all qualify. Everything else — including
// the empty string, which means "could not parse" — is treated as public.
func isPrivateHost(host string) bool {
	if host == "" {
		return false
	}

	// Strip an IPv6 zone identifier (fe80::1%eth0) before parsing.
	if i := strings.IndexByte(host, '%'); i > 0 {
		host = host[:i]
	}

	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback() ||
			ip.IsPrivate() ||
			ip.IsLinkLocalUnicast() ||
			ip.IsLinkLocalMulticast() ||
			ip.IsUnspecified()
	}

	// Docker Compose service names ("postgres", "db") and single-label hostnames
	// resolve inside the container network, not across the public internet.
	if !strings.Contains(host, ".") && !strings.Contains(host, ":") {
		return true
	}
	if strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") {
		return true
	}

	// A public DNS name resolves over the internet.
	return false
}
