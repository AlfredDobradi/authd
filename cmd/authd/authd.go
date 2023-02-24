package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	jsonAuth "github.com/alfreddobradi/authd/backend/json"
	"github.com/alfreddobradi/authd/config"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)

	jsonAuth.SetFromConfig(cfg.JSON)
	jsonAuth, err := jsonAuth.Build()
	if err != nil {
		panic(err)
	}

	r.Get("/auth", func(w http.ResponseWriter, r *http.Request) {
		user, err := jsonAuth.BasicAuth(r)
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
		w.Write([]byte("OK"))
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

	if err := http.ListenAndServe(bind, r); err != nil && err != http.ErrServerClosed {
		fmt.Printf("HTTP server error: %v\n", err)
	}
}
