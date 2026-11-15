package searchquery

import "github.com/jiva-studio/numen/libs/note/domain"

// ScoreKind says what the scores of one result set mean.
type ScoreKind int

const (
	// ScoreNone is a document predicate: no result has a score.
	ScoreNone ScoreKind = iota
	// ScoreMeaning is the similarity by meaning.
	ScoreMeaning
	// ScoreWords is the negated bm25 of the words.
	ScoreWords
	// ScoreFused is the reciprocal rank fusion of both, normalized to 0..1.
	ScoreFused
	// ScoreRerank is the score a reranking service gave.
	ScoreRerank
)

// GetScoreKind returns what the scores of the results of a query with the conditions mean.
func GetScoreKind(conditions []Condition) ScoreKind {
	var hasMeaning, hasWords bool
	for _, c := range conditions {
		if !c.IsPositive {
			continue
		}
		hasMeaning = hasMeaning || c.IsMeaning
		hasWords = hasWords || !c.IsMeaning
	}
	switch {
	case hasMeaning && hasWords:
		return ScoreFused
	case hasMeaning:
		return ScoreMeaning
	case hasWords:
		return ScoreWords
	}
	return ScoreNone
}

// ChunkRange is one chunk a result is made of, in bytes of the text of its file.
type ChunkRange struct {
	ID     domain.ChunkID
	Start  int
	Length int
}

// SearchResult is one chunk found, or neighboring chunks of one file merged. A result without chunks is a
// whole file: a document matched by document conditions only. Text is an excerpt of at most
// MaxExcerptBytes around the best part, or the start of the file for a whole file. TextVersion is the text
// version the chunks were cut from, empty for the file of a note.
type SearchResult struct {
	Document     Document
	Chunks       []ChunkRange
	Start        int
	Length       int
	Selector     domain.Selector
	PlaceLabel   string
	MatchLine    int
	Text         string
	TextVersion  string
	Score        *float64
	MeaningScore *float64
	WordScore    *float64
	RerankScore  *float64
}

// MaxExcerptBytes is the most text a result carries.
const MaxExcerptBytes = 1200

// IsWholeFile reports whether the result has no place in its file.
func (r SearchResult) IsWholeFile() bool {
	return len(r.Chunks) == 0
}

// SearchResults is the value of a query: a flat list of results in the order Sort gives.
type SearchResults struct {
	Results    []SearchResult
	ScoreKind  ScoreKind
	Floor      float64
	Candidates int
	Warnings   []string
	Reranks    []RerankReport
}
