// Package usecase contains the business logic for the server component.
//
// common.go provides CommonUsecase, the single concrete implementation that
// handles all task, node, and GA operations for the server.
package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/ryo-arima/cmn-weave/pkg/entity/model"
	"github.com/ryo-arima/cmn-weave/pkg/entity/request"
	"github.com/ryo-arima/cmn-weave/pkg/entity/response"
	"github.com/ryo-arima/cmn-weave/pkg/server/repository"
)

// CommonUsecase is the single usecase implementation for the server.
// It owns all business logic for tasks, nodes, and GA operations.
type CommonUsecase struct {
	tasks  repository.TaskRepository
	agents repository.AgentDispatcher
	nodes  repository.NodeRepository
}

// New creates a CommonUsecase wired to the given repositories.
func New(tasks repository.TaskRepository, agents repository.AgentDispatcher, nodes repository.NodeRepository) *CommonUsecase {
	return &CommonUsecase{tasks: tasks, agents: agents, nodes: nodes}
}

// ---------------------------------------------------------------------------
// Task operations
// ---------------------------------------------------------------------------

// Dispatch creates a task, persists it to PostgreSQL, and forwards it to an agent via gRPC.
func (rcvr *CommonUsecase) Dispatch(ctx context.Context, req *request.DispatchRequest) (*response.DispatchResponse, error) {
	task := &model.Task{
		ID:        generateID(),
		State:     model.TaskStatePending,
		Argv:      req.Argv,
		Env:       req.Env,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := rcvr.tasks.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("persist task: %w", err)
	}
	if err := rcvr.agents.Dispatch(ctx, task); err != nil {
		_ = rcvr.tasks.UpdateState(ctx, task.ID, model.TaskStateFailed, 0, err.Error())
		return nil, fmt.Errorf("dispatch to agent: %w", err)
	}
	_ = rcvr.tasks.UpdateState(ctx, task.ID, model.TaskStateRunning, 0, "")
	return &response.DispatchResponse{TaskID: task.ID}, nil
}

// Status queries PostgreSQL for the persisted state of a task.
func (rcvr *CommonUsecase) Status(ctx context.Context, taskID string) (*response.StatusResponse, error) {
	return rcvr.ShowTask(ctx, taskID)
}

// ShowTask returns the full detail of a task.
func (rcvr *CommonUsecase) ShowTask(ctx context.Context, taskID string) (*response.StatusResponse, error) {
	task, err := rcvr.tasks.Get(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get task %s: %w", taskID, err)
	}
	r := &response.StatusResponse{
		TaskID:       task.ID,
		NodeID:       task.NodeID,
		State:        string(task.State),
		Argv:         task.Argv,
		ExitCode:     task.ExitCode,
		ErrorMessage: task.ErrorMessage,
		CreatedAt:    task.CreatedAt.Format(time.RFC3339),
	}
	if task.StartedAt != nil {
		r.StartedAt = task.StartedAt.Format(time.RFC3339)
	}
	if task.FinishedAt != nil {
		r.FinishedAt = task.FinishedAt.Format(time.RFC3339)
	}
	return r, nil
}

// Cancel requests cancellation of a running task on the agent and updates PostgreSQL.
func (rcvr *CommonUsecase) Cancel(ctx context.Context, taskID string) error {
	if err := rcvr.agents.Cancel(ctx, taskID); err != nil {
		return fmt.Errorf("cancel task %s on agent: %w", taskID, err)
	}
	return rcvr.tasks.UpdateState(ctx, taskID, model.TaskStateCancelled, 0, "")
}

// ListTasks returns a summary of all tasks.
func (rcvr *CommonUsecase) ListTasks(ctx context.Context) (*response.TasksResponse, error) {
	tasks, err := rcvr.tasks.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	summaries := make([]response.TaskSummary, 0, len(tasks))
	for _, t := range tasks {
		summaries = append(summaries, response.TaskSummary{
			TaskID:    t.ID,
			NodeID:    t.NodeID,
			State:     string(t.State),
			Argv:      t.Argv,
			CreatedAt: t.CreatedAt.Format(time.RFC3339),
		})
	}
	return &response.TasksResponse{Tasks: summaries}, nil
}

// StartGAFloyd encodes the Floyd problem parameters and dispatches a ga-floyd task.
func (rcvr *CommonUsecase) StartGAFloyd(ctx context.Context, req *request.GAFloydRequest) (*response.GAFloydResponse, error) {
	if req.PopulationSize <= 0 {
		req.PopulationSize = 100
	}
	if req.Generations <= 0 {
		req.Generations = 500
	}
	if req.MutationRate <= 0 {
		req.MutationRate = 0.01
	}
	edgesJSON, err := json.Marshal(req.Edges)
	if err != nil {
		return nil, fmt.Errorf("marshal edges: %w", err)
	}
	argv := []string{
		"ga-floyd",
		"--size", strconv.Itoa(req.Size),
		"--pop", strconv.Itoa(req.PopulationSize),
		"--gens", strconv.Itoa(req.Generations),
	}
	env := []string{
		fmt.Sprintf("GA_EDGES=%s", edgesJSON),
		fmt.Sprintf("GA_MUTATION_RATE=%s", strconv.FormatFloat(req.MutationRate, 'f', -1, 64)),
	}
	dispatchResp, err := rcvr.Dispatch(ctx, &request.DispatchRequest{Argv: argv, Env: env})
	if err != nil {
		return nil, fmt.Errorf("start ga-floyd: %w", err)
	}
	return &response.GAFloydResponse{TaskID: dispatchResp.TaskID}, nil
}

// generateID returns a unique task identifier.
// TODO: replace with google/uuid for a proper UUIDv4.
func generateID() string {
	return fmt.Sprintf("task-%d", time.Now().UnixNano())
}

// ---------------------------------------------------------------------------
// Node operations
// ---------------------------------------------------------------------------

// ListNodes returns all registered agent nodes.
func (rcvr *CommonUsecase) ListNodes(ctx context.Context) (*response.NodesResponse, error) {
	nodes, err := rcvr.nodes.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}
	items := make([]response.NodeResponse, 0, len(nodes))
	for _, n := range nodes {
		items = append(items, response.NodeResponse{
			ID:        n.ID,
			Hostname:  n.Hostname,
			IPv4:      n.IPv4.String(),
			Status:    string(n.Status),
			CreatedAt: n.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	return &response.NodesResponse{Nodes: items}, nil
}

// ShowNode returns the detail of a single agent node.
func (rcvr *CommonUsecase) ShowNode(ctx context.Context, nodeID string) (*response.NodeResponse, error) {
	n, err := rcvr.nodes.Get(ctx, nodeID)
	if err != nil {
		return nil, fmt.Errorf("get node %s: %w", nodeID, err)
	}
	r := &response.NodeResponse{
		ID:       n.ID,
		Hostname: n.Hostname,
		Status:   string(n.Status),
	}
	if n.IPv4 != nil {
		r.IPv4 = n.IPv4.String()
	}
	if !n.CreatedAt.IsZero() {
		r.CreatedAt = n.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	return r, nil
}

