// Package usecase contains the business logic for the client component.
//
// common.go provides CommonUsecase, the single concrete implementation for all
// client operations (task dispatch/status/cancel and admin bootstrap), along
// with the cross-resource Format helper and its table renderers.
package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/goccy/go-yaml"
	"github.com/ryo-arima/cmn-weave/pkg/client/repository"
	"github.com/ryo-arima/cmn-weave/pkg/entity/request"
	"github.com/ryo-arima/cmn-weave/pkg/entity/response"
)

// Format formats v as table, json, or yaml and returns the result as a string.
func Format(format string, v interface{}) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "json":
		b, _ := json.MarshalIndent(v, "", "  ")
		return string(b) + "\n"
	case "yaml":
		b, _ := yaml.Marshal(v)
		return string(b)
	default:
		return tableString(v)
	}
}

func tableString(v interface{}) string {
	switch data := v.(type) {
	case *response.StatusResponse:
		return statusTableString(data)
	case *response.BootstrapResponse:
		return bootstrapTableString(data)
	default:
		b, _ := json.MarshalIndent(data, "", "  ")
		return string(b) + "\n"
	}
}

func statusTableString(r *response.StatusResponse) string {
	buf := &bytes.Buffer{}
	w := tabwriter.NewWriter(buf, 2, 4, 2, ' ', 0)
	fmt.Fprintln(w, strings.Join([]string{"TASK_ID", "STATE", "EXIT_CODE"}, "\t"))
	fmt.Fprintf(w, "%s\t%s\t%d\n", r.TaskID, r.State, r.ExitCode)
	w.Flush()
	return buf.String()
}

func bootstrapTableString(r *response.BootstrapResponse) string {
	buf := &bytes.Buffer{}
	w := tabwriter.NewWriter(buf, 2, 4, 2, ' ', 0)
	fmt.Fprintln(w, strings.Join([]string{"STATUS", "MESSAGE"}, "\t"))
	fmt.Fprintf(w, "%s\t%s\n", r.Status, r.Message)
	w.Flush()
	return buf.String()
}

// AdminUsecase exposes administrative operations for the client CLI.
type AdminUsecase interface {
	BootstrapDB(ctx context.Context) (*response.BootstrapResponse, error)
}

// CommonUsecase is the single usecase implementation for the client CLI.
// It covers task operations (dispatch/status/cancel) and admin bootstrap.
type CommonUsecase struct {
	server repository.TaskClient
	admin  repository.AdminRepository
}

// New creates a CommonUsecase wired to the given repositories.
func New(server repository.TaskClient, admin repository.AdminRepository) *CommonUsecase {
	return &CommonUsecase{server: server, admin: admin}
}

// ---------------------------------------------------------------------------
// Task operations
// ---------------------------------------------------------------------------

// Dispatch submits a new task and returns the server-assigned task ID.
func (rcvr *CommonUsecase) Dispatch(ctx context.Context, argv []string, env []string) (string, error) {
	resp, err := rcvr.server.Dispatch(ctx, &request.DispatchRequest{Argv: argv, Env: env})
	if err != nil {
		return "", fmt.Errorf("dispatch: %w", err)
	}
	return resp.TaskID, nil
}

// Status returns the current status of a task.
func (rcvr *CommonUsecase) Status(ctx context.Context, taskID string) (*response.StatusResponse, error) {
	return rcvr.server.Status(ctx, taskID)
}

// Cancel requests cancellation of a task.
func (rcvr *CommonUsecase) Cancel(ctx context.Context, taskID string) error {
	return rcvr.server.Cancel(ctx, taskID)
}

// ---------------------------------------------------------------------------
// Admin operations
// ---------------------------------------------------------------------------

// BootstrapDB creates all required schema objects directly in the database.
func (rcvr *CommonUsecase) BootstrapDB(ctx context.Context) (*response.BootstrapResponse, error) {
	resp, err := rcvr.admin.BootstrapDB(ctx)
	if err != nil {
		return resp, fmt.Errorf("bootstrap db: %w", err)
	}
	return resp, nil
}
