package port

import "context"

// Tracer abstracts distributed tracing for the conductor workflow.
type Tracer interface {
	// StartSpan starts a new span with the given name and returns the updated
	// context. The returned SpanEnder must be called to end the span.
	StartSpan(ctx context.Context, name string, attrs ...SpanAttr) (context.Context, SpanEnder)

	// StartSpanWithTraceID starts a new span that joins an existing trace
	// identified by the hex-encoded traceID. Used to continue a trace
	// across process boundaries (e.g., CLI → server via stored trace ID).
	StartSpanWithTraceID(ctx context.Context, traceID, name string, attrs ...SpanAttr) (context.Context, SpanEnder)
}

// SpanEnder ends a span. SetAttributes and AddEvent can be called before End.
type SpanEnder interface {
	SetAttributes(attrs ...SpanAttr)
	AddEvent(name string, attrs ...SpanAttr)
	SetError(err error)
	End()
}

// SpanAttr is a key-value attribute on a span.
type SpanAttr struct {
	Key    string
	Value  any
}

// StringAttr creates a string span attribute.
func StringAttr(key, value string) SpanAttr {
	return SpanAttr{Key: key, Value: value}
}

// IntAttr creates an int span attribute.
func IntAttr(key string, value int) SpanAttr {
	return SpanAttr{Key: key, Value: value}
}

// Float64Attr creates a float64 span attribute.
func Float64Attr(key string, value float64) SpanAttr {
	return SpanAttr{Key: key, Value: value}
}

// NoopTracer is a tracer that does nothing. Used when tracing is disabled.
type NoopTracer struct{}

func (NoopTracer) StartSpan(ctx context.Context, _ string, _ ...SpanAttr) (context.Context, SpanEnder) {
	return ctx, noopSpan{}
}

func (NoopTracer) StartSpanWithTraceID(ctx context.Context, _, _ string, _ ...SpanAttr) (context.Context, SpanEnder) {
	return ctx, noopSpan{}
}

type noopSpan struct{}

func (noopSpan) SetAttributes(_ ...SpanAttr) {}
func (noopSpan) AddEvent(_ string, _ ...SpanAttr) {}
func (noopSpan) SetError(_ error) {}
func (noopSpan) End() {}
