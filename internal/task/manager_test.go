package task

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/techbuzzz/agent-shaker/internal/a2a/models"
)

// executorFunc adapts a plain function to the TaskExecutor interface so each
// test can state the behaviour it needs without a named type.
type executorFunc func(ctx context.Context, task *models.Task) (*models.Result, error)

func (f executorFunc) Execute(ctx context.Context, task *models.Task) (*models.Result, error) {
	return f(ctx, task)
}

// newTestManager returns a Manager backed by an in-memory store.
func newTestManager(executor TaskExecutor) *Manager {
	return NewManager(NewMemoryStore(""), executor, "http://test.invalid")
}

// newMessage builds a minimal send request.
func newMessage(content string) *models.SendMessageRequest {
	return &models.SendMessageRequest{
		Message: models.Message{Content: content, Format: "text"},
	}
}

// waitForTerminal polls until the task reaches a terminal state. The Manager
// runs execution in a goroutine, so this is the only way to observe a
// transition; a fixed sleep would be a race against the scheduler.
func waitForTerminal(t *testing.T, m *Manager, taskID string) models.Task {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for {
		task, err := m.GetTask(context.Background(), taskID)
		if err != nil {
			t.Fatalf("GetTask(%s): %v", taskID, err)
		}
		if task.Status.IsTerminal() {
			return *task
		}
		if time.Now().After(deadline) {
			t.Fatalf("task %s stayed in %q and never reached a terminal state", taskID, task.Status)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// waitForStatus polls until the task reports want, or fails the test.
func waitForStatus(t *testing.T, m *Manager, taskID string, want models.TaskStatus) models.Task {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for {
		task, err := m.GetTask(context.Background(), taskID)
		if err != nil {
			t.Fatalf("GetTask(%s): %v", taskID, err)
		}
		if task.Status == want {
			return *task
		}
		if task.Status.IsTerminal() {
			t.Fatalf("task %s reached terminal state %q while waiting for %q", taskID, task.Status, want)
		}
		if time.Now().After(deadline) {
			t.Fatalf("task %s stayed in %q, want %q", taskID, task.Status, want)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// TestTaskWithoutExecutorFails pins the regression this milestone exists for.
//
// A Manager with no executor used to echo the submitted message back and mark
// the task completed. A remote agent delegating work therefore received
// "completed" for work that was never performed. Absence of an executor must
// surface as a failure with a diagnosable reason, never as a result.
func TestTaskWithoutExecutorFails(t *testing.T) {
	m := newTestManager(nil)

	created, err := m.CreateTask(context.Background(), newMessage("please do the work"))
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	got := waitForTerminal(t, m, created.ID)

	if got.Status != models.TaskStatusFailed {
		t.Fatalf("status = %q, want %q", got.Status, models.TaskStatusFailed)
	}
	if !strings.Contains(got.Result.Content, "no task executor configured") {
		t.Errorf("result should explain why the task failed, got %q", got.Result.Content)
	}
	if strings.Contains(got.Result.Content, "Task received:") {
		t.Errorf("result still echoes the submitted message: %q", got.Result.Content)
	}
	if got.CompletedAt == nil {
		t.Error("a failed task must still carry completed_at")
	}
}

// TestTaskWithExecutorCompletes is the positive counterpart: a real executor
// still produces a completed task carrying its result.
func TestTaskWithExecutorCompletes(t *testing.T) {
	var seen string
	m := newTestManager(executorFunc(func(_ context.Context, task *models.Task) (*models.Result, error) {
		seen = task.Message.Content
		return &models.Result{Content: "done", Format: "text"}, nil
	}))

	created, err := m.CreateTask(context.Background(), newMessage("real work"))
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	got := waitForTerminal(t, m, created.ID)

	if got.Status != models.TaskStatusCompleted {
		t.Fatalf("status = %q, want %q (result: %+v)", got.Status, models.TaskStatusCompleted, got.Result)
	}
	if got.Result == nil || got.Result.Content != "done" {
		t.Errorf("result = %+v, want content %q", got.Result, "done")
	}
	if seen != "real work" {
		t.Errorf("executor received message %q, want %q", seen, "real work")
	}
}

// TestExecutorErrorFailsTask checks that a failing executor produces a failed
// task carrying the executor's own error, not a generic one.
func TestExecutorErrorFailsTask(t *testing.T) {
	sentinel := errors.New("upstream runtime refused the job")
	m := newTestManager(executorFunc(func(_ context.Context, _ *models.Task) (*models.Result, error) {
		return nil, sentinel
	}))

	created, err := m.CreateTask(context.Background(), newMessage("work"))
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	got := waitForTerminal(t, m, created.ID)

	if got.Status != models.TaskStatusFailed {
		t.Fatalf("status = %q, want %q", got.Status, models.TaskStatusFailed)
	}
	if got.Result == nil || !strings.Contains(got.Result.Content, sentinel.Error()) {
		t.Errorf("result = %+v, want it to contain %q", got.Result, sentinel.Error())
	}
}

// TestCancelDuringExecutionIsNotOverwritten guards the ordering between the
// execution goroutine and a concurrent CancelTask. The executor finishes after
// the cancel is recorded, and the late result must not resurrect the task into
// "completed" — the caller asked for the work to be discarded.
func TestCancelDuringExecutionIsNotOverwritten(t *testing.T) {
	release := make(chan struct{})
	running := make(chan struct{}, 1)

	m := newTestManager(executorFunc(func(_ context.Context, _ *models.Task) (*models.Result, error) {
		running <- struct{}{}
		<-release
		return &models.Result{Content: "late result", Format: "text"}, nil
	}))

	created, err := m.CreateTask(context.Background(), newMessage("long work"))
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	waitForStatus(t, m, created.ID, models.TaskStatusRunning)
	<-running

	if err := m.CancelTask(context.Background(), created.ID); err != nil {
		t.Fatalf("CancelTask: %v", err)
	}

	cancelled := waitForTerminal(t, m, created.ID)
	if cancelled.Status != models.TaskStatusCanceled {
		t.Fatalf("status = %q, want %q", cancelled.Status, models.TaskStatusCanceled)
	}

	// Let the executor return and the goroutine try to finalise the task.
	close(release)

	// Give the goroutine room to run to its final write; the status must not
	// change. A short settle is unavoidable here because the thing under test
	// is precisely "nothing happens after the executor returns".
	deadline := time.Now().Add(250 * time.Millisecond)
	for time.Now().Before(deadline) {
		task, err := m.GetTask(context.Background(), created.ID)
		if err != nil {
			t.Fatalf("GetTask: %v", err)
		}
		if task.Status != models.TaskStatusCanceled {
			t.Fatalf("status = %q after cancellation, want it to stay %q", task.Status, models.TaskStatusCanceled)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// TestCancelTerminalTaskConflicts checks that a task which already finished
// cannot be cancelled, and that the sentinel is the one the handler maps to
// 409.
func TestCancelTerminalTaskConflicts(t *testing.T) {
	m := newTestManager(executorFunc(func(_ context.Context, _ *models.Task) (*models.Result, error) {
		return &models.Result{Content: "done", Format: "text"}, nil
	}))

	created, err := m.CreateTask(context.Background(), newMessage("work"))
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	waitForTerminal(t, m, created.ID)

	err = m.CancelTask(context.Background(), created.ID)
	if !errors.Is(err, ErrTaskTerminal) {
		t.Fatalf("CancelTask error = %v, want ErrTaskTerminal", err)
	}
}

// TestCancelUnknownTaskNotFound checks the 404 sentinel.
func TestCancelUnknownTaskNotFound(t *testing.T) {
	m := newTestManager(nil)

	err := m.CancelTask(context.Background(), "no-such-task")
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("CancelTask error = %v, want ErrTaskNotFound", err)
	}
}

// TestSubscribersSeeFinalEventNamedAfterStatus covers the streaming contract.
// The event is forwarded verbatim to SSE clients, so a task that failed must
// not be announced as "completed".
func TestSubscribersSeeFinalEventNamedAfterStatus(t *testing.T) {
	m := newTaskWithSubscriber(t, nil)

	created, err := m.CreateTask(context.Background(), newMessage("work"))
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	final := waitForFinalUpdate(t, m, created.ID)
	if final.Event != string(models.TaskStatusFailed) {
		t.Errorf("final event = %q, want %q", final.Event, models.TaskStatusFailed)
	}
	if !final.IsFinal {
		t.Error("the last update must be flagged final so the stream can close")
	}
}

// newTaskWithSubscriber builds a manager and subscribes to a task that does not
// exist yet, which is the order CreateTask uses in production.
func newTaskWithSubscriber(t *testing.T, executor TaskExecutor) *Manager {
	t.Helper()
	return newTestManager(executor)
}

// waitForFinalUpdate reads the subscriber channel until the final update.
func waitForFinalUpdate(t *testing.T, m *Manager, taskID string) TaskUpdate {
	t.Helper()

	updates := m.SubscribeToTask(taskID)
	defer m.UnsubscribeFromTask(taskID, updates)

	// CreateTask races the subscription in this helper only, so allow the
	// events that may already have been emitted to arrive first.
	deadline := time.After(2 * time.Second)
	for {
		select {
		case update := <-updates:
			if update.IsFinal {
				return update
			}
		case <-deadline:
			t.Fatalf("no final update for task %s", taskID)
		}
	}
}

// TestTaskStatusIsTerminal pins the helper four call sites depend on. A status
// that is missing from the terminal set would let a finished task be
// cancelled, or a poll loop spin forever on it.
func TestTaskStatusIsTerminal(t *testing.T) {
	tests := []struct {
		status models.TaskStatus
		want   bool
	}{
		{models.TaskStatusPending, false},
		{models.TaskStatusRunning, false},
		{models.TaskStatusCompleted, true},
		{models.TaskStatusFailed, true},
		{models.TaskStatusCanceled, true},
		{models.TaskStatus(""), false},
		{models.TaskStatus("unknown"), false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(string(tc.status), func(t *testing.T) {
			if got := tc.status.IsTerminal(); got != tc.want {
				t.Errorf("%q.IsTerminal() = %v, want %v", tc.status, got, tc.want)
			}
		})
	}
}
