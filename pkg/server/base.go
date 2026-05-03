// Package server wires the REST server DI chain and manages the service lifecycle.
//
// Dependency flow: base → router → controller → usecase → repository (psql + grpc-agent)
package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/ryo-arima/cmn-weave/pkg/config"
	"github.com/ryo-arima/cmn-weave/pkg/global"
	"github.com/ryo-arima/cmn-weave/pkg/server/controller"
	"github.com/ryo-arima/cmn-weave/pkg/server/repository"
	"github.com/ryo-arima/cmn-weave/pkg/server/share"
	"github.com/ryo-arima/cmn-weave/pkg/server/usecase"
)

// Run wires the full DI chain and starts the REST server.
// It blocks until ctx is cancelled and returns the first non-shutdown error.
func Run(ctx context.Context, cfg config.YamlConfig) error {
	srvcfg := cfg.Application.Server
	// --- TLS ---------------------------------------------------------
	tlsCfg, err := share.ServerTLSConfig(srvcfg.TLS)
	if err != nil {
		return fmt.Errorf("build tls config: %w", err)
	}

	// --- Repository layer --------------------------------------------
	// PostgreSQL task store.
	// TODO: open *sql.DB with the pgx driver and pass it here:
	//   import _ "github.com/jackc/pgx/v5/stdlib"
	//   db, _ := sql.Open("pgx", cfg.PostgreSQL.DSN())
	taskRepo := repository.NewPSQLTaskRepository(nil) // nil = no-op stub
	nodeRepo := repository.NewPSQLNodeRepository(nil) // nil = no-op stub
	// gRPC agent dispatcher.
	agentClient := repository.NewGRPCAgentDispatcher(srvcfg.Agents, tlsCfg)

	// --- Usecase layer -----------------------------------------------
	uc := usecase.New(taskRepo, agentClient, nodeRepo)

	// --- Controller layer --------------------------------------------
	ctrl := controller.New(uc)

	// --- HTTP server -------------------------------------------------
	srv := &http.Server{
		Addr:              srvcfg.ListenAddr,
		Handler:           InitRouter(ctrl),
		TLSConfig:         tlsCfg,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		share.GetServerLogger().INFO("", global.SSM1, "server listening on "+srvcfg.ListenAddr)
		errCh <- srv.ListenAndServeTLS("", "")
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
