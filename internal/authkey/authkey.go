// Package authkey mints and digests API keys.
//
// It is its own package because two call sites that must never disagree share
// the rule: the management handler mints a secret and persists only its
// digest, while the request path digests whatever the caller presented and
// looks the row up by that digest. If the two derived the digest differently,
// every request would fail with no obvious place to look.
//
// No password hashing here, deliberately: see migrations/010 for why the
// digest has to be deterministic.
package authkey

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
)

const (
	// Prefix marks a credential as ours. It buys two things: a leaked key can
	// be grepped for in log archives and secret scanners, and the resolver can
	// reject an obviously foreign token without spending a database round trip
	// on it.
	Prefix = "ask_"

	// EntropyBytes is the amount of randomness in a generated key: 32 bytes =
	// 256 bits, the same width as an AES-256 key. Anything lower would make the
	// "the digest is not a password hash, it does not need to be" argument in
	// migrations/010 materially weaker.
	EntropyBytes = 32

	// DisplayLen is how much of the plaintext is stored alongside the row so an
	// operator can tell two keys apart in a list. It is a label, not a
	// credential: 12 characters of 256-bit randomness leak nothing, and the
	// value is fixed so the UI cannot render a longer, more revealing slice.
	DisplayLen = 12
)

// ErrEntropy is returned when the system CSPRNG fails. It is not retried and
// not papered over: issuing a key from a predictable source is worse than
// failing the request.
var ErrEntropy = errors.New("authkey: the system random source is unavailable")

// Generate returns a new plaintext key together with the two values persisted
// beside it: the hex SHA-256 digest used for lookup and a short display prefix.
//
// The plaintext exists in exactly two places in the whole system — here, and in
// the POST /api/keys response body. It is never logged and never stored.
func Generate() (secret, digest, display string, err error) {
	buf := make([]byte, EntropyBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", "", "", ErrEntropy
	}
	// RawURLEncoding: base64url without padding. The alphabet is safe to paste
	// into a header, a query string, a shell variable and a JSON body without
	// escaping, which matters because operators copy these keys by hand.
	secret = Prefix + base64.RawURLEncoding.EncodeToString(buf)
	return secret, Hash(secret), DisplayPrefix(secret), nil
}

// Hash returns the hex-encoded SHA-256 digest of a presented key. This is the
// value stored in api_keys.key_hash and the value the lookup is issued
// against.
//
// Whitespace is trimmed rather than rejected: a key pasted out of a text
// editor routinely carries a trailing newline, and failing that would send
// operators to look for a compromise that never happened.
func Hash(secret string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(secret)))
	return hex.EncodeToString(sum[:])
}

// DisplayPrefix returns the non-secret slice of a key kept for the UI.
//
// Short keys are returned whole rather than sliced out of range: a short key
// cannot be one of ours, and DisplayPrefix is called on presented credentials,
// where a panic on attacker-controlled input would be a denial of service.
func DisplayPrefix(secret string) string {
	if len(secret) <= DisplayLen {
		return secret
	}
	return secret[:DisplayLen]
}

// IsWellFormed reports whether a credential could plausibly have been minted
// by Generate. The resolver uses it to reject a foreign token without a
// database round trip; it is a filter, not a security boundary.
func IsWellFormed(secret string) bool {
	return strings.HasPrefix(strings.TrimSpace(secret), Prefix)
}
