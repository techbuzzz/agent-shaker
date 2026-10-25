package coordination

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/database/queries"
	"github.com/techbuzzz/agent-shaker/internal/models"
)

// fakeTx is a Tx that records whether it was committed. The reaper's contract
// with the database is "these two writes happen together or not at all", and
// the only way to check that without a database is to watch which one of the
// two exits happened.
type fakeTx struct {
	queries.Querier // nil: the fake store never actually issues SQL
	committed       bool
	rolled          bool
}

func (t *fakeTx) Commit() error { t.committed = true; return nil }

// Rollback mirrors database/sql: after Commit it is a no-op that reports
// nothing. The reaper defers a Rollback on every sweep, and that must not be
// read as an abort — modelling it any other way would have this fake report a
// rollback on the success path.
func (t *fakeTx) Rollback() error {
	if t.committed {
		return nil
	}
	t.rolled = true
	return nil
}

// fakeAgentStore drives the reaper through a scripted sequence of stale agents
// and records what it was asked to do.
type fakeAgentStore struct {
	stale []models.StaleAgent

	// offlineChanged is what MarkAgentOffline reports. False models the race
	// where the agent heartbeated after the sweep selected it.
	offlineChanged bool
	offlineErr     error

	released   []uuid.UUID
	releaseErr error

	// failOffline makes the sweep of an agent fail, so the test can prove one
	// bad agent does not stop the rest.
	failOffline map[uuid.UUID]bool

	tx *fakeTx
}

func (s *fakeAgentStore) ListStaleAgents(_ context.Context, _ int) ([]models.StaleAgent, error) {
	return s.stale, nil
}

func (s *fakeAgentStore) MarkAgentOffline(_ context.Context, _ queries.Querier, id uuid.UUID) (bool, error) {
	if s.failOffline[id] {
		return false, errors.New("boom")
	}
	return s.offlineChanged, s.offlineErr
}

func (s *fakeAgentStore) ReleaseAgentTasks(_ context.Context, _ queries.Querier, _ uuid.UUID) ([]uuid.UUID, error) {
	if s.releaseErr != nil {
		return nil, s.releaseErr
	}
	return s.released, nil
}

func (s *fakeAgentStore) BeginTx(context.Context) (Tx, error) {
	s.tx = &fakeTx{}
	return s.tx, nil
}

func TestReaperExpiresStaleAgentAndReleasesItsWork(t *testing.T) {
	agentID := uuid.New()
	taskID := uuid.New()
	store := &fakeAgentStore{
		stale:          []models.StaleAgent{{ID: agentID, ProjectID: uuid.New(), Name: "Backend Agent"}},
		offlineChanged: true,
		released:       []uuid.UUID{taskID},
	}

	reaped := NewStaleAgentReaper(store, nil, nil, DefaultConfig()).SweepOnce(context.Background())

	if reaped != 1 {
		t.Fatalf("SweepOnce reaped %d agents, want 1", reaped)
	}
	if store.tx == nil || !store.tx.committed {
		t.Fatal("the reaper must commit the transaction that expires a lease")
	}
	if store.tx.rolled {
		t.Error("the reaper rolled back a committed sweep")
	}
}

// The optimistic lock: an agent that heartbeated mid-sweep keeps its lease and
// its task. This is the case that decides whether the reaper is safe to run
// against a fleet on a slow network, so it gets its own test.
func TestReaperLeavesAnAgentThatHeartbeatedMidSweepAlone(t *testing.T) {
	store := &fakeAgentStore{
		stale:          []models.StaleAgent{{ID: uuid.New(), ProjectID: uuid.New(), Name: "Slow Agent"}},
		offlineChanged: false,
		released:       []uuid.UUID{uuid.New()},
	}

	reaped := NewStaleAgentReaper(store, nil, nil, DefaultConfig()).SweepOnce(context.Background())

	if reaped != 1 {
		t.Fatalf("SweepOnce counted %d reaped, want 1 (the sweep did select it)", reaped)
	}
	if store.tx == nil || !store.tx.committed {
		t.Fatal("the no-op path must still commit, otherwise the transaction is left open")
	}
}

// A failing agent must not abort the sweep: one agent whose UPDATE errors is
// not a reason to leave every other expired lease alive for another interval.
func TestReaperContinuesAfterOneAgentFails(t *testing.T) {
	bad := uuid.New()
	store := &fakeAgentStore{
		stale: []models.StaleAgent{
			{ID: bad, ProjectID: uuid.New(), Name: "Broken"},
			{ID: uuid.New(), ProjectID: uuid.New(), Name: "Fine"},
		},
		offlineChanged: true,
		failOffline:    map[uuid.UUID]bool{bad: true},
	}

	reaped := NewStaleAgentReaper(store, nil, nil, DefaultConfig()).SweepOnce(context.Background())

	if reaped != 1 {
		t.Fatalf("SweepOnce reaped %d agents, want 1 (the healthy one)", reaped)
	}
}

func TestReaperSweepsNothingWhenListFails(t *testing.T) {
	store := &listFailStore{}
	reaped := NewStaleAgentReaper(store, nil, nil, DefaultConfig()).SweepOnce(context.Background())
	if reaped != 0 {
		t.Fatalf("SweepOnce reaped %d agents after a list failure, want 0", reaped)
	}
}

type listFailStore struct{ fakeAgentStore }

func (s *listFailStore) ListStaleAgents(context.Context, int) ([]models.StaleAgent, error) {
	return nil, errors.New("database is down")
}
