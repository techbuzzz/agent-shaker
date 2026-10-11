package queries

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/techbuzzz/agent-shaker/internal/models"
)

// ErrPersonEmailTaken is returned when a person is created with an address
// that already exists, case-insensitively.
//
// It is a sentinel rather than a string so the handler can answer 409 Conflict
// without pattern-matching a driver message, and without pulling a driver type
// into the HTTP layer.
var ErrPersonEmailTaken = errors.New("a person with this email already exists")

// EmailUniqueConstraint is the partial unique index in migration 009. Named in
// one place so a rename in the migration is a compile error here rather than a
// silent fall-through to a 500.
const EmailUniqueConstraint = "idx_people_email_unique"

// PeopleStore is the typed query layer for the people table.
type PeopleStore struct {
	q Querier
}

// NewPeopleStore returns a store bound to the supplied Querier.
func NewPeopleStore(q Querier) *PeopleStore {
	return &PeopleStore{q: q}
}

// Available reports whether the store has a usable Querier, so handlers can
// answer 503 instead of dereferencing a nil interface when the server runs
// without a database.
func (s *PeopleStore) Available() bool { return s.q != nil }

// CreatePerson inserts a row.
//
// An empty email is stored as NULL rather than an empty string: the uniqueness
// index is partial on IS NOT NULL, so an empty string would occupy a single
// shared slot and make the second email-less person collide with the first.
func (s *PeopleStore) CreatePerson(ctx context.Context, p *models.Person) error {
	var email any
	if p.Email != "" {
		email = p.Email
	}

	if _, err := s.q.ExecContext(ctx, `
		INSERT INTO people (id, kind, display_name, email, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, p.ID, p.Kind, p.DisplayName, email, p.CreatedAt); err != nil {
		if isUniqueViolation(err, EmailUniqueConstraint) {
			return fmt.Errorf("%w: %s", ErrPersonEmailTaken, p.Email)
		}
		return fmt.Errorf("insert person: %w", err)
	}
	return nil
}

// GetPerson returns one person by id.
func (s *PeopleStore) GetPerson(ctx context.Context, id uuid.UUID) (*models.Person, error) {
	var (
		p     models.Person
		email sql.NullString
	)
	err := s.q.QueryRowContext(ctx, `
		SELECT id, kind, display_name, email, created_at
		FROM people WHERE id = $1
	`, id).Scan(&p.ID, &p.Kind, &p.DisplayName, &email, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("person %s not found", id)
		}
		return nil, fmt.Errorf("select person: %w", err)
	}
	p.Email = email.String
	return &p, nil
}

// ListPeople returns every person ordered by display name.
//
// Alphabetical rather than newest-first: this list is looked up by a human
// scanning for a colleague, and a name is the only column they can scan. There
// is no created_at index to exploit either, which keeps the sort on a table
// that holds one row per team member.
func (s *PeopleStore) ListPeople(ctx context.Context) ([]models.Person, error) {
	rows, err := s.q.QueryContext(ctx, `
		SELECT id, kind, display_name, email, created_at
		FROM people ORDER BY display_name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list people: %w", err)
	}
	defer rows.Close()

	out := make([]models.Person, 0, 16)
	for rows.Next() {
		var (
			p     models.Person
			email sql.NullString
		)
		if err := rows.Scan(&p.ID, &p.Kind, &p.DisplayName, &email, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan person: %w", err)
		}
		p.Email = email.String
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating people: %w", err)
	}
	return out, nil
}

// isUniqueViolation reports whether err is a Postgres unique-constraint
// violation (SQLSTATE 23505) raised by the named constraint.
//
// Matching on the constraint name rather than "any unique violation" is what
// lets the caller distinguish "this email is taken" from "this id is taken",
// which have different HTTP answers.
func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == "23505" && pgErr.ConstraintName == constraint
}
