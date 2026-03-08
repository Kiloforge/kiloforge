package cli

import (
	"context"
	"testing"

	"kiloforge/internal/adapter/config"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestExtractTraceID_ValidSpan(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { tp.Shutdown(context.Background()) })

	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	traceID := extractTraceID(ctx)
	if traceID == "" {
		t.Fatal("expected non-empty trace ID")
	}
	if len(traceID) != 32 {
		t.Errorf("expected 32-char hex trace ID, got %d chars: %q", len(traceID), traceID)
	}
}

func TestExtractTraceID_NoSpan(t *testing.T) {
	traceID := extractTraceID(context.Background())
	if traceID != "" {
		t.Errorf("expected empty trace ID for context without span, got %q", traceID)
	}
}

func TestInitTracing_Disabled(t *testing.T) {
	f := false
	cfg := &config.Config{TracingEnabled: &f} // tracing explicitly disabled
	tracer, shutdown := initTracing(context.Background(), cfg)
	if tracer == nil {
		t.Fatal("expected non-nil tracer even when disabled")
	}
	if shutdown != nil {
		t.Error("expected nil shutdown when tracing is disabled")
	}
}
