package models

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// PersonKind separates the two kinds of identity the system can attribute
// action to.
//
// The distinction is stored, not inferred: a service (a CI job, an MCP client,
// the legacy shared secret) and a human authenticate identically — with a key —
// but only a human may hold approval authority. Keeping it in a column makes
// the rule checkable by a constraint instead of by a naming convention
// somebody can quietly break.
type PersonKind string

const (
	// PersonKindHuman is a person on the team.
	PersonKindHuman PersonKind = "human"

	// PersonKindService is a machine caller: an agent, a CI pipeline, an
	// integration.
	PersonKindService PersonKind = "service"
)

func (k PersonKind) String() string { return string(k) }

// ErrUnknownPersonKind is returned by ParsePersonKind for a value outside the
// vocabulary the people_kind_check constraint enforces. Handlers wrap it so
// the API answers 400 rather than letting Postgres answer 500.
var ErrUnknownPersonKind = errors.New("kind must be human or service")

// ParsePersonKind normalises a requested kind.
//
// An empty value defaults to human, because that is what the caller means when
// they create a person without qualifying it: people are created by hand, in a
// UI, by somebody adding a colleague.
func ParsePersonKind(s string) (PersonKind, error) {
	switch PersonKind(strings.ToLower(strings.TrimSpace(s))) {
	case "":
		return PersonKindHuman, nil
	case PersonKindHuman:
		return PersonKindHuman, nil
	case PersonKindService:
		return PersonKindService, nil
	default:
		return "", ErrUnknownPersonKind
	}
}

// CanApprove reports whether a principal of this kind may hold approval
// authority (see docs/ROADMAP.md, M4). Encoded here rather than at the call
// site so the rule has one definition.
func (k PersonKind) CanApprove() bool { return k == PersonKindHuman }

// Person is a row of the people table: the subject every future audit event,
// approval and attribution will point at.
type Person struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	Kind        PersonKind `json:"kind" db:"kind"`
	DisplayName string     `json:"display_name" db:"display_name"`
	Email       string     `json:"email,omitempty" db:"email"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

// CreatePersonRequest is the POST /api/people body.
//
// Kind and the identifiers are strings rather than typed values because this is
// the wire boundary: an unparsable UUID or an unknown kind has to become a 400
// with a message, not a decode error the handler has to classify by hand.
type CreatePersonRequest struct {
	Kind        string `json:"kind"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
}

// Validation limits for a person.
//
// MaxEmailLength is RFC 5321's limit (the longest address that can exist), not
// an arbitrary number: the column is VARCHAR(320) and anything longer is a
// malformed request, not a person.
const (
	MaxDisplayNameLength = 255
	MaxEmailLength       = 320
)

// ErrInvalidPerson is returned for a request that fails validation. The
// message names the offending field so the UI can show it.
var ErrInvalidPerson = errors.New("invalid person")

// Validate normalises and checks a create request.
//
// It mutates the receiver so the caller cannot accidentally persist the
// untrimmed version — a display name of " Victor " that is stored with the
// spaces is invisible in the UI and unmatchable in a search.
func (r *CreatePersonRequest) Validate() (PersonKind, error) {
	r.DisplayName = strings.TrimSpace(r.DisplayName)
	r.Email = strings.TrimSpace(r.Email)

	switch {
	case r.DisplayName == "":
		return "", errors.New("display_name is required")
	case len([]rune(r.DisplayName)) > MaxDisplayNameLength:
		return "", errors.New("display_name is too long")
	case len(r.Email) > MaxEmailLength:
		return "", errors.New("email is too long")
	case r.Email != "" && !validEmail(r.Email):
		return "", errors.New("email is not a valid address")
	}

	kind, err := ParsePersonKind(r.Kind)
	if err != nil {
		return "", err
	}
	return kind, nil
}

// validEmail is a deliberately shallow check: exactly one "@", something on
// both sides, no whitespace. Full RFC 5322 conformance is a parser, not a
// predicate, and rejecting a real address is worse than accepting an
// undeliverable one.
func validEmail(s string) bool {
	at := strings.Index(s, "@")
	if at <= 0 || at == len(s)-1 {
		return false
	}
	return !strings.ContainsAny(s, " \t\r\n") && strings.Count(s, "@") == 1
}
