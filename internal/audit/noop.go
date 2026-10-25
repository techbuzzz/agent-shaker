package audit

import (
	"context"
)

// NoopRecorder writes nothing.
//
// It exists so that a server without a database still has a recorder, and so
// that tests do not need one. The journal is a record of what happened; a
// server that cannot record still has to work.
type NoopRecorder struct{}

// NewNoopRecorder returns a recorder that discards everything.
func NewNoopRecorder() *NoopRecorder { return &NoopRecorder{} }

// Record implements Recorder and does nothing.
func (NoopRecorder) Record(_ context.Context, _ Event) error { return nil }
