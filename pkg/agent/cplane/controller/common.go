// Package controller hosts the gRPC service handler for the agent.
//
// All handler methods and the Controller struct live here.
// Wire the Controller into a gRPC server via:
//
//	agentpb.RegisterAgentServiceServer(grpcSrv, ctrl)
package controller

import (
	"context"

	agentpb "github.com/ryo-arima/cmn-weave/pkg/agent/grpc/auto"
	"github.com/ryo-arima/cmn-weave/pkg/agent/cplane/usecase"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CommonController implements the AgentService gRPC interface.
// It delegates all business logic to CommonUsecase.
type CommonController struct {
	agentpb.UnimplementedAgentServiceServer
	task *usecase.CommonUsecase
}

// New creates a CommonController backed by the given CommonUsecase.
func New(task *usecase.CommonUsecase) *CommonController {
	return &CommonController{task: task}
}

// Dispatch enqueues a new task on the agent.
func (rcvr *CommonController) Dispatch(ctx context.Context, req *agentpb.DispatchRequest) (*agentpb.DispatchResponse, error) {
	if err := rcvr.task.Execute(ctx, req.TaskId, req.Argv, req.Env); err != nil {
		return nil, status.Errorf(codes.Internal, "execute: %v", err)
	}
	return &agentpb.DispatchResponse{
		TaskId: req.TaskId,
		State:  agentpb.TaskState_TASK_STATE_RUNNING,
	}, nil
}

// Status returns the current state of a task.
func (rcvr *CommonController) Status(_ context.Context, req *agentpb.StatusRequest) (*agentpb.StatusResponse, error) {
	// TODO: query actual task state from a local store.
	return &agentpb.StatusResponse{
		TaskId: req.TaskId,
		State:  agentpb.TaskState_TASK_STATE_RUNNING,
	}, nil
}

// Cancel requests cooperative termination of a running task.
func (rcvr *CommonController) Cancel(_ context.Context, req *agentpb.CancelRequest) (*agentpb.CancelResponse, error) {
	// TODO: signal the running process to terminate.
	return &agentpb.CancelResponse{
		TaskId: req.TaskId,
		State:  agentpb.TaskState_TASK_STATE_CANCELLED,
	}, nil
}

// StreamLogs streams stdout/stderr lines for a running task.
func (rcvr *CommonController) StreamLogs(_ *agentpb.StreamLogsRequest, _ grpc.ServerStreamingServer[agentpb.LogLine]) error {
	return status.Errorf(codes.Unimplemented, "StreamLogs not yet implemented")
}
