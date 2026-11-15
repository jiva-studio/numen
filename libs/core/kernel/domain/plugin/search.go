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

// SearchMode selects the participants of a text search. The zero value fuses all of them.
type SearchMode int

const (
	// SearchModeHybrid: words, section names and meaning, fused.
	SearchModeHybrid SearchMode = iota
	// SearchModeWords: the words of the index alone.
	SearchModeWords
	// SearchModeMeaning: the search source alone.
	SearchModeMeaning
)

// SearchResultQuery is a text search request. ResultsPerSource is the most results of one source,
// three where it is zero or less. IsRawWords passes Query to the words as an FTS5 query and is valid
// in SearchModeWords only. IsWholeText keeps the whole text of each chunk next to its excerpt.
type SearchResultQuery struct {
	Query            string
	Limit            int
	ResultsPerSource int
	Kinds            []domain.SourceKind
	PathPrefix       string
	Mode             SearchMode
	IsRawWords       bool
	IsWholeText      bool
}

// ScoreCondition is one condition a chunk is scored against: a text compared by meaning or an FTS5
// query matched against the words. Exactly one of the two is set.
type ScoreCondition struct {
	Meaning string
	Words   string
}

// ScoreQuery is the chunks to score and the conditions to score them against.
type ScoreQuery struct {
	ChunkIDs   []domain.ChunkID
	Conditions []ScoreCondition
}

// ConditionScore is how one chunk scores against one condition. Similarity is set for a meaning
// condition when a small chunk inside the chunk carries a vector.
type ConditionScore struct {
	Similarity *float64
	IsMatch    bool
}

// ChunkScore is the score of one chunk against each condition, in the order of the conditions.
type ChunkScore struct {
	ChunkID    domain.ChunkID
	Conditions []ConditionScore
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
