package tracing

import (
	"context"
	"errors"
	"testing"

	"kiloforge/internal/core/port"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func setupTestTracer(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { tp.Shutdown(context.Background()) })
	return exp
}

func TestOTelTracer_StartSpan(t *testing.T) {
	exp := setupTestTracer(t)
	tracer := NewOTelTracer()

	ctx, span := tracer.StartSpan(context.Background(), "test-op",
		port.StringAttr("conductor.track.id", "track-123"),
		port.IntAttr("conductor.phase", 1),
	)
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}
	span.End()

	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].Name != "test-op" {
		t.Errorf("expected span name 'test-op', got %q", spans[0].Name)
	}
}

func TestOTelTracer_SetAttributes(t *testing.T) {
	exp := setupTestTracer(t)
	tracer := NewOTelTracer()

	_, span := tracer.StartSpan(context.Background(), "task/1.1")
	span.SetAttributes(
		port.IntAttr("tokens.input", 5000),
		port.IntAttr("tokens.output", 1000),
		port.Float64Attr("cost.usd", 0.15),
	)
	span.End()

	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	attrs := spans[0].Attributes
	found := map[string]bool{}
	for _, a := range attrs {
		found[string(a.Key)] = true
	}
	for _, key := range []string{"tokens.input", "tokens.output", "cost.usd"} {
		if !found[key] {
			t.Errorf("expected attribute %q", key)
		}
	}
}

func TestOTelTracer_AddEvent(t *testing.T) {
	exp := setupTestTracer(t)
	tracer := NewOTelTracer()

	_, span := tracer.StartSpan(context.Background(), "agent/invoke")
	span.AddEvent("agent.spawned", port.IntAttr("pid", 12345))
	span.AddEvent("agent.completed")
	span.End()

	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	events := spans[0].Events
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Name != "agent.spawned" {
		t.Errorf("expected event 'agent.spawned', got %q", events[0].Name)
	}
}

func TestOTelTracer_SetError(t *testing.T) {
	exp := setupTestTracer(t)
	tracer := NewOTelTracer()

	_, span := tracer.StartSpan(context.Background(), "failing-op")
	span.SetError(errors.New("spawn failed"))
	span.End()

	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].Status.Code.String() != "Error" {
		t.Errorf("expected Error status, got %s", spans[0].Status.Code)
	}
}

func TestOTelTracer_ParentChild(t *testing.T) {
	exp := setupTestTracer(t)
	tracer := NewOTelTracer()

	ctx, parent := tracer.StartSpan(context.Background(), "track/abc",
		port.StringAttr("conductor.track.id", "abc"),
	)
	_, child := tracer.StartSpan(ctx, "phase/1",
		port.IntAttr("conductor.phase", 1),
	)
	child.End()
	parent.End()

	spans := exp.GetSpans()
	if len(spans) != 2 {
		t.Fatalf("expected 2 spans, got %d", len(spans))
	}

	// Both should share the same trace ID.
	if spans[0].SpanContext.TraceID() != spans[1].SpanContext.TraceID() {
		t.Error("expected parent and child to share trace ID")
	}
}

func TestOTelTracer_StartSpanWithTraceID(t *testing.T) {
	exp := setupTestTracer(t)
	tracer := NewOTelTracer()

	// Create a span in a known trace.
	_, rootSpan := tracer.StartSpan(context.Background(), "track/root")
	rootSpan.End()

	rootSpans := exp.GetSpans()
	traceID := rootSpans[0].SpanContext.TraceID().String()

	// Start a new span joining that trace.
	_, childSpan := tracer.StartSpanWithTraceID(context.Background(), traceID, "webhook/pr",
		port.StringAttr("pr.number", "42"),
	)
	childSpan.End()

	allSpans := exp.GetSpans()
	if len(allSpans) != 2 {
		t.Fatalf("expected 2 spans, got %d", len(allSpans))
	}

	// The child span should share the same trace ID as the root.
	if allSpans[1].SpanContext.TraceID().String() != traceID {
		t.Errorf("child trace ID %q != root trace ID %q",
			allSpans[1].SpanContext.TraceID(), traceID)
	}
}

func TestOTelTracer_StartSpanWithTraceID_InvalidFallback(t *testing.T) {
	exp := setupTestTracer(t)
	tracer := NewOTelTracer()

	// Invalid trace ID should fall back to creating a new trace.
	_, span := tracer.StartSpanWithTraceID(context.Background(), "not-a-hex-id", "fallback-span")
	span.End()

	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].Name != "fallback-span" {
		t.Errorf("expected span name 'fallback-span', got %q", spans[0].Name)
	}
}

func TestOTelTracer_ImplementsInterface(t *testing.T) {
	var _ port.Tracer = (*OTelTracer)(nil)
}
