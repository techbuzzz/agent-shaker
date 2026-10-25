package events

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type recordingSubscriber struct {
	received []Event
}

func (s *recordingSubscriber) HandleEvent(_ context.Context, ev Event) {
	s.received = append(s.received, ev)
}

type panickingSubscriber struct{}

func (panickingSubscriber) HandleEvent(_ context.Context, _ Event) {
	panic("subscriber blew up")
}

func TestPublishReachesEverySubscriber(t *testing.T) {
	broker := NewBroker()
	first := &recordingSubscriber{}
	second := &recordingSubscriber{}
	broker.Subscribe(first)
	broker.Subscribe(second)

	projectID := uuid.New()
	broker.Publish(context.Background(), New(TaskCompleted, projectID, Options{
		EntityType: EntityTask,
		EntityID:   uuid.New(),
		Payload:    map[string]any{PayloadTitle: "a task"},
	}))

	if len(first.received) != 1 || len(second.received) != 1 {
		t.Fatalf("the subscribers received %d and %d events, want 1 each",
			len(first.received), len(second.received))
	}
	if first.received[0].Type != TaskCompleted {
		t.Fatalf("the event type was %q, want %q", first.received[0].Type, TaskCompleted)
	}
	if first.received[0].ProjectID != projectID {
		t.Fatal("the event must carry the project it happened in")
	}
}

func TestPublishContainsAPanickingSubscriber(t *testing.T) {
	broker := NewBroker()
	broker.Subscribe(panickingSubscriber{})
	survivor := &recordingSubscriber{}
	broker.Subscribe(survivor)

	// A subscriber that panics must not take down the request that produced
	// the event, and must not stop the other subscribers from being told.
	broker.Publish(context.Background(), New(TaskClaimed, uuid.New(), Options{}))

	if len(survivor.received) != 1 {
		t.Fatalf("the surviving subscriber received %d events, want 1", len(survivor.received))
	}
}

func TestNewCopiesThePayload(t *testing.T) {
	payload := map[string]any{PayloadTitle: "original"}
	event := New(TaskClaimed, uuid.New(), Options{Payload: payload})

	payload[PayloadTitle] = "changed"

	if event.Payload[PayloadTitle] != "original" {
		t.Fatal("the event must not see later changes to the caller's map")
	}
}

func TestNewAlwaysHasAPayloadMap(t *testing.T) {
	event := New(TaskClaimed, uuid.New(), Options{})

	if event.Payload == nil {
		t.Fatal("the payload must be an empty map, not nil")
	}
}

func TestIsKnownAcceptsTheVocabularyAndNothingElse(t *testing.T) {
	for _, eventType := range KnownTypes() {
		if !IsKnown(eventType) {
			t.Fatalf("%q must be a known event", eventType)
		}
	}
	if IsKnown("task.exploded") {
		t.Fatal("an unknown event must not be accepted")
	}
}
