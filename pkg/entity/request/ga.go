package request

// GAFloydRequest is the JSON body for POST /api/v1/ga/floyd.
//
// The endpoint accepts a weighted directed graph and GA hyperparameters.
// The server encodes the problem and dispatches it to an available agent
// as a ga-floyd task.
type GAFloydRequest struct {
	// Size is the number of vertices in the graph.
	Size int `json:"size"`
	// Edges defines the directed, weighted edges.
	Edges []GAEdge `json:"edges"`
	// PopulationSize is the number of candidate solutions per generation.
	// Defaults to 100 when zero.
	PopulationSize int `json:"population_size,omitempty"`
	// Generations is the maximum number of GA iterations.
	// Defaults to 500 when zero.
	Generations int `json:"generations,omitempty"`
	// MutationRate is the per-gene mutation probability in [0.0, 1.0].
	// Defaults to 0.01 when zero.
	MutationRate float64 `json:"mutation_rate,omitempty"`
}

// GAEdge represents a directed, weighted edge in the input graph.
type GAEdge struct {
	From   int `json:"from"`
	To     int `json:"to"`
	Weight int `json:"weight"`
}
