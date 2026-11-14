package plugin

import "github.com/jiva-studio/numen/libs/note/domain"

// The limits of the search index calls.
const (
	// MaxSampledQueries is how many queries one SampleSimilarities call takes, and MaxSampledChunks how
	// many small chunks it compares each of them with.
	MaxSampledQueries = 256
	MaxSampledChunks  = 500
)

// EmbeddingState is the model in use and how far a vault is embedded under it.
type EmbeddingState struct {
	Recipe        string
	Model         string
	FloorHint     float64
	ChunkCount    int64
	EmbeddedCount int64
}

// IsReady reports whether the vault holds chunks and every one of them carries a vector.
func (s EmbeddingState) IsReady() bool {
	return s.ChunkCount > 0 && s.EmbeddedCount >= s.ChunkCount
}

// EmbeddedTexts are the vectors the model in use made of texts, and its recipe.
type EmbeddedTexts struct {
	Recipe  string
	Vectors [][]float32
}

// SimilarityQuery is a query vector, and the path of the note it was made from. The chunks of
// that note are left out of its sample; an empty path leaves out none.
type SimilarityQuery struct {
	Path   string
	Vector []float32
}

// SimilaritySample is the cosine similarity of each query with the sampled chunks of the vault.
type SimilaritySample struct {
	ChunkCount  int
	SourceCount int
	Rows        [][]float64
}

// FloorSource is where the floor a search applies comes from.
type FloorSource int

const (
	// FloorDefault: no other source answered.
	FloorDefault FloorSource = iota + 1
	// FloorModelHint: the starting floor the plugin of the embedding model reports.
	FloorModelHint
	// FloorStored: a plugin stored the floor for the vault and the recipe.
	FloorStored
)

// SearchFloor is the floor a search applies to a vault under a recipe, and the sources it was chosen from.
type SearchFloor struct {
	Floor   float64
	Source  FloorSource
	Stored  float64
	Hint    float64
	Default float64
}

// SearchResultQuery is a text search request.
type SearchResultQuery struct {
	Query      string
	Limit      int
	Kinds      []domain.SourceKind
	PathPrefix string
}

// ExtensionStats is the sources of one file extension and the chunks they hold.
type ExtensionStats struct {
	Extension string
	Sources   int64
	Chunks    int64
}

// IndexStats is what the index of a vault holds, and the space it takes.
type IndexStats struct {
	Recipe        string
	Model         string
	Dimensions    int64
	Chunks        int64
	Vectors       int64
	Sources       int64
	Extensions    []ExtensionStats
	DatabaseBytes int64
	VectorBytes   int64
}
