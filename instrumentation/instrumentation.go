package instrumentation

import (
	"context"
	"fmt"
	"time"

	"github.com/alfreddobradi/authd/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var provider *trace.TracerProvider

func Provider(cfg config.InstrumentationConfig) error {
	if provider == nil {
		if cfg.Endpoint == "" {
			return fmt.Errorf("The endpoint can't be empty if instrumentation is enabled")
		}
		ctx := context.Background()

		res, err := resource.New(ctx,
			resource.WithAttributes(
				// the service name used to display traces in backends
				semconv.ServiceName("test-service"),
			),
		)
		if err != nil {
			return fmt.Errorf("failed to create resource: %w", err)
		}

		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		conn, err := grpc.DialContext(ctx, cfg.Endpoint,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)
		if err != nil {
			return fmt.Errorf("failed to create gRPC connection to collector: %w", err)
		}

		// Set up a trace exporter
		traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
		if err != nil {
			return fmt.Errorf("failed to create trace exporter: %w", err)
		}

		bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
		tracerProvider := sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
			sdktrace.WithResource(res),
			sdktrace.WithSpanProcessor(bsp),
		)
		otel.SetTracerProvider(tracerProvider)

		otel.SetTextMapPropagator(propagation.TraceContext{})

		provider = tracerProvider
	}

	return nil
}

func Shutdown(ctx context.Context) error {
	if provider != nil {
		return provider.Shutdown(ctx)
	}
	return nil
}
