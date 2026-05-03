// Package agent wires the gRPC server DI chain and manages the service lifecycle.
//
// Dependency flow: base → controller → usecase → repository
package agent

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	agentpb "github.com/ryo-arima/cmn-weave/pkg/agent/grpc/auto"
	agentshare "github.com/ryo-arima/cmn-weave/pkg/agent/share"
	"github.com/ryo-arima/cmn-weave/pkg/agent/cplane/controller"
	"github.com/ryo-arima/cmn-weave/pkg/agent/cplane/repository"
	"github.com/ryo-arima/cmn-weave/pkg/agent/cplane/usecase"
	"github.com/ryo-arima/cmn-weave/pkg/config"
)

// Main wires the full DI chain and starts the agent gRPC server.
// It blocks until ctx is cancelled.
func Main(ctx context.Context, cfg config.YamlConfig) error {
	agentcfg := cfg.Application.Agent
	// --- TLS ---------------------------------------------------------
	tlsCfg, err := agentshare.ServerTLSConfig(agentcfg.TLS)
	if err != nil {
		return fmt.Errorf("build tls config: %w", err)
	}

	// --- Repository --------------------------------------------------
	execRepo := repository.NewOSExecRepository()

	// --- Usecase -----------------------------------------------------
	taskUC := usecase.New(execRepo)

	// --- Controller --------------------------------------------------
	ctrl := controller.New(taskUC)

	// --- gRPC listener -----------------------------------------------
	lis, err := net.Listen("tcp", agentcfg.ListenAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", agentcfg.ListenAddr, err)
	}
	defer lis.Close()
	slog.Info("agent listening", "addr", agentcfg.ListenAddr)

	// --- gRPC server -------------------------------------------------
	grpcSrv := grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsCfg)))
	agentpb.RegisterAgentServiceServer(grpcSrv, ctrl)

	go grpcSrv.Serve(lis) //nolint:errcheck
	<-ctx.Done()
	grpcSrv.GracefulStop()
	return nil
}
