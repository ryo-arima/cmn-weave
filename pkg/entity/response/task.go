// Package response holds outbound response DTOs for the server REST API.
package response

// DispatchResponse is returned by POST /api/v1/dispatch on success.
type DispatchResponse struct {
	// TaskID is the server-assigned opaque identifier for the new task.
	TaskID string `json:"task_id"`
}

// StatusResponse is returned by GET /api/v1/tasks/:task_id.
type StatusResponse struct {
	TaskID       string `json:"task_id"`
	NodeID       string `json:"node_id,omitempty"`
	State        string `json:"state"`
	Argv         []string `json:"argv,omitempty"`
	ExitCode     int32  `json:"exit_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
	StartedAt    string `json:"started_at,omitempty"`
	FinishedAt   string `json:"finished_at,omitempty"`
}

// TaskSummary is a compact task representation used in list responses.
type TaskSummary struct {
	TaskID    string `json:"task_id"`
	NodeID    string `json:"node_id,omitempty"`
	State     string `json:"state"`
	Argv      []string `json:"argv,omitempty"`
	CreatedAt string `json:"created_at"`
}

// TasksResponse is returned by GET /api/v1/tasks.
type TasksResponse struct {
	Tasks []TaskSummary `json:"tasks"`
}

// BootstrapResponse is returned by POST /api/v1/bootstrap/db.
type BootstrapResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

