package container

import "context"

// FloorAdvisor reports the similarity a chunk starts out having to reach to answer a query under the
// source. The hint is not part of the identity and is absent from the recipe.
type FloorAdvisor interface {
	// GetFloorHint returns the starting floor, and zero where the source has none.
	GetFloorHint(ctx context.Context) float64
}
