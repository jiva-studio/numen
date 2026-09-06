package port

import (
	"context"
	"fmt"
	"strconv"
)

// EmbeddingModel names the model a vector came from. It is stored beside the
// vector, and a vector whose model differs from the one now configured is
// stale: its numbers describe directions another model chose.
//
// Every field is part of the identity. Two models of the same name at different
// widths produce vectors that cannot be compared, and the width is what a
// reader needs to know before it decodes stored bytes.
type EmbeddingModel struct {
	Name       string
	Dimensions int

	// MaxTokens is where the model truncates what it is given. Zero is a model
	// that did not say, and then a chunk is cut at whatever the caller asks.
	//
	// A chunk is cut in words and bounded in characters, so this is turned into
	// characters by a floor and not a measurement: text of another script takes
	// several tokens a word, and a chunk silently truncated is a chunk indexed
	// for text it does not contain.
	MaxTokens int

	// Pooling is how the model's output becomes one vector. Two poolings of one
	// model put a text in two places, so it is part of the identity.
	Pooling string

	// From is where the vectors are made: the model run on this machine, or the
	// service reached at one address. One name is run here and served by more
	// than one place, and each of them answers with numbers of its own.
	From string
}

// String is the identity as one value, for a column that holds it.
func (m EmbeddingModel) String() string {
	return m.Name + "@" + strconv.Itoa(m.Dimensions)
}

// Recipe is everything about this model that decides what a vector is, as one
// value.
//
// A vector kept beyond the row that pointed at it is claimed again by the text
// it was made from and the recipe it was made under. Anything left out of the
// recipe is something that can change while the key does not, and a vector
// found under a key that no longer describes it is worse than one that was
// never kept.
//
// Where the vector was made leads it. Two places serving one name are two sets
// of rows, and a question is asked under the recipe the index was filled with.
//
// How the numbers are stored is the quantisation and the scale it was applied
// at. The scale is the grid the bytes sit on, so a scale moved makes every byte
// already stored mean something else, and the recipe is what notices.
func (m EmbeddingModel) Recipe() string {
	return fmt.Sprintf("%s|%s|%d|%d|%s|%s@%g",
		m.From, m.Name, m.Dimensions, m.MaxTokens, m.Pooling, QuantisedInt8, Int8Scale)
}

// Embedder turns text into vectors. The core asks for it and does not know
// whether the answer is computed on this machine or bought from a service.
//
// Vectors are unit length, so a dot product is a cosine and the sign of a
// dimension carries meaning.
type Embedder interface {
	// Model is the identity every vector this embedder returns belongs to.
	Model() EmbeddingModel
	// Embed returns one vector per text, in the order the texts arrived.
	// Any number of texts may be passed: how a request to the model is
	// bounded is the implementation's business.
	Embed(ctx context.Context, texts []string) ([][]float32, error)

	// Close releases whatever the model holds.
	Close() error
}
