package hooks

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/techbuzzz/agent-shaker/internal/models"
)

// Matcher decides whether a hook applies to a payload.
//
// The field is a dotted path into the event payload, so "title" matches the
// task title and "actor.name" matches the acting agent's name. A payload that
// does not carry the field does not match: a hook that fires on "the field is
// missing" is a hook that fires on everything.
type Matcher struct {
	field   string
	pattern *regexp.Regexp
}

// CompileMatcher compiles a hook matcher.
//
// A nil matcher matches every event of its type, which is the common case:
// most hooks care about the event, not about one field of it.
func CompileMatcher(m *models.HookMatcher) (*Matcher, error) {
	if m == nil {
		return nil, nil
	}

	field := strings.TrimSpace(m.Field)
	if field == "" {
		return nil, errors.New("hook matcher needs a field")
	}

	re, err := regexp.Compile(m.Pattern)
	if err != nil {
		return nil, fmt.Errorf("compile hook matcher pattern: %w", err)
	}

	return &Matcher{field: field, pattern: re}, nil
}

// Matches reports whether the payload's field matches the pattern.
func (m *Matcher) Matches(payload map[string]any) bool {
	if m == nil {
		return true
	}

	value, ok := lookupPath(payload, m.field)
	if !ok {
		return false
	}
	return m.pattern.MatchString(value)
}

// lookupPath walks a dotted path through a nested payload and returns the
// value at the end as a string.
func lookupPath(payload map[string]any, path string) (string, bool) {
	var current any = payload

	for _, segment := range strings.Split(path, ".") {
		node, ok := current.(map[string]any)
		if !ok {
			return "", false
		}
		current, ok = node[segment]
		if !ok {
			return "", false
		}
	}

	switch value := current.(type) {
	case string:
		return value, true
	case nil:
		return "", true
	default:
		return fmt.Sprintf("%v", value), true
	}
}
