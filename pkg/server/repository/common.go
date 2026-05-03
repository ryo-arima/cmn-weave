// Package repository owns outbound I/O for the server component.
//
// common.go declares the repository interfaces consumed by the usecase layer
// and provides all concrete implementations: PSQLTaskRepository,
// PSQLNodeRepository, and GRPCAgentDispatcher.
package repository

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"sync"

	agentpb "github.com/ryo-arima/cmn-weave/pkg/agent/grpc/auto"
	"github.com/ryo-arima/cmn-weave/pkg/entity/model"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// ---------------------------------------------------------------------------
// Interfaces
// ---------------------------------------------------------------------------

// TaskRepository persists task state to a PostgreSQL database.
type TaskRepository interface {
	// Create stores a new task row.
	Create(ctx context.Context, task *model.Task) error
	// Get retrieves a task by its ID.
	Get(ctx context.Context, taskID string) (*model.Task, error)
	// List returns all task rows ordered by created_at descending.
	List(ctx context.Context) ([]*model.Task, error)
	// UpdateState sets the task state, exit code, and error message.
	UpdateState(ctx context.Context, taskID string, state model.TaskState, exitCode int32, errMsg string) error
	// Bootstrap creates the tasks table if it does not already exist.
	Bootstrap(ctx context.Context) error
}

// NodeRepository persists agent-node records to a PostgreSQL database.
type NodeRepository interface {
	// List returns all registered nodes.
	List(ctx context.Context) ([]*model.Node, error)
	// Get retrieves a single node by its ID.
	Get(ctx context.Context, nodeID string) (*model.Node, error)
	// Bootstrap creates the nodes table if it does not already exist.
	Bootstrap(ctx context.Context) error
}

// AgentDispatcher dispatches work to remote agents via gRPC and queries their status.
type AgentDispatcher interface {
	// Dispatch forwards the task to an available agent endpoint.
	Dispatch(ctx context.Context, task *model.Task) error
	// Cancel requests cooperative termination of a task on the agent.
	Cancel(ctx context.Context, taskID string) error
	// QueryStatus fetches the current execution state from the agent.
	QueryStatus(ctx context.Context, taskID string) (model.TaskState, int32, string, error)
}

// ---------------------------------------------------------------------------
// PSQLTaskRepository
// ---------------------------------------------------------------------------

// PSQLTaskRepository implements TaskRepository against a PostgreSQL database.
//
// To activate the driver, register pgx in the binary entry point:
//
//	import _ "github.com/jackc/pgx/v5/stdlib"
//
// Then open the connection and pass it here:
//
//	db, err := sql.Open("pgx", cfg.PostgresDSN)
//	repo := NewPSQLTaskRepository(db)
type PSQLTaskRepository struct {
	db *sql.DB
}

// NewPSQLTaskRepository creates a PSQLTaskRepository.
// Pass nil for a no-op stub (useful before PostgreSQL is configured).
func NewPSQLTaskRepository(db *sql.DB) *PSQLTaskRepository {
	return &PSQLTaskRepository{db: db}
}

// Create inserts a new task row.
// TODO: implement with INSERT INTO tasks (id, state, argv, env, ...) VALUES ($1, ...).
func (rcvr *PSQLTaskRepository) Create(ctx context.Context, task *model.Task) error {
	if rcvr.db == nil {
		return nil // no-op stub
	}
	_ = task
	return fmt.Errorf("psql: Create not yet implemented")
}

// Get retrieves a task by ID.
// TODO: implement with SELECT ... FROM tasks WHERE id = $1.
func (rcvr *PSQLTaskRepository) Get(ctx context.Context, taskID string) (*model.Task, error) {
	if rcvr.db == nil {
		return &model.Task{ID: taskID, State: model.TaskStatePending}, nil
	}
	return nil, fmt.Errorf("psql: Get not yet implemented")
}

// List returns all task rows.
// TODO: implement with SELECT ... FROM tasks ORDER BY created_at DESC.
func (rcvr *PSQLTaskRepository) List(ctx context.Context) ([]*model.Task, error) {
	if rcvr.db == nil {
		return []*model.Task{}, nil // no-op stub
	}
	return nil, fmt.Errorf("psql: List not yet implemented")
}

// UpdateState sets state, exit_code, and error_message for a task.
// TODO: implement with UPDATE tasks SET state=$2, exit_code=$3, error_message=$4 WHERE id=$1.
func (rcvr *PSQLTaskRepository) UpdateState(ctx context.Context, taskID string, state model.TaskState, exitCode int32, errMsg string) error {
	if rcvr.db == nil {
		return nil // no-op stub
	}
	return fmt.Errorf("psql: UpdateState not yet implemented")
}

// Bootstrap creates the tasks table if it does not already exist.
func (rcvr *PSQLTaskRepository) Bootstrap(ctx context.Context) error {
	if rcvr.db == nil {
		return fmt.Errorf("psql: no database connection")
	}
	_, err := rcvr.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS tasks (
			id            TEXT PRIMARY KEY,
			node_id       TEXT,
			state         TEXT NOT NULL,
			argv          JSONB,
			env           JSONB,
			exit_code     INTEGER NOT NULL DEFAULT 0,
			error_message TEXT,
			created_at    TIMESTAMPTZ NOT NULL,
			updated_at    TIMESTAMPTZ NOT NULL,
			started_at    TIMESTAMPTZ,
			finished_at   TIMESTAMPTZ
		)`)
	if err != nil {
		return fmt.Errorf("psql: bootstrap tasks: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// PSQLNodeRepository
// ---------------------------------------------------------------------------

// PSQLNodeRepository implements NodeRepository against a PostgreSQL database.
type PSQLNodeRepository struct {
	db *sql.DB
}

// NewPSQLNodeRepository creates a PSQLNodeRepository.
// Pass nil for a no-op stub (useful before PostgreSQL is configured).
func NewPSQLNodeRepository(db *sql.DB) *PSQLNodeRepository {
	return &PSQLNodeRepository{db: db}
}

// List returns all registered nodes.
// TODO: implement with SELECT ... FROM nodes ORDER BY created_at DESC.
func (rcvr *PSQLNodeRepository) List(ctx context.Context) ([]*model.Node, error) {
	if rcvr.db == nil {
		return []*model.Node{}, nil // no-op stub
	}
	return nil, fmt.Errorf("psql: List nodes not yet implemented")
}

// Get retrieves a single node by ID.
// TODO: implement with SELECT ... FROM nodes WHERE id = $1.
func (rcvr *PSQLNodeRepository) Get(ctx context.Context, nodeID string) (*model.Node, error) {
	if rcvr.db == nil {
		return &model.Node{ID: nodeID}, nil // no-op stub
	}
	return nil, fmt.Errorf("psql: Get node not yet implemented")
}

// Bootstrap creates the nodes table if it does not already exist.
func (rcvr *PSQLNodeRepository) Bootstrap(ctx context.Context) error {
	if rcvr.db == nil {
		return fmt.Errorf("psql: no database connection")
	}
	_, err := rcvr.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS nodes (
			id         TEXT PRIMARY KEY,
			hostname   TEXT NOT NULL,
			ipv4       TEXT,
			ipv6       TEXT,
			status     TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			deleted_at TIMESTAMPTZ
		)`)
	if err != nil {
		return fmt.Errorf("psql: bootstrap nodes: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// GRPCAgentDispatcher
// ---------------------------------------------------------------------------

// GRPCAgentDispatcher implements AgentDispatcher by sending RPCs to agent
// endpoints over mTLS gRPC. Connections are cached per endpoint.
type GRPCAgentDispatcher struct {
	endpoints []string
	creds     credentials.TransportCredentials
	mu        sync.Mutex
	conns     map[string]*grpc.ClientConn
}

// NewGRPCAgentDispatcher creates a GRPCAgentDispatcher targeting the given
// endpoints. tlsCfg must be a client-side mTLS config (RootCAs + Certificates).
func NewGRPCAgentDispatcher(endpoints []string, tlsCfg *tls.Config) *GRPCAgentDispatcher {
	return &GRPCAgentDispatcher{
		endpoints: endpoints,
		creds:     credentials.NewTLS(tlsCfg),
		conns:     make(map[string]*grpc.ClientConn),
	}
}

// dial returns a cached or newly-dialled *grpc.ClientConn for addr.
func (rcvr *GRPCAgentDispatcher) dial(addr string) (*grpc.ClientConn, error) {
	rcvr.mu.Lock()
	defer rcvr.mu.Unlock()
	if cc, ok := rcvr.conns[addr]; ok {
		return cc, nil
	}
	cc, err := grpc.NewClient(addr, grpc.WithTransportCredentials(rcvr.creds))
	if err != nil {
		return nil, fmt.Errorf("grpc dial %s: %w", addr, err)
	}
	rcvr.conns[addr] = cc
	return cc, nil
}

// firstEndpoint returns the first configured agent endpoint or an error.
func (rcvr *GRPCAgentDispatcher) firstEndpoint() (string, error) {
	if len(rcvr.endpoints) == 0 {
		return "", fmt.Errorf("grpc agent: no endpoints configured")
	}
	return rcvr.endpoints[0], nil
}

// taskStateFromProto maps an agentpb.TaskState to model.TaskState.
func taskStateFromProto(s agentpb.TaskState) model.TaskState {
	switch s {
	case agentpb.TaskState_TASK_STATE_PENDING:
		return model.TaskStatePending
	case agentpb.TaskState_TASK_STATE_RUNNING:
		return model.TaskStateRunning
	case agentpb.TaskState_TASK_STATE_SUCCEEDED:
		return model.TaskStateSucceeded
	case agentpb.TaskState_TASK_STATE_FAILED:
		return model.TaskStateFailed
	case agentpb.TaskState_TASK_STATE_CANCELLED:
		return model.TaskStateCancelled
	default:
		return model.TaskStateUnspecified
	}
}

// Dispatch forwards the task to the first available agent endpoint via the
// AgentService.Dispatch RPC.
func (rcvr *GRPCAgentDispatcher) Dispatch(ctx context.Context, task *model.Task) error {
	addr, err := rcvr.firstEndpoint()
	if err != nil {
		return err
	}
	cc, err := rcvr.dial(addr)
	if err != nil {
		return err
	}
	client := agentpb.NewAgentServiceClient(cc)
	req := &agentpb.DispatchRequest{
		TaskId: task.ID,
		Argv:   task.Argv,
		Env:    task.Env,
	}
	_, err = client.Dispatch(ctx, req)
	if err != nil {
		return fmt.Errorf("agent Dispatch RPC: %w", err)
	}
	return nil
}

// Cancel sends a Cancel RPC to the agent executing taskID.
func (rcvr *GRPCAgentDispatcher) Cancel(ctx context.Context, taskID string) error {
	addr, err := rcvr.firstEndpoint()
	if err != nil {
		return err
	}
	cc, err := rcvr.dial(addr)
	if err != nil {
		return err
	}
	client := agentpb.NewAgentServiceClient(cc)
	_, err = client.Cancel(ctx, &agentpb.CancelRequest{TaskId: taskID})
	if err != nil {
		return fmt.Errorf("agent Cancel RPC: %w", err)
	}
	return nil
}

// QueryStatus fetches the current state of a task from the agent.
func (rcvr *GRPCAgentDispatcher) QueryStatus(ctx context.Context, taskID string) (model.TaskState, int32, string, error) {
	addr, err := rcvr.firstEndpoint()
	if err != nil {
		return model.TaskStateUnspecified, 0, "", err
	}
	cc, err := rcvr.dial(addr)
	if err != nil {
		return model.TaskStateUnspecified, 0, "", err
	}
	client := agentpb.NewAgentServiceClient(cc)
	resp, err := client.Status(ctx, &agentpb.StatusRequest{TaskId: taskID})
	if err != nil {
		return model.TaskStateUnspecified, 0, "", fmt.Errorf("agent Status RPC: %w", err)
	}
	return taskStateFromProto(resp.State), resp.ExitCode, resp.ErrorMessage, nil
}

