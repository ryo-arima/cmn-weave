// Package usecase holds the agent-side task execution business logic.
package usecase

import (
	"context"

	"github.com/ryo-arima/cmn-weave/pkg/agent/cplane/repository"
)

// CommonUsecase orchestrates task execution via the ExecutionRepository.
type CommonUsecase struct {
	exec repository.ExecutionRepository
}

// New creates a CommonUsecase backed by the given ExecutionRepository.
func New(exec repository.ExecutionRepository) *CommonUsecase {
	return &CommonUsecase{exec: exec}
}

// Execute runs a task identified by taskID with the given argv and env.
// It delegates to the underlying ExecutionRepository and discards stdout/stderr
// (callers that need output should use StreamLogs).
func (rcvr *CommonUsecase) Execute(ctx context.Context, taskID string, argv []string, env []string) error {
	_, _, err := rcvr.exec.Run(ctx, argv, env)
	return err
}
