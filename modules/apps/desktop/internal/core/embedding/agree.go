package embedding

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// Agreement is how near two embedders' answers stand for the two to be one
// model. The same weights answer the same text with the same vector, and the
// distance left is what arithmetic on two machines leaves.
const Agreement = 0.99

// asked is the text both are asked about. It is short, so the question costs a
// service almost nothing, and it carries several scripts, so two models that
// differ only outside one alphabet still differ here.
const asked = "Śrī Caitanya Mahāprabhu — Кришна — the holy name"

// Agree says whether two embedders are two placements of one model.
//
// A vault's vectors are made by one of them and a question is asked with the
// other, and a question in another space finds nothing however well it is
// written. Nothing in a configuration file shows this: two placements name a
// model by whatever each of them calls it, so they are asked instead.
func Agree(ctx context.Context, indexing, asking port.Embedder) error {
	if indexing == nil || asking == nil {
		return nil
	}
	if a, b := indexing.Model().Recipe(), asking.Model().Recipe(); a != b {
		return fmt.Errorf("vectors are made under %s and asked for under %s", a, b)
	}

	first, err := indexing.Embed(ctx, []string{asked})
	if err != nil {
		return fmt.Errorf("asking what indexes: %w", err)
	}
	second, err := asking.Embed(ctx, []string{asked})
	if err != nil {
		return fmt.Errorf("asking what questions are embedded by: %w", err)
	}
	if len(first) != 1 || len(second) != 1 {
		return fmt.Errorf("one text was answered with %d and %d vectors", len(first), len(second))
	}
	if near := dot(first[0], second[0]); near < Agreement {
		return fmt.Errorf(
			"%s and %s are not one model: they put one text %.3f apart, and a question embedded by the second finds nothing the first indexed",
			indexing.Model(), asking.Model(), 1-near)
	}
	return nil
}

// dot is the cosine of two unit vectors, and nothing for two of different
// widths.
func dot(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}
	var sum float64
	for i := range a {
		sum += float64(a[i]) * float64(b[i])
	}
	return sum
}
