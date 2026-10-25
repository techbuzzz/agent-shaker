package coordination

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/database/queries"
	"github.com/techbuzzz/agent-shaker/internal/models"
)

// fakeDepsStore scripts the dependency graph: which tasks depend on the
// completed one, how many predecessors each still has outstanding, and whether
// the promotion UPDATE would actually move a row.
type fakeDepsStore struct {
	dependents []uuid.UUID
	unfinished map[uuid.UUID]int
	promoteOK  map[uuid.UUID]bool
	blocked    map[uuid.UUID]bool
	countErr   map[uuid.UUID]error
}

func (s *fakeDepsStore) DependentTaskIDs(context.Context, queries.Querier, uuid.UUID) ([]uuid.UUID, error) {
	return s.dependents, nil
}

func (s *fakeDepsStore) CountUnfinishedDependencies(_ context.Context, _ queries.Querier, id uuid.UUID) (int, error) {
	if err := s.countErr[id]; err != nil {
		return 0, err
	}
	return s.unfinished[id], nil
}

func (s *fakeDepsStore) MarkTaskReady(_ context.Context, _ queries.Querier, id uuid.UUID) (bool, error) {
	return s.promoteOK[id], nil
}

func (s *fakeDepsStore) MarkTaskBlocked(_ context.Context, _ queries.Querier, id uuid.UUID) error {
	if s.blocked[id] {
		return nil
	}
	return errors.New("not scripted to block")
}

type fakeTaskStore struct {
	tasks map[uuid.UUID]*models.Task
}

func (s *fakeTaskStore) GetTask(_ context.Context, id uuid.UUID) (*models.Task, error) {
	t, ok := s.tasks[id]
	if !ok {
		return nil, errors.New("task not found")
	}
	return t, nil
}

func TestResolverPromotesOnlyTasksWithNoUnfinishedDependencies(t *testing.T) {
	readyTask := uuid.New()
	stillWaiting := uuid.New()
	store := &fakeDepsStore{
		dependents: []uuid.UUID{readyTask, stillWaiting},
		unfinished: map[uuid.UUID]int{
			readyTask:    0,
			stillWaiting: 2,
		},
		promoteOK: map[uuid.UUID]bool{readyTask: true},
	}
	tasks := &fakeTaskStore{tasks: map[uuid.UUID]*models.Task{
		readyTask: {ID: readyTask, ProjectID: uuid.New(), Title: "Wire the UI"},
	}}

	promoted, err := NewDependencyResolver(store, tasks, nil, nil).ResolveAfterCompletion(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("ResolveAfterCompletion: %v", err)
	}
	if len(promoted) != 1 || promoted[0] != readyTask {
		t.Fatalf("promoted %v, want exactly [%s]", promoted, readyTask)
	}
}

// Two completions can resolve the same dependent at the same time. Only one of
// them moves the row, and only the one that did may report it — otherwise the
// UI announces work that nothing just unblocked.
func TestResolverDoesNotReportAPromotionTheDatabaseRefused(t *testing.T) {
	task := uuid.New()
	store := &fakeDepsStore{
		dependents: []uuid.UUID{task},
		unfinished: map[uuid.UUID]int{task: 0},
		promoteOK:  map[uuid.UUID]bool{task: false},
	}
	tasks := &fakeTaskStore{tasks: map[uuid.UUID]*models.Task{}}

	promoted, err := NewDependencyResolver(store, tasks, nil, nil).ResolveAfterCompletion(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("ResolveAfterCompletion: %v", err)
	}
	if len(promoted) != 0 {
		t.Fatalf("promoted %v, want none when the guarded UPDATE affected no rows", promoted)
	}
}

func TestResolverKeepsGoingWhenOneDependentCannotBeChecked(t *testing.T) {
	broken := uuid.New()
	fine := uuid.New()
	store := &fakeDepsStore{
		dependents: []uuid.UUID{broken, fine},
		unfinished: map[uuid.UUID]int{fine: 0},
		promoteOK:  map[uuid.UUID]bool{fine: true},
		countErr:   map[uuid.UUID]error{broken: errors.New("deadlock")},
	}
	tasks := &fakeTaskStore{tasks: map[uuid.UUID]*models.Task{}}

	promoted, err := NewDependencyResolver(store, tasks, nil, nil).ResolveAfterCompletion(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("ResolveAfterCompletion: %v", err)
	}
	if len(promoted) != 1 || promoted[0] != fine {
		t.Fatalf("promoted %v, want [%s]: one failure must not block the rest", promoted, fine)
	}
}

func TestResolverReleaseKeepsABlockerTaskBlockedWhileDependenciesRemain(t *testing.T) {
	task := uuid.New()
	store := &fakeDepsStore{
		unfinished: map[uuid.UUID]int{task: 1},
		blocked:    map[uuid.UUID]bool{task: true},
	}
	resolver := NewDependencyResolver(store, &fakeTaskStore{}, nil, nil)

	if err := resolver.ReleaseAfterBlockerResolution(context.Background(), nil, task, true); err != nil {
		t.Fatalf("ReleaseAfterBlockerResolution: %v", err)
	}
	if store.promoteOK[task] {
		t.Error("a task with an unfinished dependency must not be promoted to ready")
	}
}

func TestResolverReleaseSendsAResumedTaskBackToWork(t *testing.T) {
	task := uuid.New()
	store := &fakeDepsStore{unfinished: map[uuid.UUID]int{task: 0}}
	resolver := NewDependencyResolver(store, &fakeTaskStore{}, nil, nil)

	if err := resolver.ReleaseAfterBlockerResolution(context.Background(), nil, task, true); err != nil {
		t.Fatalf("ReleaseAfterBlockerResolution: %v", err)
	}
	if store.promoteOK[task] {
		t.Error("a task that was already in progress must stay in progress, not be demoted to ready")
	}
	if store.blocked[task] {
		t.Error("a task with no outstanding dependencies must not be blocked")
	}
}
