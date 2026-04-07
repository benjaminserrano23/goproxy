package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/benjaminserrano23/goproxy/config"
	"github.com/benjaminserrano23/goproxy/proxy"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	mux, err := proxy.BuildMux(cfg)
	if err != nil {
		log.Fatalf("failed to build routes: %v", err)
	}

	addr := ":" + cfg.Server.Port
	server := &http.Server{Addr: addr, Handler: mux}

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		fmt.Printf("goproxy listening on %s\n", addr)
		fmt.Printf("routes: %d configured\n", len(cfg.Routes))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	fmt.Println("\nshutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown error: %v", err)
	}
	fmt.Println("server stopped")
}
