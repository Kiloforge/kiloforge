package analytics

import "context"

// NoopTracker is a no-op analytics tracker used when analytics is disabled.
type NoopTracker struct{}

func (*NoopTracker) Track(_ context.Context, _ string, _ map[string]any) {}
func (*NoopTracker) Shutdown(_ context.Context) error                    { return nil }
