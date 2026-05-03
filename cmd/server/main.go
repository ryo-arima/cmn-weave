// Package main starts the cmn-weave REST server.
package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"

	"github.com/ryo-arima/cmn-weave/pkg/config"
	"github.com/ryo-arima/cmn-weave/pkg/server"
)

func main() {
	configPath := flag.String("config", "etc/server.yaml", "path to server config file")
	flag.Parse()

	var cfg config.YamlConfig
	if err := config.Load(*configPath, &cfg); err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := server.Run(ctx, cfg); err != nil {
		log.Fatalf("server exited with error: %v", err)
	}
}
