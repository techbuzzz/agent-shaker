package queries

import (
	"database/sql"
	"database/sql/driver"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/models"
)

// fakeRow hands scanTaskRow a fixed column list, with nil for the nullable
// ones. It exists because the alternative — asserting against a live database —
// would not catch this at all: the bug is in how the scanner treats NULL, and
// every row that has it is invisible to a test that only looks at fresh data.
type fakeRow struct {
	values  []driver.Value
	scanErr error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	for i, d := range dest {
		if i >= len(r.values) {
			break
		}
		v := r.values[i]
		switch p := d.(type) {
		case *sql.NullString:
			p.String, p.Valid = "", false
			if v != nil {
				p.String, p.Valid = v.(string), true
			}
		case *sql.NullTime:
			p.Time, p.Valid = time.Time{}, false
			if v != nil {
				p.Time, p.Valid = v.(time.Time), true
			}
		case *sql.NullInt64:
			p.Int64, p.Valid = 0, false
			if v != nil {
				p.Int64, p.Valid = v.(int64), true
			}
		case *uuid.UUID:
			*p = v.(uuid.UUID)
		case **uuid.UUID:
			// A nullable uuid column arrives as a nil driver.Value, which is the
			// case that broke the listing: scanning NULL straight into the
			// pointer target is what database/sql rejects.
			if v == nil {
				*p = nil
				continue
			}
			id := v.(uuid.UUID)
			*p = &id
		case *string:
			if v == nil {
				*p = ""
			} else {
				*p = v.(string)
			}
		default:
			// pq.StringArray and anything else: leave the zero value.
		}
	}
	return nil
}

func TestScanTaskRowReadsTasksThatHaveNoHandoff(t *testing.T) {
	id := uuid.New()
	projectID := uuid.New()
	createdBy := uuid.New()
	now := time.Now().UTC()

	// Columns exactly as SELECTed by ListTasks/GetTask, with NULL for
	// description, priority, output, predecessor_task_id and handoff_context —
	// the shape of every task that predates the Coordination Plane.
	row := fakeRow{values: []driver.Value{
		id, projectID, "Ship the form layer", nil, "in_progress", nil,
		createdBy, nil, nil, nil, nil, now, now, nil, nil,
	}}

	var task models.Task
	if err := scanTaskRow(row, &task); err != nil {
		t.Fatalf("scanTaskRow on a task with no handoff: %v", err)
	}
	if task.ID != id || task.Title != "Ship the form layer" {
		t.Fatalf("scanned the wrong task: %+v", task)
	}
	if task.Status != models.StatusTaskInProgress {
		t.Errorf("status = %q, want in_progress", task.Status)
	}
	if task.HandoffContext != "" {
		t.Errorf("handoff_context = %q, want empty for a task that has none", task.HandoffContext)
	}
	if task.PredecessorTaskID != nil {
		t.Errorf("predecessor_task_id = %v, want nil", *task.PredecessorTaskID)
	}
}

// A NULL status is reachable (the column is nullable) and a CHECK evaluates to
// NULL and therefore passes, so the scanner must report the column's own
// default rather than leaving the zero value.
func TestScanTaskRowFallsBackToPendingForANullStatus(t *testing.T) {
	row := fakeRow{values: []driver.Value{
		uuid.New(), uuid.New(), "Title", nil, nil, nil,
		uuid.Nil, nil, nil, nil, nil, time.Now(), time.Now(), nil, nil,
	}}
	var task models.Task
	if err := scanTaskRow(row, &task); err != nil {
		t.Fatalf("scanTaskRow: %v", err)
	}
	if task.Status != models.StatusTaskPending {
		t.Errorf("status = %q, want pending", task.Status)
	}
}

func TestScanTaskRowKeepsAHandoffContext(t *testing.T) {
	row := fakeRow{values: []driver.Value{
		uuid.New(), uuid.New(), "Title", nil, "ready", nil,
		uuid.Nil, nil, nil, nil, nil, time.Now(), time.Now(), nil, "API is live at /users",
	}}
	var task models.Task
	if err := scanTaskRow(row, &task); err != nil {
		t.Fatalf("scanTaskRow: %v", err)
	}
	if task.HandoffContext != "API is live at /users" {
		t.Errorf("handoff_context = %q", task.HandoffContext)
	}
}
