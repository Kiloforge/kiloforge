package tracing

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const serviceName = "kiloforge"

// InitResult holds the output of Init.
type InitResult struct {
	Shutdown func(context.Context) error
	Store    SpanRecorder
}

// InitOption configures the OTel trace provider initialization.
type InitOption func(*initOpts)

type initOpts struct {
	recorder SpanRecorder
}

// WithSpanRecorder sets a custom SpanRecorder (e.g., sqlite.TraceStore) instead
// of the default in-memory Store.
func WithSpanRecorder(r SpanRecorder) InitOption {
	return func(o *initOpts) { o.recorder = r }
}

// Init initializes the OTel trace provider with an OTLP HTTP exporter
// and a span recorder for API queries.
// If endpoint is empty, defaults to localhost:4318 (Jaeger OTLP HTTP).
// By default, an in-memory Store is used as the recorder; use WithSpanRecorder
// to supply a persistent store (e.g., sqlite.TraceStore).
func Init(ctx context.Context, endpoint string, options ...InitOption) (*InitResult, error) {
	var iopts initOpts
	for _, o := range options {
		o(&iopts)
	}

	opts := []otlptracehttp.Option{
		otlptracehttp.WithInsecure(),
	}
	if endpoint != "" {
		opts = append(opts, otlptracehttp.WithEndpoint(endpoint))
	}

	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create OTLP exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			attribute.String("service.name", serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	recorder := iopts.recorder
	if recorder == nil {
		recorder = NewStore()
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter, sdktrace.WithBatchTimeout(5*time.Second)),
		sdktrace.WithSpanProcessor(NewStoreProcessor(recorder)),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	return &InitResult{
		Shutdown: tp.Shutdown,
		Store:    recorder,
	}, nil
}
