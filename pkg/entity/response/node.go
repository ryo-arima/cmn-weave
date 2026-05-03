package response

// NodeResponse is a single agent-node representation.
type NodeResponse struct {
	ID        string `json:"id"`
	Hostname  string `json:"hostname"`
	IPv4      string `json:"ipv4,omitempty"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// NodesResponse is returned by GET /api/v1/nodes.
type NodesResponse struct {
	Nodes []NodeResponse `json:"nodes"`
}
