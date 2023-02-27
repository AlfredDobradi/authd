package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	jsonAuth "github.com/alfreddobradi/authd/backend/json"
	"github.com/alfreddobradi/authd/config"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const Realm string = "authd realm"

type OkResponse struct {
	User          string `json:"user"`
	Authenticated bool   `json:"authenticated"`
}

func main() {
	cfgPath := os.Getenv("AUTHD_CFG_PATH")
	if cfgPath == "" {
		cfgPath = "./config.yaml"
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		panic(err)
	}

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt)

	shutdown, err := initProvider()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := shutdown(context.Background()); err != nil {
			log.Fatal("failed to shutdown TracerProvider: %w", err)
		}
	}()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)

	jsonAuth.SetFromConfig(cfg.JSON)
	jsonAuth, err := jsonAuth.Build()
	if err != nil {
		panic(err)
	}

	r.Get("/auth", func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer("handler").Start(r.Context(), r.URL.Path)
		defer span.End()
		user, err := jsonAuth.BasicAuth(ctx, r)
		if err != nil {
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, Realm))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		response := OkResponse{
			User:          user,
			Authenticated: true,
		}
		encoder := json.NewEncoder(w)
		encoder.Encode(response) // nolint
	})

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_, span := otel.Tracer("handler").Start(r.Context(), r.URL.Path)
		defer span.End()
		w.Write([]byte("OK"))
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		_, span := otel.Tracer("handler").Start(r.Context(), r.URL.Path)
		span.RecordError(fmt.Errorf("404 Not Found"))
		defer span.End()
		w.WriteHeader(404)
		w.Write([]byte("custom 404 page"))
	})

	address := "127.0.0.1"
	if cfg.HTTP.Address != "" {
		address = cfg.HTTP.Address
	}

	port := "80"
	if cfg.HTTP.Port != "" {
		port = cfg.HTTP.Port
	}
	bind := fmt.Sprintf("%s:%s", address, port)
	fmt.Printf("Listening on %s\n", bind)

	s := http.Server{
		Addr:    bind,
		Handler: r,
	}

	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	<-sigchan
	if err := s.Shutdown(context.Background()); err != nil {
		log.Printf("Received error while shutting down http server: %s", err)
	}
}

// Initializes an OTLP exporter, and configures the corresponding trace and
// metric providers.
func initProvider() (func(context.Context) error, error) {
	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceName("test-service"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// If the OpenTelemetry Collector is running on a local cluster (minikube or
	// microk8s), it should be accessible through the NodePort service at the
	// `localhost:30080` endpoint. Otherwise, replace `localhost` with the
	// endpoint of your cluster. If you run the app inside k8s, then you can
	// probably connect directly to the service through dns.
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, "127.0.0.1:4317",
		// Note the use of insecure transport here. TLS is recommended in production.
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection to collector: %w", err)
	}

	// Set up a trace exporter
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Register the trace exporter with a TracerProvider, using a batch
	// span processor to aggregate spans before export.
	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tracerProvider)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Shutdown will flush any remaining spans and shut down the exporter.
	return tracerProvider.Shutdown, nil
}
