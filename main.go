package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/benjaminserrano23/goproxy/config"
	"github.com/benjaminserrano23/goproxy/proxy"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	mux := proxy.BuildMux(cfg)

	addr := ":" + cfg.Server.Port
	fmt.Printf("goproxy listening on %s\n", addr)
	fmt.Printf("routes: %d configured\n", len(cfg.Routes))
	log.Fatal(http.ListenAndServe(addr, mux))
}
