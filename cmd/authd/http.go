package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	jsonAuth "github.com/alfreddobradi/authd/backend/json"
	"github.com/alfreddobradi/authd/config"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel"
)

func buildServer(cfg *config.Config) (*http.Server, error) {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)

	jsonAuth.SetFromConfig(cfg.JSON)
	jsonAuth, err := jsonAuth.Build()
	if err != nil {
		return nil, err
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

	return &s, nil
}
