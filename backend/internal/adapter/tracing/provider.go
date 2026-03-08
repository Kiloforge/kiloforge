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
	Store    *Store
}

// Init initializes the OTel trace provider with an OTLP HTTP exporter
// and an in-memory store for API queries.
// If endpoint is empty, defaults to localhost:4318 (Jaeger OTLP HTTP).
func Init(ctx context.Context, endpoint string) (*InitResult, error) {
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

	store := NewStore()

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter, sdktrace.WithBatchTimeout(5*time.Second)),
		sdktrace.WithSpanProcessor(NewStoreProcessor(store)),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	return &InitResult{
		Shutdown: tp.Shutdown,
		Store:    store,
	}, nil
}
