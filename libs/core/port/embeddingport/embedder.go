package embeddingport

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

	// MaxTokens is the model's truncation limit. Zero means the model did not
	// report one, and a chunk is then cut at the length the caller requests.
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
// A vector retained beyond the row that pointed at it is looked up by the text
// it was made from and the recipe it was made under. A property missing from
// the recipe can change while the key does not, and a vector found under a key
// that does not describe it is worse than no stored vector.
//
// The provider of the vector comes first. Two places serving one name are two
// sets of rows, and a query uses the recipe the index was filled with.
//
// How the numbers are stored is the quantisation and the scale applied to it.
// The scale defines the grid of byte values, so a changed scale gives every
// stored byte a different meaning, and the recipe detects the change.
func (m EmbeddingModel) Recipe() string {
	return fmt.Sprintf("%s|%s|%d|%d|%s|%s@%g",
		m.From, m.Name, m.Dimensions, m.MaxTokens, m.Pooling, QuantisedInt8, Int8Scale)
}

// Embedder converts text into vectors. The core requests them and does not know
// whether they are computed on this machine or obtained from a service.
//
// Vectors are unit length, so a dot product is a cosine and the sign of a
// dimension carries meaning.
type Embedder interface {
	// GetModel is the identity every vector this embedder returns belongs to.
	GetModel() EmbeddingModel
	// Embed returns one vector per text, in the order the texts arrived.
	// Any number of texts may be passed: how a request to the model is
	// bounded is the implementation's business.
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	// Concurrency is how many batches the embedder runs in parallel, and one for
	// an embedder that runs them one at a time.
	Concurrency() int

	// Close releases whatever the model holds.
	Close() error
}

// FloorAdvisor is an Embedder that reports the similarity a passage starts out
// having to reach to answer a query under its model.
//
// The hint is not part of the model's identity and is absent from Recipe.
type FloorAdvisor interface {
	// GetFloorHint returns the starting floor, and zero where the model has none.
	GetFloorHint(ctx context.Context) float64
}
