package searchquery

import "github.com/jiva-studio/numen/libs/note/domain"

// DocumentQuery asks for the documents of a vault under any of the path prefixes (all when empty), of the
// source kinds (all when empty) and at the paths (all when empty), counting their small chunks with a vector
// under the recipe. Limit bounds the documents returned to Limit+1; zero or below returns all of them.
type DocumentQuery struct {
	PathPrefixes []string
	SourceKinds  []domain.SourceKind
	Paths        []string
	Recipe       string
	Limit        int
}

// DocumentVectorQuery asks for the vectors under the recipe of the small chunks of the documents at the
// paths: at most Limit of them after the chunk After, in order of chunk id.
type DocumentVectorQuery struct {
	Paths  []string
	Recipe string
	After  domain.ChunkID
	Limit  int
}

// WordQuery asks for the small chunks whose text matches an FTS5 expression, within Paths when set, else
// within PathPrefixes and SourceKinds, at most Limit of them.
type WordQuery struct {
	Expression   string
	Paths        []string
	PathPrefixes []string
	SourceKinds  []domain.SourceKind
	Limit        int
}

// WordMatch is one small chunk that matches a words expression, with its negated bm25.
type WordMatch struct {
	Chunk domain.ChunkID
	Score float64
}
