package searchquery_test

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func newMeaning(text string) searchquery.RawNode {
	return searchquery.RawNode{Operator: searchquery.OperatorLeaf, Leaf: searchquery.RawLeaf{Kind: searchquery.LeafMeaning, Text: text, Position: searchquery.StartPosition()}}
}

func newWords(text string) searchquery.RawNode {
	return searchquery.RawNode{Operator: searchquery.OperatorLeaf, Leaf: searchquery.RawLeaf{Kind: searchquery.LeafWords, Text: text, Position: searchquery.StartPosition()}}
}

func newKind(kinds ...string) searchquery.RawNode {
	return searchquery.RawNode{Operator: searchquery.OperatorLeaf, Leaf: searchquery.RawLeaf{Kind: searchquery.LeafKindIn, Kinds: kinds, Position: searchquery.StartPosition()}}
}

func newPrefix(prefix string) searchquery.RawNode {
	return searchquery.RawNode{Operator: searchquery.OperatorLeaf, Leaf: searchquery.RawLeaf{Kind: searchquery.LeafPathPrefix, Prefix: prefix, Position: searchquery.StartPosition()}}
}

func newAnd(children ...searchquery.RawNode) searchquery.RawNode {
	return searchquery.RawNode{Operator: searchquery.OperatorAnd, Children: children}
}

func newOr(children ...searchquery.RawNode) searchquery.RawNode {
	return searchquery.RawNode{Operator: searchquery.OperatorOr, Children: children}
}

func newNot(child searchquery.RawNode) searchquery.RawNode {
	return searchquery.RawNode{Operator: searchquery.OperatorNot, Children: []searchquery.RawNode{child}}
}

func normalize(t *testing.T, raw searchquery.RawNode) searchquery.Query {
	t.Helper()
	q, err := searchquery.Normalize(raw, searchquery.DefaultLimits(), time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	return q
}

func similarity(value float64) searchquery.ConditionScore {
	return searchquery.ConditionScore{Similarity: new(value)}
}

func getNote(path string) searchquery.Document {
	return searchquery.Document{Path: path, SourceKind: domain.KindNote, Kind: searchquery.KindNote}
}

func getPDF(path string) searchquery.Document {
	return searchquery.Document{Path: path, SourceKind: domain.KindFile, Kind: searchquery.KindPDF}
}

func TestEvaluateReturnsTheMinimumForAndAndTheMaximumOfMatchingChildrenForOr(t *testing.T) {
	note := getNote("a.md")
	both := normalize(t, newAnd(newMeaning("a"), newMeaning("b")))
	got := searchquery.Evaluate(both.Root, both.Conditions, []searchquery.ConditionScore{similarity(0.8), similarity(0.5)}, 0.3, note)
	if !got.IsMatch || got.MeaningScore == nil || *got.MeaningScore != 0.5 {
		t.Errorf("&& gave %+v, want a match at 0.5", got)
	}
	either := normalize(t, newOr(newMeaning("a"), newMeaning("b")))
	got = searchquery.Evaluate(either.Root, either.Conditions, []searchquery.ConditionScore{similarity(0.8), similarity(0.5)}, 0.3, note)
	if !got.IsMatch || *got.MeaningScore != 0.8 {
		t.Errorf("|| gave %+v, want a match at 0.8", got)
	}
	got = searchquery.Evaluate(either.Root, either.Conditions, []searchquery.ConditionScore{similarity(0.1), similarity(0.5)}, 0.3, note)
	if !got.IsMatch || *got.MeaningScore != 0.5 {
		t.Errorf("|| with a child below the floor gave %+v, want 0.5", got)
	}
	got = searchquery.Evaluate(both.Root, both.Conditions, []searchquery.ConditionScore{similarity(0.1), similarity(0.5)}, 0.3, note)
	if got.IsMatch {
		t.Errorf("&& with a child below the floor gave %+v, want no match", got)
	}
}

func TestANegatedMeaningDropsAChunkCloserToItThanToEveryPositiveCondition(t *testing.T) {
	note := getNote("a.md")
	q := normalize(t, newAnd(newMeaning("c"), newNot(newMeaning("x"))))
	closerToX := []searchquery.ConditionScore{similarity(0.5), similarity(0.7)}
	if searchquery.Evaluate(q.Root, q.Conditions, closerToX, 0.3, note).IsMatch {
		t.Error("a chunk above the floor and closer to x than to c matched")
	}
	closerToC := []searchquery.ConditionScore{similarity(0.8), similarity(0.7)}
	if !searchquery.Evaluate(q.Root, q.Conditions, closerToC, 0.3, note).IsMatch {
		t.Error("a chunk closer to c than to x did not match")
	}
	alone := normalize(t, newAnd(newWords("w"), newNot(newMeaning("x"))))
	reached := []searchquery.ConditionScore{{IsMatch: true}, similarity(0.3)}
	if searchquery.Evaluate(alone.Root, alone.Conditions, reached, 0.3, note).IsMatch {
		t.Error("with no positive meaning condition a chunk at the floor of x matched")
	}
	below := []searchquery.ConditionScore{{IsMatch: true}, similarity(0.29)}
	if !searchquery.Evaluate(alone.Root, alone.Conditions, below, 0.3, note).IsMatch {
		t.Error("a chunk below the floor of x was dropped")
	}
}

func TestANegatedWordsKeepsAChunkWithoutTheWords(t *testing.T) {
	note := getNote("a.md")
	q := normalize(t, newAnd(newMeaning("c"), newNot(newWords("x"))))
	if !searchquery.Evaluate(q.Root, q.Conditions, []searchquery.ConditionScore{similarity(0.8), {IsMatch: false}}, 0.3, note).IsMatch {
		t.Error("a chunk without x was dropped")
	}
	if searchquery.Evaluate(q.Root, q.Conditions, []searchquery.ConditionScore{similarity(0.8), {IsMatch: true}}, 0.3, note).IsMatch {
		t.Error("a chunk with x was kept")
	}
}

func TestANegatedDocumentLeafDropsEveryChunkOfADocumentThatSatisfiesIt(t *testing.T) {
	q := normalize(t, newAnd(newMeaning("c"), newNot(newKind("pdf"))))
	scores := []searchquery.ConditionScore{similarity(0.9)}
	if searchquery.Evaluate(q.Root, q.Conditions, scores, 0.3, getPDF("b.pdf")).IsMatch {
		t.Error("a chunk of a pdf matched !kind == pdf")
	}
	if !searchquery.Evaluate(q.Root, q.Conditions, scores, 0.3, getNote("a.md")).IsMatch {
		t.Error("a chunk of a note was dropped by !kind == pdf")
	}
}

func TestIsDocumentMatchIsTrueForADocumentThatSatisfiesTheDocumentPart(t *testing.T) {
	q := normalize(t, newOr(newKind("pdf"), newMeaning("a")))
	if !searchquery.IsDocumentMatch(q.Root, getPDF("b.pdf")) {
		t.Error("a pdf does not match kind == pdf || meaning")
	}
	if searchquery.IsDocumentMatch(q.Root, getNote("a.md")) {
		t.Error("a note matches kind == pdf || meaning with no chunk")
	}
}

func TestFuseScoresTheChunkFirstInBothRankingsOneAndAChunkFirstInOneHalf(t *testing.T) {
	results := []searchquery.SearchResult{
		{Document: getNote("a.md"), Start: 0, MeaningScore: new(0.9), WordScore: new(5.0)},
		{Document: getNote("b.md"), Start: 0, MeaningScore: new(0.8), WordScore: new(4.0)},
	}
	searchquery.SetScores(results, searchquery.ScoreFused)
	if math.Abs(*results[0].Score-1) > 1e-9 {
		t.Errorf("the chunk first in both scored %v, want 1", *results[0].Score)
	}
	half := []searchquery.SearchResult{{Document: getNote("a.md"), MeaningScore: new(0.9)}}
	searchquery.SetScores(half, searchquery.ScoreFused)
	if math.Abs(*half[0].Score-0.5) > 1e-9 {
		t.Errorf("the chunk first by meaning and absent from words scored %v, want 0.5", *half[0].Score)
	}
}

func newChunkResult(path string, start, length int, value float64) searchquery.SearchResult {
	return searchquery.SearchResult{
		Document: getNote(path), Start: start, Length: length, Text: path, Score: new(value), MeaningScore: new(value),
		Chunks: []searchquery.ChunkRange{{ID: domain.ChunkID(path), Start: start, Length: length}},
	}
}

func TestMergeJoinsChunksWithinTheGapAndKeepsApartTheOthers(t *testing.T) {
	far := newChunkResult("a.md", 1500, 100, 0.4)
	merged := searchquery.Merge([]searchquery.SearchResult{newChunkResult("a.md", 250, 100, 0.9), newChunkResult("a.md", 0, 100, 0.6), far}, 200)
	if len(merged) != 3-1 {
		t.Fatalf("merged into %d results, want 2", len(merged))
	}
	first := merged[0]
	if first.Start != 0 || first.Length != 350 || *first.Score != 0.9 {
		t.Errorf("the joined result is %+v, want 0..350 at 0.9", first)
	}
	if len(first.Chunks) != 2 || first.Chunks[0].Start != 0 || first.Chunks[1].Start != 250 {
		t.Errorf("the joined chunks are %+v, want them in order of start", first.Chunks)
	}
	apart := searchquery.Merge([]searchquery.SearchResult{newChunkResult("a.md", 0, 100, 0.6), newChunkResult("a.md", 350, 100, 0.9)}, 200)
	if len(apart) != 2 {
		t.Errorf("chunks 250 bytes apart merged into %d results", len(apart))
	}
}

func TestSortOrdersEqualRoundedScoresByPathThenStartAndWholeFilesLast(t *testing.T) {
	r := searchquery.SearchResults{Results: []searchquery.SearchResult{
		{Document: getNote("z.md")},
		newChunkResult("b.md", 10, 1, 0.8121),
		{Document: getNote("c.md")},
		newChunkResult("b.md", 0, 1, 0.8124),
		newChunkResult("a.md", 50, 1, 0.8121),
	}}
	searchquery.Sort(&r)
	want := []string{"a.md@50", "b.md@0", "b.md@10", "c.md@0", "z.md@0"}
	for i, one := range r.Results {
		if got := one.Document.Path + "@" + strconv.Itoa(one.Start); got != want[i] {
			t.Errorf("result %d is %s, want %s", i, got, want[i])
		}
	}
}

func TestGetKindNamesTheKindFromTheSourceKindAndTheMimeType(t *testing.T) {
	for _, c := range []struct {
		source domain.SourceKind
		mime   domain.ContentType
		want   string
	}{
		{domain.KindNote, "text/markdown", searchquery.KindNote},
		{domain.KindFile, "application/pdf", searchquery.KindPDF},
		{domain.KindFile, "audio/mp4", searchquery.KindAudio},
		{domain.KindFile, domain.UnknownContentType, searchquery.KindFile},
	} {
		if got := searchquery.GetKind(c.source, c.mime); got != c.want {
			t.Errorf("GetKind(%s, %s) = %s, want %s", c.source, c.mime, got, c.want)
		}
	}
}

func TestGetTitleReturnsTheNoteTitleElseTheFileNameWithoutItsLastExtension(t *testing.T) {
	if got := searchquery.GetTitle("books/Gita.v2.pdf", ""); got != "Gita.v2" {
		t.Errorf("the title of a pdf is %q", got)
	}
	if got := searchquery.GetTitle("notes/a.md", "River"); got != "River" {
		t.Errorf("the title of a note is %q", got)
	}
}

func TestGetScopesReturnsTheDocumentConditionsOfTheAndAncestorsOfEachCondition(t *testing.T) {
	q := normalize(t, newOr(newAnd(newMeaning("a"), newPrefix("x/")), newWords("b")))
	scopes := searchquery.GetScopes(q, nil)
	if len(scopes) != 2 {
		t.Fatalf("got %d scopes", len(scopes))
	}
	if scopes[0].IsEveryDocument() || len(scopes[0].PathPrefixes) != 1 || scopes[0].PathPrefixes[0] != "x/" {
		t.Errorf("the meaning scope is %+v, want startsWith x/", scopes[0])
	}
	if !scopes[0].IsDocumentMatch(getNote("x/a.md")) || scopes[0].IsDocumentMatch(getNote("y/a.md")) {
		t.Error("the meaning scope does not test the prefix")
	}
	if !scopes[1].IsEveryDocument() {
		t.Errorf("the words scope is %+v, want every document", scopes[1])
	}
}

func TestSuggestNameFindsACloseNameAndNoneForAFarOne(t *testing.T) {
	if got, ok := searchquery.SuggestName("wrods", searchquery.GetBuiltinCatalog()); !ok || got != "words" {
		t.Errorf("wrods suggests %q, %v", got, ok)
	}
	if got, ok := searchquery.SuggestName("zzz", searchquery.GetBuiltinCatalog()); ok {
		t.Errorf("zzz suggests %q", got)
	}
}

func TestNormalizeAppliesDeMorganAndFlattensGroups(t *testing.T) {
	q := normalize(t, newAnd(newMeaning("c"), newNot(newOr(newMeaning("a"), newWords("b")))))
	if q.Root.Operator != searchquery.OperatorAnd || len(q.Root.Children) != 3 {
		t.Fatalf("the root is %+v, want one && of three leaves", q.Root)
	}
	for i, child := range q.Root.Children {
		if child.Operator != searchquery.OperatorLeaf || child.IsNegated != (i > 0) {
			t.Errorf("child %d is %+v", i, child)
		}
	}
}

func TestNormalizeRejectsTheSeventeenthDistinctCondition(t *testing.T) {
	var children []searchquery.RawNode
	for i := range 16 {
		children = append(children, newWords("w"+strconv.Itoa(i+1)))
	}
	if _, err := searchquery.Normalize(newOr(children...), searchquery.DefaultLimits(), time.UTC); err != nil {
		t.Errorf("16 conditions: %v", err)
	}
	_, err := searchquery.Normalize(newOr(append(children, newWords("w17"))...), searchquery.DefaultLimits(), time.UTC)
	if parseError, ok := errors.AsType[*searchquery.ParseError](err); !ok || len(parseError.Problems) != 1 {
		t.Errorf("17 conditions: %v", err)
	}
}

func TestNormalizeCountsTheSpacesAroundAConditionTextInItsSize(t *testing.T) {
	limits := searchquery.DefaultLimits()
	padded := strings.Repeat(" ", limits.MaxConditionBytes) + "x"
	_, err := searchquery.Normalize(newWords(padded), limits, time.UTC)
	if parseError, ok := errors.AsType[*searchquery.ParseError](err); !ok || len(parseError.Problems) == 0 {
		t.Errorf("a text of %d bytes: %v", len(padded), err)
	}
}
