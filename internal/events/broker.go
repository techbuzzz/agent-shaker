package events

import (
	"context"
	"log/slog"
	"sync"
)

// Subscriber receives domain events.
//
// Implementations must return quickly. The broker calls subscribers in the
// publisher's goroutine, so a subscriber that computes turns every state
// change in the product into a slow state change.
type Subscriber interface {
	HandleEvent(ctx context.Context, ev Event)
}

// SubscriberFunc adapts a function to the Subscriber interface.
type SubscriberFunc func(ctx context.Context, ev Event)

// HandleEvent implements Subscriber.
func (f SubscriberFunc) HandleEvent(ctx context.Context, ev Event) {
	f(ctx, ev)
}

// Broker fans domain events out to its subscribers.
//
// It exists so that one change has one publication point. Before it, the
// WebSocket hub was the only way to learn that something happened, which
// meant the audit trail and the hook layer had no way to learn either — they
// would have to parse the same broadcast payloads the browser parses.
type Broker struct {
	mu   sync.RWMutex
	subs []Subscriber
}

// NewBroker returns an empty broker.
func NewBroker() *Broker {
	return &Broker{subs: make([]Subscriber, 0, 4)}
}

// Subscribe registers a subscriber.
//
// Subscribers are called in registration order. There is no unsubscribe:
// a subscriber lives as long as the process does, and the components that
// register one are process-scoped too.
func (b *Broker) Subscribe(s Subscriber) {
	if s == nil {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs = append(b.subs, s)
}

// Publish delivers the event to every subscriber.
//
// A panicking subscriber is contained rather than allowed to take down the
// request that produced the event: subscribers are written independently, and
// one of them failing is not a reason for a task status change to be lost.
func (b *Broker) Publish(ctx context.Context, ev Event) {
	b.mu.RLock()
	subs := b.subs
	b.mu.RUnlock()

	for _, s := range subs {
		deliver(ctx, s, ev)
	}
}

// Send publishes an event when there is a broker to publish it to.
//
// A nil broker is a server running without the hook layer. Every publisher
// would otherwise have to check for one, and a check that is easy to forget
// is a check that will be forgotten.
func Send(ctx context.Context, b *Broker, ev Event) {
	if b == nil {
		return
	}
	b.Publish(ctx, ev)
}

func deliver(ctx context.Context, s Subscriber, ev Event) {
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "event subscriber panicked",
				"event_type", ev.Type,
				"panic", r,
			)
		}
	}()

	s.HandleEvent(ctx, ev)
}
