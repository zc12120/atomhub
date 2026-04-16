package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/zc12120/atomhub/internal/app"
	"github.com/zc12120/atomhub/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("bootstrap app: %v", err)
	}
	defer func() {
		if closeErr := application.Close(); closeErr != nil {
			log.Printf("close app: %v", closeErr)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Printf("AtomHub listening on %s", cfg.HTTPAddr)
	if err := application.Run(ctx); err != nil {
		log.Fatalf("run app: %v", err)
	}
}
