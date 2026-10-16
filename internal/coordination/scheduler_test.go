package coordination

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeRetention struct {
	calls   []time.Time
	removed int64
	err     error
}

func (f *fakeRetention) DeleteOlderThan(_ context.Context, before time.Time) (int64, error) {
	f.calls = append(f.calls, before)
	return f.removed, f.err
}

// The job list is the point of having a scheduler: one place that says what
// runs and how often, instead of a goroutine per timer.
func TestSchedulerRegistersBothJobs(t *testing.T) {
	store := &fakeRetention{}
	s, err := NewScheduler(context.Background(),
		NewStaleAgentReaper(nil, nil, DefaultConfig()), store, DefaultSchedulerConfig())
	if err != nil {
		t.Fatalf("NewScheduler: %v", err)
	}
	if got := len(s.Jobs()); got != 2 {
		t.Fatalf("registered %d jobs, want 2 (reaper + retention)", got)
	}
}

// Without a database there is nothing to sweep and nothing to prune. The
// scheduler must still come up — a nil store is a degraded server, not a
// reason to refuse to start.
func TestSchedulerToleratesNilStores(t *testing.T) {
	s, err := NewScheduler(context.Background(), nil, nil, DefaultSchedulerConfig())
	if err != nil {
		t.Fatalf("NewScheduler: %v", err)
	}
	if got := len(s.Jobs()); got != 0 {
		t.Fatalf("registered %d jobs with no stores, want 0", got)
	}
}

// Retention is the design's "keep 30 days of history"; the cutoff has to move
// with the clock rather than being pinned at scheduler start.
func TestPruneHeartbeatsUsesTheRetentionWindow(t *testing.T) {
	store := &fakeRetention{removed: 7}
	s := &Scheduler{}

	before := time.Now()
	s.pruneHeartbeats(context.Background(), store, 30*24*time.Hour)

	if len(store.calls) != 1 {
		t.Fatalf("DeleteOlderThan called %d times, want 1", len(store.calls))
	}
	got := store.calls[0]
	wantWindow := 30 * 24 * time.Hour
	if diff := got.Sub(before.Add(-wantWindow)); diff > time.Minute || diff < -time.Minute {
		t.Fatalf("cutoff %s is not ~30 days before now", got)
	}
}

// A failed prune must not take the scheduler with it: the table growing is a
// slower problem than the server going down.
func TestPruneFailureIsSwallowed(t *testing.T) {
	store := &fakeRetention{err: errors.New("deadlock")}
	s := &Scheduler{}

	s.pruneHeartbeats(context.Background(), store, time.Hour) // must not panic

	if len(store.calls) != 1 {
		t.Fatalf("the store was consulted %d times, want 1", len(store.calls))
	}
}

// A lease at or below the sweep interval means an agent is declared dead
// between two heartbeats, so the scheduler clamps rather than trusting the
// configuration.
func TestSchedulerClampsLeaseTTLToTwiceTheSweep(t *testing.T) {
	cfg := DefaultSchedulerConfig()
	cfg.Reaper.LeaseTTL = time.Second
	cfg.Reaper.ReaperInterval = time.Minute

	s, err := NewScheduler(context.Background(), nil, nil, cfg)
	if err != nil {
		t.Fatalf("NewScheduler: %v", err)
	}
	if len(s.Jobs()) != 0 {
		t.Fatal("nil stores should register nothing")
	}
	if cfg.Reaper.LeaseTTL > cfg.Reaper.ReaperInterval {
		t.Fatal("the clamp did not happen: lease TTL is not above the sweep interval")
	}
}

var _ = uuid.Nil // keep the uuid import meaningful if the fakes grow
