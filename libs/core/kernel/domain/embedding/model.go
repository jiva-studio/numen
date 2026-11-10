// Package embedding holds the identity of an embedding model and the vectors made under it.
package embedding

import (
	"fmt"
	"strconv"
)

// Model names the model a vector came from. It is stored beside the
// vector, and a vector whose model differs from the one configured is stale.
// Every field but MinChunkLength and ChunkTokens is part of the identity. Two
// models of the same name at different widths produce vectors that cannot be
// compared, and a reader needs the width before it decodes stored bytes.
type Model struct {
	Name       string
	Dimensions int

	MaxTokens int

	Pooling string

	From string

	Handling string

	MinChunkLength int

	ChunkTokens int
}

// String is the identity as one value, for a column that holds it.
func (m Model) String() string {
	return m.Name + "@" + strconv.Itoa(m.Dimensions)
}

// Recipe is everything about this model that decides what a vector is, as one value.
// A retained vector is looked up by its source text and the recipe it was made
// under, so a property missing from the recipe can change while the key does
// not. The provider comes first: two places serving one name are two sets of
// rows. The quantization and scale come last: a changed scale gives every stored byte a new meaning.
func (m Model) Recipe() string {
	return fmt.Sprintf("%s|%s|%d|%d|%s|%s|%s@%g",
		m.From, m.Name, m.Dimensions, m.MaxTokens, m.Pooling, m.Handling, QuantizedInt8, Int8Scale)
}
