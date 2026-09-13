package observability

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync/atomic"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// TracingShutdown flushes and stops the global tracer provider. Always
// call it during server shutdown so in-flight spans are exported.
type TracingShutdown func(context.Context) error

// tracingEnabled is set to true when InitTracing installs a real provider.
// Read via TracerEnabled from goroutines that want to skip span creation
// when tracing is off (no-op providers still work; this is purely an opt).
var tracingEnabled atomic.Bool

// NoopTracingShutdown is returned when tracing is not configured. Calling
// it is a safe no-op.
func NoopTracingShutdown(context.Context) error { return nil }

// TracerEnabled returns true when InitTracing wired a real provider.
func TracerEnabled() bool { return tracingEnabled.Load() }

// InitTracing constructs the global TracerProvider and propagator. When
// OTEL_EXPORTER_OTLP_ENDPOINT is empty or unset, the global provider stays
// at its no-op default and the returned shutdown is a no-op — the cold-start
// cost is zero.
//
// Sampling defaults to ParentBased(TraceIDRatioBased(0.05)) when tracing is
// enabled, matching the plan's recommendation. Operators override via the
// standard OTEL_TRACES_SAMPLER env var (the OTel SDK reads it directly).
func InitTracing(ctx context.Context, serviceName, version string) (TracingShutdown, error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		slog.Info("tracing disabled (OTEL_EXPORTER_OTLP_ENDPOINT not set)")
		return NoopTracingShutdown, nil
	}

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("create OTLP exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(version),
		),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		return nil, fmt.Errorf("create OTel resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(0.05))),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	tracingEnabled.Store(true)

	slog.Info("tracing enabled",
		"endpoint", endpoint,
		"service", serviceName,
		"version", version,
	)

	return func(shutdownCtx context.Context) error {
		tracingEnabled.Store(false)
		if err := tp.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("tracer provider shutdown: %w", err)
		}
		return nil
	}, nil
}
