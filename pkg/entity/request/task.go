// Package request holds inbound request DTOs for the server REST API.
package request

// DispatchRequest is the JSON body for POST /api/v1/dispatch.
type DispatchRequest struct {
	// Argv is the command and its arguments to execute on the agent.
	Argv []string `json:"argv"`
	// Env contains optional KEY=VALUE environment variable overrides.
	Env []string `json:"env,omitempty"`
}
