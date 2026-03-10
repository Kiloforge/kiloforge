package analytics

import "context"

// Noop is an analytics tracker that does nothing.
type Noop struct{}

func (n *Noop) Track(_ context.Context, _ string, _ map[string]any) {}

func (n *Noop) Shutdown(_ context.Context) error { return nil }
