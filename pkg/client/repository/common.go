// Package repository owns outbound I/O for the client component.
//
// common.go holds HTTPServerClient, its constructor, the TaskClient interface
// and all HTTP methods, the AdminRepository interface, GORMAdminRepository,
// and small shared utilities.
package repository

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/ryo-arima/cmn-weave/pkg/entity/request"
	"github.com/ryo-arima/cmn-weave/pkg/entity/response"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// HTTPServerClient is an mTLS HTTP client targeting the server REST API.
// Resource-specific interfaces (TaskClient, …) are implemented on this type.
type HTTPServerClient struct {
	base   string
	client *http.Client
}

// NewHTTPServerClient creates an HTTPServerClient targeting the given base URL
// (e.g. "https://server:8443") with the given mTLS config.
func NewHTTPServerClient(base string, tlsCfg *tls.Config) *HTTPServerClient {
	return &HTTPServerClient{
		base: strings.TrimRight(base, "/"),
		client: &http.Client{
			Transport: &http.Transport{TLSClientConfig: tlsCfg},
		},
	}
}

// ---------------------------------------------------------------------------
// TaskClient
// ---------------------------------------------------------------------------

// TaskClient communicates with the server's task REST endpoints over mTLS.
type TaskClient interface {
	Dispatch(ctx context.Context, req *request.DispatchRequest) (*response.DispatchResponse, error)
	Status(ctx context.Context, taskID string) (*response.StatusResponse, error)
	Cancel(ctx context.Context, taskID string) error
}

// Dispatch calls POST /api/v1/dispatch.
func (rcvr *HTTPServerClient) Dispatch(ctx context.Context, req *request.DispatchRequest) (*response.DispatchResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, rcvr.base+"/api/v1/dispatch", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := rcvr.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("dispatch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("dispatch: unexpected status %s", resp.Status)
	}
	var out response.DispatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &out, nil
}

// Status calls GET /api/v1/tasks/:task_id.
func (rcvr *HTTPServerClient) Status(ctx context.Context, taskID string) (*response.StatusResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, rcvr.base+"/api/v1/tasks/"+taskID, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	resp, err := rcvr.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("status: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status: unexpected status %s", resp.Status)
	}
	var out response.StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &out, nil
}

// Cancel calls DELETE /api/v1/tasks/:task_id.
func (rcvr *HTTPServerClient) Cancel(ctx context.Context, taskID string) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, rcvr.base+"/api/v1/tasks/"+taskID, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	resp, err := rcvr.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("cancel: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("cancel: unexpected status %s", resp.Status)
	}
	return nil
}

// AdminRepository performs direct database administration operations.
// It is consumed only by admin-class client commands (e.g. "bootstrap db").
type AdminRepository interface {
	// BootstrapDB creates all schema objects that do not already exist.
	BootstrapDB(ctx context.Context) (*response.BootstrapResponse, error)
}

// GORMAdminRepository implements AdminRepository using GORM with a PostgreSQL driver.
type GORMAdminRepository struct {
	db *gorm.DB
}

// NewGORMAdminRepository opens a PostgreSQL connection via GORM using the given DSN.
func NewGORMAdminRepository(dsn string) (*GORMAdminRepository, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	return &GORMAdminRepository{db: db}, nil
}

// BootstrapDB creates the tasks and nodes tables if they do not already exist.
func (rcvr *GORMAdminRepository) BootstrapDB(ctx context.Context) (*response.BootstrapResponse, error) {
	ddls := []struct {
		name string
		sql  string
	}{
		{
			name: "tasks",
			sql: `CREATE TABLE IF NOT EXISTS tasks (
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
			)`,
		},
		{
			name: "nodes",
			sql: `CREATE TABLE IF NOT EXISTS nodes (
				id         TEXT PRIMARY KEY,
				hostname   TEXT NOT NULL,
				ipv4       TEXT,
				ipv6       TEXT,
				status     TEXT NOT NULL,
				created_at TIMESTAMPTZ NOT NULL,
				updated_at TIMESTAMPTZ NOT NULL,
				deleted_at TIMESTAMPTZ
			)`,
		},
	}

	errs := []string{}
	for _, d := range ddls {
		if err := rcvr.db.WithContext(ctx).Exec(d.sql).Error; err != nil {
			errs = append(errs, d.name+": "+err.Error())
		}
	}
	if len(errs) > 0 {
		msg := joinStrings(errs, "; ")
		return &response.BootstrapResponse{Status: "error", Message: msg},
			fmt.Errorf("bootstrap: %s", msg)
	}
	return &response.BootstrapResponse{
		Status:  "ok",
		Message: "all tables created or already up-to-date",
	}, nil
}

// joinStrings joins a slice with a separator (avoids importing strings).
func joinStrings(ss []string, sep string) string {
	if len(ss) == 0 {
		return ""
	}
	out := ss[0]
	for _, s := range ss[1:] {
		out += sep + s
	}
	return out
}
