package searchquery

import (
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

type evaluator struct {
	req         *searchquery.Evaluation
	getDocument func(path string, sourceKind domain.SourceKind) searchquery.Document
}

func (e evaluator) evaluate(
	candidates []domain.SearchResult, scores [][]searchquery.ConditionScore, texts map[string]fileText,
	documents []searchquery.Document, kind searchquery.ScoreKind,
) []searchquery.SearchResult {
	var matched []searchquery.SearchResult
	for i, c := range candidates {
		d := e.getDocument(c.Path, c.Kind)
		o := searchquery.Evaluate(e.req.Query.Root, e.req.Query.Conditions, scores[i], e.req.Floor, d)
		if !o.IsMatch {
			continue
		}
		matched = append(matched, newChunkResult(d, c, o, texts[c.Path]))
	}
	searchquery.SetScores(matched, kind)
	results := e.mergeFiles(matched)
	hasChunks := map[string]bool{}
	for _, r := range results {
		hasChunks[r.Document.Path] = true
	}
	for _, d := range documents {
		if !hasChunks[d.Path] {
			results = append(results, searchquery.SearchResult{Document: d, Text: getExcerpt(texts[d.Path].prose, 0, len(texts[d.Path].prose))})
		}
	}
	return results
}

func (e evaluator) mergeFiles(matched []searchquery.SearchResult) []searchquery.SearchResult {
	byPath := map[string][]searchquery.SearchResult{}
	var paths []string
	for _, r := range matched {
		if _, found := byPath[r.Document.Path]; !found {
			paths = append(paths, r.Document.Path)
		}
		byPath[r.Document.Path] = append(byPath[r.Document.Path], r)
	}
	slices.Sort(paths)
	var merged []searchquery.SearchResult
	for _, path := range paths {
		merged = append(merged, searchquery.Merge(byPath[path], e.req.Limits.MergeGapBytes)...)
	}
	return merged
}

func newChunkResult(d searchquery.Document, c domain.SearchResult, o searchquery.Outcome, text fileText) searchquery.SearchResult {
	r := searchquery.SearchResult{
		Document: d, Chunks: []searchquery.ChunkRange{{ID: c.ChunkID, Start: c.Start, Length: c.Length}},
		Start: c.Start, Length: c.Length, PlaceLabel: c.PlaceLabel, TextVersion: c.VersionID,
		MeaningScore: o.MeaningScore, WordScore: o.WordScore,
	}
	if text.isRead {
		r.Text = getExcerpt(text.prose, c.Start, c.Length)
		r.MatchLine = getLine(text.prose, c.Start+c.MatchOffset)
		if text.version != nil {
			r.Selector = text.version.GetSelector(c.Start + c.MatchOffset)
		}
	}
	return r
}
