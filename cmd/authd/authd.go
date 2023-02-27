package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/alfreddobradi/authd/config"
	"github.com/alfreddobradi/authd/instrumentation"
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

	if cfg.Instrumentation.Enabled {
		if err := instrumentation.Provider(cfg.Instrumentation); err != nil {
			log.Fatal(err)
		}
		defer func() {
			if err := instrumentation.Shutdown(context.Background()); err != nil {
				log.Fatal("failed to shutdown TracerProvider: %w", err)
			}
		}()
	}

	s, err := buildServer(cfg)
	if err != nil {
		log.Fatalf("Failed to create HTTP server: %v", err)
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
