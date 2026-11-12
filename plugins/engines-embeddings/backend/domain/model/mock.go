package model

// The width of a mock vector: a part every text shares, and a part its words
// are placed in.
const (
	MockSharedDimensions = 64
	MockWordDimensions   = 2048
)

// Prefixes of the mock model, those of a model that tells a question from a passage. The mock
// vector does not depend on them, so a test can read what was sent without the search changing.
const (
	MockQueryPrefix    = "query: "
	MockDocumentPrefix = "passage: "
)

// GetMockModel returns the identity of the vectors the mock embedder answers with.
func GetMockModel() ModelInfo {
	return ModelInfo{
		Name:           "mock-embedding",
		Dimensions:     MockSharedDimensions + MockWordDimensions,
		MaxTokens:      512,
		Pooling:        PoolMean,
		From:           "mock",
		Concurrency:    1,
		FloorHint:      0,
		MinChunkLength: 0,
		ChunkTokens:    0,
		Handling:       Handling{QueryPrefix: MockQueryPrefix, DocumentPrefix: MockDocumentPrefix, File: "", Normalize: true, ExtraInputs: nil},
	}
}
