package tracing

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestInit_SetsGlobalProvider(t *testing.T) {
	ctx := context.Background()

	// Use a non-routable endpoint so the exporter doesn't actually send.
	result, err := Init(ctx, "192.0.2.1:4318")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer result.Shutdown(ctx)

	tp := otel.GetTracerProvider()
	if _, ok := tp.(*sdktrace.TracerProvider); !ok {
		t.Errorf("expected *sdktrace.TracerProvider, got %T", tp)
	}

	if result.Store == nil {
		t.Error("expected non-nil Store")
	}
}

func TestInit_ShutdownIsIdempotent(t *testing.T) {
	ctx := context.Background()
	result, err := Init(ctx, "192.0.2.1:4318")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Calling shutdown twice should not panic.
	if err := result.Shutdown(ctx); err != nil {
		t.Errorf("first shutdown failed: %v", err)
	}
	if err := result.Shutdown(ctx); err != nil {
		t.Errorf("second shutdown failed: %v", err)
	}
}
