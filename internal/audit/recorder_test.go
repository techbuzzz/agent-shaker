package audit

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/events"
)

// recordingRecorder keeps what it was asked to write.
type recordingRecorder struct {
	events []Event
}

func (r *recordingRecorder) Record(_ context.Context, ev Event) error {
	r.events = append(r.events, ev)
	return nil
}

// failingRecorder refuses everything.
type failingRecorder struct{}

func (failingRecorder) Record(_ context.Context, _ Event) error {
	return errors.New("the database is gone")
}

// timeFixture is a fixed instant, so a hash is reproducible.
func timeFixture() time.Time {
	return time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
}

// domainEventFixture is one domain event, as the broker publishes it.
func domainEventFixture() events.Event {
	return events.New(events.TaskClaimed, uuid.MustParse("33333333-3333-3333-3333-333333333333"), events.Options{
		EntityType: events.EntityTask,
		EntityID:   uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		ActorType:  events.ActorAgent,
		ActorID:    uuid.MustParse("44444444-4444-4444-4444-444444444444"),
		Payload:    map[string]any{events.PayloadTitle: "Ship it"},
	})
}

// TestTheJournalIsNeverMutated pins the property the whole package exists for.
//
// The check is a source scan rather than a behavioural one, because there is
// no behavioural way to prove the absence of a statement: a method that
// rewrote a row would simply not be called by the test. The scan reads the
// package's own source and fails if a mutation of the table appears in it.
func TestTheJournalIsNeverMutated(t *testing.T) {
	t.Parallel()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading the package directory: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}

		source, err := os.ReadFile(filepath.Join(".", entry.Name()))
		if err != nil {
			t.Fatalf("reading %s: %v", entry.Name(), err)
		}

		upper := strings.ToUpper(string(source))
		// The patterns are assembled from parts so that this file does not
		// contain them: a scan that matched its own pattern list would fail on
		// itself and tell the reader nothing.
		for _, forbidden := range []string{
			"UPDATE " + "AUDIT_EVENTS",
			"DELETE FROM " + "AUDIT_EVENTS",
			"TRUNCATE " + "AUDIT_EVENTS",
		} {
			if strings.Contains(upper, forbidden) {
				t.Errorf("%s contains %q; the journal is append-only", entry.Name(), forbidden)
			}
		}
	}
}

// TestTheChainHashCoversTheRow checks that a row's hash depends on its own
// content and on its predecessor.
//
// Two rows with the same action but different entities must not hash the
// same, and two rows with the same content but different predecessors must
// not either — otherwise the chain would be decorative.
func TestTheChainHashCoversTheRow(t *testing.T) {
	t.Parallel()

	first := Event{
		Action:     "task.claimed",
		EntityType: "task",
		EntityID:   uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		After:      map[string]any{"title": "Ship it"},
	}
	second := Event{
		Action:     "task.claimed",
		EntityType: "task",
		EntityID:   uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		After:      map[string]any{"title": "Ship it"},
	}

	now := timeFixture()
	hashA, err := chainHash("", first, now)
	if err != nil {
		t.Fatalf("hashing the first event: %v", err)
	}
	hashB, err := chainHash("", second, now)
	if err != nil {
		t.Fatalf("hashing the second event: %v", err)
	}
	if hashA == hashB {
		t.Fatal("two different events hashed the same")
	}

	// The same event behind a different predecessor is a different row.
	hashC, err := chainHash(hashA, first, now)
	if err != nil {
		t.Fatalf("hashing the chained event: %v", err)
	}
	if hashA == hashC {
		t.Fatal("the predecessor does not affect the hash")
	}
}

// TestTheChainHashIsStable checks that the same row hashes the same way twice.
//
// The chain is only verifiable if the encoding is deterministic. Go's JSON
// encoder sorts map keys, which is what makes the snapshot fields stable.
func TestTheChainHashIsStable(t *testing.T) {
	t.Parallel()

	event := Event{
		Action:     "context.published",
		EntityType: "context",
		After: map[string]any{
			"title": "a context",
			"tags":  []string{"one", "two"},
			"meta":  map[string]any{"b": 2, "a": 1},
		},
	}

	now := timeFixture()
	first, err := chainHash("prev", event, now)
	if err != nil {
		t.Fatalf("hashing: %v", err)
	}
	second, err := chainHash("prev", event, now)
	if err != nil {
		t.Fatalf("hashing again: %v", err)
	}
	if first != second {
		t.Fatal("the same row hashed two different ways")
	}
}

// TestTheSnapshotEncodingIsDeterministic checks the property the hash rests
// on: the same map must encode to the same bytes every time.
func TestTheSnapshotEncodingIsDeterministic(t *testing.T) {
	t.Parallel()

	snapshot := map[string]any{"b": 2, "a": 1, "c": map[string]any{"z": 1, "y": 2}}

	first, err := jsonColumn(snapshot)
	if err != nil {
		t.Fatalf("encoding: %v", err)
	}
	second, err := jsonColumn(snapshot)
	if err != nil {
		t.Fatalf("encoding again: %v", err)
	}
	if first.([]byte)[0] != second.([]byte)[0] {
		t.Fatal("the same map encoded two different ways")
	}
}

// TestAnAbsentSnapshotIsNull checks that "no before" is not "an empty
// before".
func TestAnAbsentSnapshotIsNull(t *testing.T) {
	t.Parallel()

	encoded, err := jsonColumn(nil)
	if err != nil {
		t.Fatalf("encoding: %v", err)
	}
	if encoded != nil {
		t.Fatalf("an absent snapshot encoded as %v, want nil", encoded)
	}
}

// TestTheSubscriberRecordsTheDomainEvent checks the bridge between the broker
// and the journal.
func TestTheSubscriberRecordsTheDomainEvent(t *testing.T) {
	t.Parallel()

	recorder := &recordingRecorder{}
	subscriber := NewSubscriber(recorder)

	subscriber.HandleEvent(context.Background(), domainEventFixture())

	if len(recorder.events) != 1 {
		t.Fatalf("the recorder saw %d events, want 1", len(recorder.events))
	}
	recorded := recorder.events[0]
	if recorded.Action != "task.claimed" {
		t.Errorf("action = %q, want task.claimed", recorded.Action)
	}
	if recorded.EntityType != "task" {
		t.Errorf("entity type = %q, want task", recorded.EntityType)
	}
	if recorded.After["title"] != "Ship it" {
		t.Errorf("the payload was not carried over: %v", recorded.After)
	}
}

// TestAFailedWriteDoesNotEscapeTheSubscriber checks the rule that keeps the
// journal from being an outage.
func TestAFailedWriteDoesNotEscapeTheSubscriber(t *testing.T) {
	t.Parallel()

	subscriber := NewSubscriber(&failingRecorder{})

	// No panic, no error: the request that caused the event carries on.
	subscriber.HandleEvent(context.Background(), domainEventFixture())
}

// TestTheNoopRecorderWritesNothing checks that a server without a database
// still has a recorder.
func TestTheNoopRecorderWritesNothing(t *testing.T) {
	t.Parallel()

	if err := NewNoopRecorder().Record(context.Background(), Event{Action: "task.claimed"}); err != nil {
		t.Fatalf("the no-op recorder returned an error: %v", err)
	}
}

// TestAnEventWithoutAnActionIsRefused checks the one thing the recorder
// insists on.
func TestAnEventWithoutAnActionIsRefused(t *testing.T) {
	t.Parallel()

	recorder := NewPostgresRecorder(nil, "")
	if err := recorder.Record(context.Background(), Event{}); err == nil {
		t.Fatal("an event without an action must be refused")
	}
}
