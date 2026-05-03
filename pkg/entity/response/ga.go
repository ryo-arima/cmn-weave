package response

// GAFloydResponse is returned by POST /api/v1/ga/floyd on success.
type GAFloydResponse struct {
	// TaskID is the server-assigned identifier for the dispatched GA computation.
	TaskID string `json:"task_id"`
}
