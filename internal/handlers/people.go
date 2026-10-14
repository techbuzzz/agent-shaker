package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/database/queries"
	"github.com/techbuzzz/agent-shaker/internal/httpx"
	"github.com/techbuzzz/agent-shaker/internal/models"
)

// PeopleHandler exposes the principal directory.
//
// A principal is the "who" behind an API key: a human approver or a service.
// The distinction is stored rather than inferred, because it decides whether a
// principal may hold approval rights (models.PersonKind.CanApprove) and a
// directory of principals is what makes a key list legible — a bare uuid in a
// key inventory tells an operator nothing.
//
// Gated by models.ScopeAdmin at the route table, like the key surface.
type PeopleHandler struct {
	store *queries.PeopleStore
}

func NewPeopleHandler(store *queries.PeopleStore) *PeopleHandler {
	return &PeopleHandler{store: store}
}

// ListPeople returns the principal directory, ordered by display name.
//
// GET /api/people
func (h *PeopleHandler) ListPeople(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		httpx.WriteError(w, r, fmt.Errorf("people store: %w", httpx.ErrUnavailable))
		return
	}

	people, err := h.store.ListPeople(r.Context())
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("list people: %w", err))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"people": people})
}

// CreatePerson adds a principal to the directory.
//
// POST /api/people {"kind": "human", "display_name": "Victor", "email": "..."}
//
// The kind is validated by the request type rather than by a handler-local
// rule, so "human or service" is enforced identically wherever a person is
// created.
func (h *PeopleHandler) CreatePerson(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		httpx.WriteError(w, r, fmt.Errorf("people store: %w", httpx.ErrUnavailable))
		return
	}

	var req models.CreatePersonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid request body: %w: %w", httpx.ErrBadRequest, err))
		return
	}

	kind, err := req.Validate()
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("validate person: %w: %w", httpx.ErrBadRequest, err))
		return
	}

	person := models.Person{
		ID:          uuid.New(),
		Kind:        kind,
		DisplayName: req.DisplayName,
		Email:       req.Email,
		CreatedAt:   time.Now().UTC(),
	}
	if err := h.store.CreatePerson(r.Context(), &person); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("create person: %w", err))
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, person)
}

// GetPerson returns one principal.
//
// GET /api/people/{id}
func (h *PeopleHandler) GetPerson(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		httpx.WriteError(w, r, fmt.Errorf("people store: %w", httpx.ErrUnavailable))
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("id must be a uuid: %w: %w", httpx.ErrBadRequest, err))
		return
	}

	person, err := h.store.GetPerson(r.Context(), id)
	if err != nil {
		// Distinguish "no such principal" from "the directory is unreachable":
		// without this a missing id surfaces as a 500, which tells the caller
		// to retry a request that can never succeed.
		if errors.Is(err, models.ErrPersonNotFound) {
			httpx.WriteError(w, r, fmt.Errorf("person: %w", httpx.ErrNotFound))
			return
		}
		httpx.WriteError(w, r, fmt.Errorf("get person: %w", err))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, person)
}
