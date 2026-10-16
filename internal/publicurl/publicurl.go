// Package publicurl resolves the origin that a machine outside this deployment
// should use to reach the server.
//
// It exists because two surfaces now need the same answer for the same reason:
// the A2A agent card advertises where to delegate work, and the generated MCP
// configuration has to tell an agent running on a developer's laptop which URL
// to call. Both are generated per request and both are read by something that
// is not this process, so an origin derived from the wrong hop sends a real
// client to a dead address.
//
// The logic is unchanged from where it started life inside internal/a2a/server;
// only its home moved, because a second caller is no longer a good reason to
// export it from a package about A2A.
package publicurl

import (
	"net/http"
	"strings"
)

// Resolve returns the origin that external clients should use to reach this
// server, as observed on the request `r`.
//
// Why this exists: the A2A agent card, artifact URLs and generated MCP configs
// are the parts of the API whose whole purpose is to tell another machine "here
// is my address". A hardcoded default (`http://localhost:8080`) is correct only
// for a developer running the binary directly. In the containerised topology the
// Go service publishes no host port and the public origin is the TLS edge, so a
// card that advertises `localhost:8080` sends every client to a dead address.
//
// Precedence:
//
//  1. `configured` — the operator's explicit BASE_URL. Authoritative when set,
//     because it is the only way to express an origin that is not derivable
//     from a request (a separate API hostname, say).
//  2. X-Forwarded-Proto / X-Forwarded-Host, when a reverse proxy set them.
//     Caddy sets both; without them a TLS-terminated deployment would advertise
//     `http://` because the hop to Caddy is the only TLS hop.
//  3. The request's own scheme and Host.
//
// The result never has a trailing slash, so callers can append a path directly.
// It may be empty when no trustworthy origin can be determined; callers then
// emit root-relative URLs, which are still usable by a client that reached the
// server on the same origin.
//
// Trust note: X-Forwarded-* is client-settable, so a caller can influence the
// advertised origin. That is acceptable here because these responses are
// generated per request, are never cached, and already require an API key — the
// worst case is a caller poisoning a card it is the sole reader of. The
// operator escape hatch is BASE_URL, which short-circuits all of it. Deployments
// that expose this service directly to the internet rather than behind a proxy
// should set BASE_URL explicitly; the supplied topology does not, because the
// service is internal-only.
func Resolve(configured string, r *http.Request) string {
	if c := strings.TrimRight(strings.TrimSpace(configured), "/"); c != "" {
		return c
	}
	if r == nil {
		return ""
	}

	scheme := firstForwardedValue(r.Header.Get("X-Forwarded-Proto"))
	if scheme != "http" && scheme != "https" {
		// Absent, or not a scheme we recognise: fall back to the hop we can see.
		scheme = "http"
		if r.TLS != nil {
			scheme = "https"
		}
	}

	host := firstForwardedValue(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = r.Host
	}
	if !IsValidAuthority(host) {
		// A Host header is attacker-controlled. Anything that is not a bare
		// host[:port] would let the value break out of the URL it is spliced
		// into (`evil.example/x`, `user@evil.example`, an embedded newline),
		// so refuse rather than emit a malformed or injected URL.
		return ""
	}

	return scheme + "://" + host
}

// firstForwardedValue returns the first entry of a comma-separated
// X-Forwarded-* list, which is the value added by the proxy closest to the
// client. Later entries are added by proxies further back and, in a chain, can
// be attacker-supplied.
func firstForwardedValue(v string) string {
	if i := strings.IndexByte(v, ','); i >= 0 {
		v = v[:i]
	}
	return strings.TrimSpace(v)
}

// IsValidAuthority reports whether s is a bare host:port suitable for splicing
// into a URL authority position.
func IsValidAuthority(s string) bool {
	if s == "" || len(s) > 255 {
		return false
	}
	// Reject anything that could terminate the authority or inject a new
	// component. The allow-list is deliberately strict: a real deployment only
	// needs DNS names, IPv6 literals in brackets, and an optional port.
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '.', r == '-', r == ':', r == '[', r == ']':
		default:
			return false
		}
	}
	return true
}
