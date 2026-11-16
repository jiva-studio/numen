package host

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/core/protocol"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
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
	// ErrLimitExceeded is what a call gets that names more items, or longer ones, than one call takes.
	ErrLimitExceeded = errors.New("host: the call exceeds the limits of one call")
	// ErrEmbeddingFailed is what an embedding call gets when the model could not answer.
	ErrEmbeddingFailed = errors.New("host: the embedding model could not answer")
	// ErrInvalidQuery is what a sampling call gets whose query vector has another dimension count than the vectors of the index.
	ErrInvalidQuery = errors.New("host: a query vector does not match the vectors of the index")
	// ErrInvalidCondition is what ScoreChunks gets for a condition with no text, with both texts or with an
	// empty text. A words query FTS5 rejects gets ErrInvalidWords.
	ErrInvalidCondition = errors.New("host: a condition names neither or both of meaning and words")
	// ErrInvalidWords is what a words query gets that FTS5 rejects.
	ErrInvalidWords = errors.New("host: the words query is not valid FTS5")
	// ErrSearchNotFound is a search the plugin did not start, or one removed or not read for 10 minutes.
	ErrSearchNotFound = errors.New("host: no such search")
	// ErrTooManySearches is a start while the most searches kept are all running.
	ErrTooManySearches = errors.New("host: too many searches are kept")
	// ErrInvalidPageToken is a page token of another search.
	ErrInvalidPageToken = errors.New("host: the page token is not valid for this search")
)

// The permissions a plugin declares in its manifest. The search reads, the embedding included, need
// PermissionIndexRead, and the search floor needs PermissionSearchPolicy. Adding, renaming, removing
// and opening vaults needs PermissionVaultManage, every Window call needs PermissionWindowFocus, and a
// external link opened in the system browser needs PermissionLinkOpen. Permissions are consent and
// API hygiene: a plugin is trusted code, and its own network requests are not gated.
const (
	PermissionIndexRead    = "index:read"
	PermissionSearchPolicy = "search:policy"
	PermissionVaultManage  = "vault:manage"
	PermissionWindowFocus  = "window:focus"
	PermissionLinkOpen     = "link:open"
)

// The limits the host applies to a search call.
const (
	MaxEmbeddedTexts = 256
	MaxEmbeddedBytes = 8192
	// MaxSampledQueries is how many queries one SampleSimilarities call takes, and MaxSampledChunks
	// how many small chunks the host compares each of them with.
	MaxSampledQueries = 256
	MaxSampledChunks  = 500
)

// EmbeddingState is the model in use and how far a vault is embedded under it.
//
//numen:api
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
//
//numen:api
type ExtensionStats struct {
	// Extension is in lower case, without the dot, and empty for a source with none.
	Extension string
	// SourceCount is how many sources have the extension, and ChunkCount how many chunks that can
	// carry a vector belong to them.
	SourceCount int64
	ChunkCount  int64
}

// IndexStats is what the search index of a vault holds and the space it takes.
//
//numen:api
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
// The chunks of that note are left out of the sample of the query; an empty path leaves out none.
type SimilarityQuery struct {
	Path   string
	Vector []float32
}

// SimilaritySample is the cosine similarity of each query with the small chunks the host sampled.
//
//numen:api
type SimilaritySample struct {
	// ChunkCount is how many small chunks the host sampled, and SourceCount how many sources they
	// belong to.
	ChunkCount  int
	SourceCount int
	// Rows holds, for each query in the order given, its similarity with each sampled chunk of another
	// note, in the order of the chunks. The row of a query with no vector is empty.
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

// SearchIndex reads the search index of a vault. It needs the index:read permission. A call naming a
// recipe other than the one of the model in use returns ErrRecipeMismatch, a call with no model
// installed returns ErrNoModel, and a call while the host shows no vault returns ErrNoVault.
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
	// most MaxSampledChunks small chunks in a fixed order, so one index yields one sample, and at most
	// MaxSampledQueries queries per call. A query with another dimension count than the index returns
	// ErrInvalidQuery.
	SampleSimilarities(ctx context.Context, recipe string, queries []SimilarityQuery) (SimilaritySample, error)
	// GetBestSimilarity is the cosine similarity of the chunk of the vault nearest the query vector
	// under the recipe, and zero where the vault has none.
	GetBestSimilarity(ctx context.Context, recipe string, query []float32) (float64, error)
}

// PlaceSearch finds the chunks of the place of a search source nearest a vector. It needs the index:read
// permission. A search source answers a query of the core with it, on the client With the vault of the
// place; a recipe other than the one of the source in use returns ErrRecipeMismatch.
type PlaceSearch interface {
	// FindNearestChunks returns the small chunks nearest the vector under the recipe, nearest first,
	// at or above the floor of the query, else the effective floor of the vault.
	FindNearestChunks(ctx context.Context, query NearestQuery) ([]NearestMatch, error)
}

// NearestQuery is a search of the place of a search source by a vector.
//
//numen:api
type NearestQuery struct {
	// Recipe is the recipe of the place.
	Recipe string
	Vector []float32
	// Limit is the most chunks to return.
	Limit int
	// Kinds are the kinds of source to search, KindNote or KindFile. Empty means every kind.
	Kinds []domain.SourceKind
	// PathPrefix limits the search to the sources whose path starts with it, relative to the vault.
	PathPrefix string
	// Floor is the cosine similarity a chunk reaches at least; zero applies the floor of the vault.
	Floor float64
}

// NearestMatch is a chunk of the place and its cosine similarity with the vector.
//
//numen:api
type NearestMatch struct {
	ChunkID    string
	Similarity float64
}

const (
	// DefaultSearchedResultsPerSource is the number of search results of one source a SearchText call
	// returns when the query names none.
	DefaultSearchedResultsPerSource = 3
	// MaxExcerptBytes is the longest Text of a SearchResult: the excerpt of the chunk around the hit.
	MaxExcerptBytes = 1200
	// DefaultSearchedResults is the maximum number of search results a SearchText call returns when its limit
	// is zero, and MaxSearchedResults the maximum for any limit.
	DefaultSearchedResults = 20
	MaxSearchedResults     = 500
	// MaxScoredChunks is how many chunks one ScoreChunks call takes, MaxScoredConditions how many
	// conditions, and MaxConditionBytes the longest text of a condition.
	MaxScoredChunks     = 500
	MaxScoredConditions = 16
	MaxConditionBytes   = 8192
)

// SearchMode selects the participants of a SearchText call.
type SearchMode int

const (
	// SearchModeHybrid fuses the words, the section names and the meaning. It is the zero value.
	SearchModeHybrid SearchMode = iota
	// SearchModeWords asks the words of the index alone.
	SearchModeWords
	// SearchModeMeaning asks the search source alone. A host with no model returns ErrNoModel.
	SearchModeMeaning
)

// SearchResultQuery is a text search request.
//
//numen:api
type SearchResultQuery struct {
	Query string
	// Limit is the maximum number of search results to return: DefaultSearchedResults when zero, at
	// most MaxSearchedResults.
	Limit int
	// ResultsPerSource is the maximum number of search results of one source:
	// DefaultSearchedResultsPerSource when zero or less, at most Limit.
	ResultsPerSource int
	// Kinds are the kinds of source to search, KindNote or KindFile. Empty means every kind.
	Kinds []domain.SourceKind
	// PathPrefix limits the search to the sources whose path starts with it, relative to the vault, in
	// either Unicode form (NFC or NFD). The index applies it before it truncates the list to Limit. Empty
	// means the whole vault.
	PathPrefix string
	// Mode selects the participants; the zero value is SearchModeHybrid.
	Mode SearchMode
	// IsRawWords passes Query to the words of the index as an FTS5 query (phrases, NEAR, prefix *, AND,
	// OR, NOT), unquoted. It is valid with SearchModeWords only; a query FTS5 rejects returns
	// ErrInvalidWords.
	IsRawWords bool
	// IsWholeText asks for the whole text of the chunk of each result in WholeText, next to the excerpt in
	// Text.
	IsWholeText bool
}

// SearchResult is a span of source text matching a query.
//
//numen:api
type SearchResult struct {
	// ChunkID names the chunk that was hit. ScoreChunks takes it.
	ChunkID string
	// Path is the file the text is read from, relative to the vault.
	Path string
	// Text is the text around the hit, at most MaxExcerptBytes bytes of the chunk, and so possibly
	// starting and ending mid-sentence.
	Text string
	// PlaceLabel is the location label in the source's numbering (page, chapter, time), empty
	// when the format has none.
	PlaceLabel string
	// Start and Length locate the whole search result in the source text, in bytes.
	Start  int
	Length int
	// MatchLine is the line of the hit, counted from 0 at the first line of the source's prose, for every
	// source.
	MatchLine int
	// Selector locates the hit in the text version, as JSON, with the keys of domain.Selector. Empty
	// for a note.
	Selector string
	// Similarity is the cosine similarity of the query with the best small chunk of this result the
	// search by meaning returned; nil when the search by meaning did not return this result.
	Similarity *float64
	// WordScore is the negated bm25 of the best hit of the words in this result: positive, higher is
	// better, comparable only within one query; nil when the search by words did not return it.
	WordScore *float64
	// WholeText is the whole text of the chunk Text is cut from; set only when the query asks with
	// IsWholeText.
	WholeText string
	// Kind is the kind of the file: note, pdf, epub, audio, video, image or file. Set by GetSearch.
	Kind string
	// Title is the note title, else the file name without its extension. Set by GetSearch.
	Title string
	// Score is the score of the result, nil for a file matched by document conditions only. Set by GetSearch.
	Score *float64
	// IsWholeFile is a result with no place in its file: Start and Length are zero and Text is the start of
	// the file. Set by GetSearch.
	IsWholeFile bool
}

// ScoreCondition is one condition ScoreChunks scores a chunk against.
//
//numen:api
type ScoreCondition struct {
	// Meaning is a text compared by meaning. Exactly one of Meaning and Words is set.
	Meaning string
	// Words is an FTS5 query in the syntax of the index (phrases, NEAR, prefix *, AND, OR, NOT).
	Words string
}

// ScoreQuery is the chunks a ScoreChunks call scores and the conditions it scores them against.
//
//numen:api
type ScoreQuery struct {
	// ChunkIDs are the chunk ids of search results, at most MaxScoredChunks.
	ChunkIDs []string
	// Conditions are at most MaxScoredConditions, each text at most MaxConditionBytes.
	Conditions []ScoreCondition
}

// ConditionScore is how one chunk scores against one condition.
//
//numen:api
type ConditionScore struct {
	// Similarity is set for a meaning condition when a small chunk inside the chunk carries a vector:
	// the best cosine similarity among them.
	Similarity *float64
	// IsMatch is, for a meaning condition, whether the similarity reaches the floor, and for a words
	// condition whether the whole text of the chunk matches the query.
	IsMatch bool
}

// ChunkScore is the score of one chunk against each condition.
//
//numen:api
type ChunkScore struct {
	ChunkID string
	// Conditions holds one score per condition, in the order of the conditions.
	Conditions []ConditionScore
}

// ChunkScores is the answer of ScoreChunks.
//
//numen:api
type ChunkScores struct {
	// Floor is the floor applied to the meaning conditions, zero when there are none.
	Floor float64
	// Chunks holds one score per distinct given id the vault holds, in the order given. An id of a
	// source removed or cut again since the search is left out.
	Chunks []ChunkScore
}

// SearchPolicy reads and sets the floor that a search by meaning applies to a vault. It needs the
// search:policy permission. A plugin that works another vault than the shown one
// calls With on SearchClient and keeps the narrow interface of the calls it needs on the result.
type SearchPolicy interface {
	// GetSearchFloor is the floor a search applies to the vault under the recipe, and where it comes from.
	GetSearchFloor(ctx context.Context, recipe string) (SearchFloor, error)
	// SetSearchFloor stores the effective floor of the vault under the recipe. A floor of zero removes
	// the stored floor, and a floor above one or below zero returns ErrInvalidFloor. Setting the floor
	// that is stored changes nothing.
	SetSearchFloor(ctx context.Context, recipe string, floor float64) error
}

// ReadIndexChangedEvent returns the identity of the vault whose index an event of the host names as changed,
// and false for any other event. The host emits the event after each indexing pass that changed the index
// and left nothing pending, so it repeats: a handler reads the state again and acts only on what is not done.
func ReadIndexChangedEvent(event string) (domain.VaultID, bool) {
	vaultID, found := protocol.ReadIndexChangedEvent(event)
	return domain.VaultID(vaultID), found
}
