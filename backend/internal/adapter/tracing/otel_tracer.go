package tracing

import (
	"context"
	"fmt"

	"kiloforge/internal/core/port"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "kiloforge/conductor"

// OTelTracer implements port.Tracer using the OpenTelemetry SDK.
type OTelTracer struct {
	tracer trace.Tracer
}

// NewOTelTracer creates a new OTel-backed tracer.
func NewOTelTracer() *OTelTracer {
	return &OTelTracer{
		tracer: otel.Tracer(tracerName),
	}
}

func (t *OTelTracer) StartSpan(ctx context.Context, name string, attrs ...port.SpanAttr) (context.Context, port.SpanEnder) {
	otelAttrs := toOTelAttrs(attrs)
	ctx, span := t.tracer.Start(ctx, name, trace.WithAttributes(otelAttrs...))
	return ctx, &otelSpan{span: span}
}

type otelSpan struct {
	span trace.Span
}

func (s *otelSpan) SetAttributes(attrs ...port.SpanAttr) {
	s.span.SetAttributes(toOTelAttrs(attrs)...)
}

func (s *otelSpan) AddEvent(name string, attrs ...port.SpanAttr) {
	s.span.AddEvent(name, trace.WithAttributes(toOTelAttrs(attrs)...))
}

func (s *otelSpan) SetError(err error) {
	if err != nil {
		s.span.RecordError(err)
		s.span.SetStatus(codes.Error, err.Error())
	}
}

func (s *otelSpan) End() {
	s.span.End()
}

func toOTelAttrs(attrs []port.SpanAttr) []attribute.KeyValue {
	result := make([]attribute.KeyValue, 0, len(attrs))
	for _, a := range attrs {
		switch v := a.Value.(type) {
		case string:
			result = append(result, attribute.String(a.Key, v))
		case int:
			result = append(result, attribute.Int(a.Key, v))
		case float64:
			result = append(result, attribute.Float64(a.Key, v))
		case bool:
			result = append(result, attribute.Bool(a.Key, v))
		default:
			result = append(result, attribute.String(a.Key, fmt.Sprintf("%v", v)))
		}
	}
	return result
}
