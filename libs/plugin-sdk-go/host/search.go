package host

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/protocol"
)

var (
	// ErrPermissionDenied is what a search call gets from a plugin whose manifest does not declare the permission the call needs.
	ErrPermissionDenied = errors.New("host: the plugin does not hold the permission")
	// ErrNoModel is what a search call gets while the installation has no embedding model.
	ErrNoModel = errors.New("host: the installation has no embedding model")
	// ErrRecipeMismatch is what a call gets that names a recipe other than the one of the model in use.
	ErrRecipeMismatch = errors.New("host: the recipe is not the one of the model in use")
	// ErrInvalidFloor is what a floor gets that is not above zero and at most one.
	ErrInvalidFloor = errors.New("host: a floor is a cosine similarity above 0 and at most 1")
	// ErrLimitExceeded is what an embedding call gets that names more texts, or longer ones, than one call embeds.
	ErrLimitExceeded = errors.New("host: the call exceeds the limits of one embedding call")
	// ErrEmbeddingFailed is what an embedding call gets when the model could not answer.
	ErrEmbeddingFailed = errors.New("host: the embedding model could not answer")
	// ErrInvalidQuery is what a sampling call gets whose query vector has another dimension count than the vectors of the index.
	ErrInvalidQuery = errors.New("host: a query vector does not match the vectors of the index")
)

// The permissions of the search calls. The reads, the embedding included, need PermissionIndexRead, and
// the floor needs PermissionSearchPolicy.
const (
	PermissionIndexRead    = "index:read"
	PermissionSearchPolicy = "search:policy"
)

// The limits the host applies to a search call.
const (
	MaxEmbeddedTexts = 256
	MaxEmbeddedBytes = 8192
	// MaxSampledQueries is how many queries one SampleSimilarities call takes, and MaxSampledPassages
	// how many small chunks the host compares each of them with.
	MaxSampledQueries  = 256
	MaxSampledPassages = 500
)

// EmbeddingState is the model in use and how far a vault is embedded under it.
type EmbeddingState struct {
	// Recipe identifies the model in use. Every vector the host returns, and every floor it stores,
	// is kept under it.
	Recipe string
	// Model is the name of the model.
	Model string
	// FloorHint is the starting floor the plugin of the model reports, zero when it has none.
	FloorHint float64
	// ChunkCount is how many chunks of the vault can carry a vector, and EmbeddedCount how many do.
	ChunkCount    int64
	EmbeddedCount int64
	// IsReady is whether the vault holds chunks and every one of them carries a vector.
	IsReady bool
}

// ExtensionStats is the sources of one file extension and the chunks they hold.
type ExtensionStats struct {
	// Extension is in lower case, without the dot, and empty for a source with none.
	Extension string
	// SourceCount is how many sources have the extension, and ChunkCount how many chunks that can
	// carry a vector belong to them.
	SourceCount int64
	ChunkCount  int64
}

// IndexStats is what the search index of a vault holds and the space it takes.
type IndexStats struct {
	// Recipe, Model and Dimensions identify the model in use.
	Recipe     string
	Model      string
	Dimensions int64
	// ChunkCount is how many chunks can carry a vector, and VectorCount how many do.
	ChunkCount  int64
	VectorCount int64
	// SourceCount is the notes and files of the vault, and Extensions splits them by file extension,
	// the extension with the most sources first.
	SourceCount int64
	Extensions  []ExtensionStats
	// DatabaseBytes is the size of the index database on disk, and VectorBytes the size of the tables
	// that hold vectors.
	DatabaseBytes int64
	VectorBytes   int64
}

// EmbeddedTexts are the vectors the model in use made of texts.
type EmbeddedTexts struct {
	Recipe  string
	Vectors [][]float32
}

// SimilarityQuery is a query vector of the model in use, and the path of the note it was made from.
// The passages of that note are left out of the sample of the query; an empty path leaves out none.
type SimilarityQuery struct {
	Path   string
	Vector []float32
}

// SimilaritySample is the cosine similarity of each query with the small chunks the host sampled.
type SimilaritySample struct {
	// PassageCount is how many small chunks the host sampled, and SourceCount how many sources they
	// belong to.
	PassageCount int
	SourceCount  int
	// Rows holds, for each query in the order given, its similarity with each sampled passage of another
	// note, in the order of the passages. The row of a query with no vector is empty.
	Rows [][]float64
}

// FloorSource is where the floor a search applies comes from.
type FloorSource int

const (
	// FloorFromDefault: no other source answered.
	FloorFromDefault FloorSource = iota + 1
	// FloorFromModelHint: the starting floor the plugin of the embedding model reports.
	FloorFromModelHint
	// FloorFromStore: a plugin stored the floor for the vault and the recipe.
	FloorFromStore
)

// SearchFloor is the floor a search applies to a vault under a recipe, and the floor of each source
// it was chosen from. A source with no floor holds zero.
type SearchFloor struct {
	Floor   float64
	Source  FloorSource
	Stored  float64
	Hint    float64
	Default float64
}

// Search is the search index and the search floor of the vault the host shows, as a plugin works
// them. SearchClient works them through HostSearchService, and tests work them through
// hosttest.Search.
//
// A call that names a recipe other than the one of the model in use returns ErrRecipeMismatch, and a
// call while the installation has no model returns ErrNoModel. A call while the host shows no vault
// returns ErrNoVault.
type Search interface {
	// With is the search of the vault of the identity. A call on it returns ErrUnknownVault when the
	// installation has no such vault.
	With(id domain.VaultID) Search
	SearchIndex
	PassageSearch
	SearchPolicy
}

// SearchIndex reads the search index of a vault. It needs the index:read permission.
type SearchIndex interface {
	// GetEmbeddingState is the model in use and how far the vault is embedded under it. It is read
	// from the index at each call, so a plugin that starts after the vectors exist reads the same
	// state as one that was running when they were made.
	GetEmbeddingState(ctx context.Context) (EmbeddingState, error)
	// GetIndexStats is what the index holds for the vault: its chunks, vectors and sources, the
	// sources by file extension, the model in use and the size on disk. The host reads the counts at
	// most once a second and the sizes at most once every half minute, so a view may ask every few
	// seconds.
	GetIndexStats(ctx context.Context) (IndexStats, error)
	// EmbedTexts embeds the texts with the model in use. At most MaxEmbeddedTexts texts of at most
	// MaxEmbeddedBytes bytes each are embedded in one call.
	EmbedTexts(ctx context.Context, texts []string) (EmbeddedTexts, error)
	// SampleSimilarities is the cosine similarity of each query with the small chunks of the vault that
	// carry a vector under the recipe and belong to a note other than the query's. The host takes at
	// most MaxSampledPassages small chunks in a fixed order, so one index yields one sample, and at most
	// MaxSampledQueries queries per call. A query with another dimension count than the index returns
	// ErrInvalidQuery.
	SampleSimilarities(ctx context.Context, recipe string, queries []SimilarityQuery) (SimilaritySample, error)
	// GetBestSimilarity is the cosine similarity of the passage of the vault nearest the query vector
	// under the recipe, and zero where the vault has none.
	GetBestSimilarity(ctx context.Context, recipe string, query []float32) (float64, error)
}

// PassageSearch searches the text of a vault. It needs the index:read permission.
type PassageSearch interface {
	// SearchPassages is the text of the vault that answers the query, best first, from the index of
	// every source and every text version: by the words, and by the meaning where the installation has
	// an embedding model. An installation with no model answers by the words alone, so the call
	// does not return ErrNoModel. It returns at most MaxSearchedPassagesPerSource passages of one source.
	SearchPassages(ctx context.Context, query PassageQuery) ([]Passage, error)
}

const (
	// MaxSearchedPassagesPerSource is how many passages of one source a SearchPassages call returns at most.
	MaxSearchedPassagesPerSource = 3
	// DefaultSearchedPassages is how many passages a SearchPassages call returns at most when its limit
	// is zero, and MaxSearchedPassages how many it returns at most for any limit.
	DefaultSearchedPassages = 20
	MaxSearchedPassages     = 100
)

// PassageQuery is a question to the passages of a vault.
type PassageQuery struct {
	Query string
	// Limit is how many passages to return: DefaultSearchedPassages when zero, at most MaxSearchedPassages.
	Limit int
	// Kinds are the kinds of source to look in, KindNote or KindFile. Empty looks in every one.
	Kinds []domain.SourceKind
	// PathPrefix limits the search to the sources whose path starts with it, relative to the vault.
	// The index applies it before it cuts the list at Limit. Empty looks in the whole vault.
	PathPrefix string
}

// Passage is a stretch of a source that answers a query.
type Passage struct {
	// Path is the file the text is read from, relative to the vault.
	Path string
	// Text is the words around the hit, cut to a size and so possibly ending mid-sentence.
	Text string
	// Location is what the source's own numbering calls the place, empty when the format offers none.
	Location string
	// Start and Length locate the whole passage in the source text, in bytes.
	Start  int
	Length int
	// Line is the line of the hit counted from the first line of the source's prose, and zero for
	// a source that has no lines.
	Line int
	// Selector locates the hit in the text version, as JSON. Empty for a note.
	Selector string
}

// SearchPolicy reads and sets the floor that a search by meaning applies to a vault. It needs the
// search:policy permission.
type SearchPolicy interface {
	// GetSearchFloor is the floor a search applies to the vault under the recipe, and where it comes from.
	GetSearchFloor(ctx context.Context, recipe string) (SearchFloor, error)
	// SetSearchFloor stores the effective floor of the vault under the recipe. A floor of zero removes
	// the stored floor, and a floor above one or below zero returns ErrInvalidFloor. Setting the floor
	// that is stored changes nothing.
	SetSearchFloor(ctx context.Context, recipe string, floor float64) error
}

// GetVectorsReadyEvent is the event the host emits to plugins when every chunk of the vault of the
// identity carries a vector under the model in use, but the chunks of a source whose indexing a person
// cancelled, which stay without one until the source changes.
func GetVectorsReadyEvent(id domain.VaultID) string {
	return protocol.GetVectorsReadyEvent(string(id))
}

// ReadVectorsReadyEvent returns the identity of the vault that an event of the host names as having
// every chunk embedded, and false for any other event. The host emits the event after each indexing
// pass that leaves no vector pending, so it repeats: a handler reads the state again and acts only on
// what is not done.
func ReadVectorsReadyEvent(event string) (domain.VaultID, bool) {
	vaultID, found := protocol.ReadVectorsReadyEvent(event)
	return domain.VaultID(vaultID), found
}
