package embed

import (
	"fmt"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// CheckIdentity says what is missing from the identity a local model's vectors
// are kept under. Where a text is cut off is part of what a vector is, and this
// machine is what does the cutting, so a model run here says where.
func CheckIdentity(identity port.EmbeddingModel, repository string) error {
	if identity.Dimensions <= 0 {
		return fmt.Errorf("%s: dimensions must be known before a vector is stored", repository)
	}
	if identity.MaxTokens <= 0 {
		return fmt.Errorf("%s: where a text is cut off must be known before a vector is stored", repository)
	}
	if identity.Pooling != "" && identity.Pooling != PoolMean && identity.Pooling != PoolHead {
		return fmt.Errorf("%s is pooled %q, and a model is pooled %q or %q",
			identity.Name, identity.Pooling, PoolMean, PoolHead)
	}
	return nil
}
