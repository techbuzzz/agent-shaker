package middleware

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"golang.org/x/time/rate"
)

// RateFromEnv parses the env var as a rate.Limit (tokens per second). On
// failure or absence it returns the supplied default.
func RateFromEnv(envName string, def rate.Limit) rate.Limit {
	v := os.Getenv(envName)
	if v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil || f <= 0 {
		return def
	}
	return rate.Limit(f)
}

// IntFromEnv parses the env var as a positive int. On failure or absence it
// returns the supplied default.
func IntFromEnv(envName string, def int) int {
	v := os.Getenv(envName)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

// SkipPaths returns a predicate that returns true when the request's URL
// path exactly matches any of the supplied paths. Useful for exempting
// /healthz, /readyz, /metrics from the rate limiter.
func SkipPaths(paths ...string) func(*http.Request) bool {
	set := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		set[p] = struct{}{}
	}
	return func(r *http.Request) bool {
		_, ok := set[r.URL.Path]
		return ok
	}
}

// CORSOriginsFromEnv builds the CORS allow-list from a comma-separated env
// var plus the supplied defaults. Empty tokens are skipped.
func CORSOriginsFromEnv(envName string, defaults ...string) []string {
	merged := make([]string, 0, len(defaults)+8)
	merged = append(merged, defaults...)
	if v := os.Getenv(envName); v != "" {
		for _, o := range strings.Split(v, ",") {
			o = strings.TrimSpace(o)
			if o != "" {
				merged = append(merged, o)
			}
		}
	}
	return merged
}
