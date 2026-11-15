package searchpolicyport

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// FloorReader returns the similarity floor of a vault under a recipe.
type FloorReader interface {
	GetFloor(ctx context.Context, v domain.Vault, recipe string) float64
}
