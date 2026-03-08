package tracing

import (
	"context"
	"errors"
	"testing"

	"kiloforge/internal/core/port"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestIntegration_FullPipeline(t *testing.T) {
	// Set up store + processor + tracer provider.
	store := NewStore()
	proc := NewStoreProcessor(store)
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(proc))
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())

	tracer := NewOTelTracer()

	// Simulate a track implementation lifecycle.
	ctx, trackSpan := tracer.StartSpan(context.Background(), "track/impl-auth_20250115",
		port.StringAttr("conductor.track.id", "impl-auth_20250115"),
		port.StringAttr("conductor.track.type", "feature"),
	)

	// Phase 1
	ctx2, phaseSpan := tracer.StartSpan(ctx, "phase/1",
		port.IntAttr("conductor.phase", 1),
	)

	// Task 1.1
	_, taskSpan := tracer.StartSpan(ctx2, "task/1.1",
		port.StringAttr("conductor.task", "1.1"),
	)
	taskSpan.AddEvent("agent.spawned", port.IntAttr("pid", 12345))
	taskSpan.SetAttributes(
		port.IntAttr("tokens.input", 45000),
		port.IntAttr("tokens.output", 8000),
		port.Float64Attr("cost.usd", 0.12),
	)
	taskSpan.AddEvent("agent.completed")
	taskSpan.End()

	// Task 1.2 (fails)
	_, task2Span := tracer.StartSpan(ctx2, "task/1.2",
		port.StringAttr("conductor.task", "1.2"),
	)
	task2Span.SetError(errors.New("test compilation failed"))
	task2Span.AddEvent("agent.failed")
	task2Span.End()

	phaseSpan.End()
	trackSpan.End()

	// Verify via store.
	traces := store.ListTraces()
	if len(traces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(traces))
	}
	if traces[0].SpanCount != 4 {
		t.Errorf("expected 4 spans (track + phase + 2 tasks), got %d", traces[0].SpanCount)
	}
	if traces[0].RootName != "track/impl-auth_20250115" {
		t.Errorf("expected root name, got %q", traces[0].RootName)
	}

	spans := store.GetTrace(traces[0].TraceID)
	if len(spans) != 4 {
		t.Fatalf("expected 4 spans, got %d", len(spans))
	}

	// Verify task 1.1 has attributes.
	var task1 *SpanSummary
	for _, s := range spans {
		if s.Name == "task/1.1" {
			task1 = &s
			break
		}
	}
	if task1 == nil {
		t.Fatal("task/1.1 span not found")
	}
	if task1.Attributes["tokens.input"] != "45000" {
		t.Errorf("expected tokens.input=45000, got %q", task1.Attributes["tokens.input"])
	}
	if len(task1.Events) != 2 {
		t.Errorf("expected 2 events on task/1.1, got %d", len(task1.Events))
	}

	// Verify task 1.2 has error status.
	var task2 *SpanSummary
	for _, s := range spans {
		if s.Name == "task/1.2" {
			task2 = &s
			break
		}
	}
	if task2 == nil {
		t.Fatal("task/1.2 span not found")
	}
	if task2.Status != "error" {
		t.Errorf("expected error status on task/1.2, got %q", task2.Status)
	}
}

func TestIntegration_NoopTracer(t *testing.T) {
	tracer := port.NoopTracer{}

	// Should not panic or error.
	ctx, span := tracer.StartSpan(context.Background(), "track/test")
	span.SetAttributes(port.StringAttr("key", "value"))
	span.AddEvent("test-event")
	span.SetError(errors.New("test error"))

	_, child := tracer.StartSpan(ctx, "child")
	child.End()
	span.End()
}
