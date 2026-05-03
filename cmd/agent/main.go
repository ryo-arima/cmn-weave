// Package main starts the cmn-weave agent.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ryo-arima/cmn-weave/pkg/agent"
	"github.com/ryo-arima/cmn-weave/pkg/config"
)

func main() {
	configPath := flag.String("config", "etc/agent.yaml", "path to agent config file")
	flag.Parse()

	var cfg config.YamlConfig
	if err := config.Load(*configPath, &cfg); err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := agent.Main(ctx, cfg); err != nil {
		log.Fatalf("agent: %v", err)
	}
}
