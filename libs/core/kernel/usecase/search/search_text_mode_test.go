package search_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/chunk"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
)

func TestSearchTextReturnsOnlyResultsTheWordsMatchedInWordsMode(t *testing.T) {
	c := newCorpus(t)
	c.vectorize(t, c.first, newVector(+1))
	searchText := search.NewSearchText(c.search(oneWay{direction: newVector(+1)}))

	hybrid, err := searchText.Execute(t.Context(), c.first, search.SearchResultQuery{Query: "zzqqxx"})
	if err != nil || len(hybrid) == 0 {
		t.Fatalf("a hybrid search found %v, %v, want the notes the meaning search reaches", sources(hybrid), err)
	}

	unsaid, err := searchText.Execute(t.Context(), c.first, search.SearchResultQuery{Query: "zzqqxx", Mode: search.Lexical})
	if err != nil || len(unsaid) != 0 {
		t.Errorf("words mode found %v, %v for a word no note has, want nothing", sources(unsaid), err)
	}
	said, err := searchText.Execute(t.Context(), c.first, search.SearchResultQuery{Query: "disorder", Mode: search.Lexical})
	if err != nil || len(said) != 1 || !strings.Contains(said[0].Path, "Entropy") {
		t.Fatalf("words mode found %v, %v, want the note with the word", sources(said), err)
	}
	if said[0].Similarity != nil {
		t.Errorf("words mode returned the similarity %v, want none", *said[0].Similarity)
	}
}

func TestSearchTextReturnsOnlyResultsTheMeaningSearchFoundInMeaningMode(t *testing.T) {
	c := newCorpus(t)
	c.cut(t, c.first, "notes/Entropy.md", "a measure of disorder, shared")
	c.cut(t, c.first, "notes/Heat.md", "energy in transit, shared")
	c.vectorizePath(t, c.first, "notes/Entropy.md", func(chunk.Record) []float32 { return newVector(+1) })
	searchText := search.NewSearchText(c.search(oneWay{direction: newVector(+1)}))

	found, err := searchText.Execute(t.Context(), c.first, search.SearchResultQuery{Query: "shared", Mode: search.Sources})
	if err != nil {
		t.Fatal(err)
	}
	if got := sources(found); !slices.Equal(got, []string{"notes/Entropy.md"}) {
		t.Errorf("meaning mode found %v, want only the note with a vector", got)
	}
	for _, p := range found {
		if p.WordScore != nil {
			t.Errorf("meaning mode returned the word score %v, want none", *p.WordScore)
		}
	}
}

func TestSearchTextFusesWordsNamesAndMeaningWhenTheQueryNamesNoMode(t *testing.T) {
	finder := &recordingFinder{}

	if _, err := search.NewSearchText(finder).Execute(t.Context(), getMarshVault(), search.SearchResultQuery{Query: "heron", Limit: 7}); err != nil {
		t.Fatal(err)
	}

	p := finder.params[0]
	if p.Lexical == 0 || p.Named == 0 || p.Sources == 0 || p.IsRawWords {
		t.Errorf("a query with no mode requested %+v, want words, names and meaning, with the words quoted", p)
	}
}

func TestSearchTextPassesTheRawWordsFlagInWordsMode(t *testing.T) {
	finder := &recordingFinder{}

	query := search.SearchResultQuery{Query: "NEAR(heron reed, 3)", Mode: search.Lexical, IsRawWords: true}
	if _, err := search.NewSearchText(finder).Execute(t.Context(), getMarshVault(), query); err != nil {
		t.Fatal(err)
	}

	p := finder.params[0]
	if !p.IsRawWords || p.Lexical == 0 || p.Named != 0 || p.Sources != 0 || finder.query[0] != "NEAR(heron reed, 3)" {
		t.Errorf("a raw words query requested %q with %+v, want the words alone, raw, with the query as written", finder.query[0], p)
	}
}

func TestSearchTextFindsAChunkByARawNearQueryAndNotOneWhoseWordsAreTooFarApart(t *testing.T) {
	c := newCorpus(t)
	v := c.addVault(t, map[string]string{
		"notes/Close.md":   "The heron waits by the reed at dawn.\n",
		"notes/Distant.md": "The heron " + strings.Repeat("walks along the bank ", 6) + "and finds a reed.\n",
	})
	searchText := search.NewSearchText(c.search(nil))

	quoted, err := searchText.Execute(t.Context(), v, search.SearchResultQuery{Query: "heron reed", Mode: search.Lexical})
	if err != nil || len(quoted) != 2 {
		t.Fatalf("both words found %v, %v, want both notes", sources(quoted), err)
	}

	near, err := searchText.Execute(t.Context(), v, search.SearchResultQuery{Query: "NEAR(heron reed, 3)", Mode: search.Lexical, IsRawWords: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := sources(near); !slices.Equal(got, []string{"notes/Close.md"}) {
		t.Errorf("a raw NEAR query found %v, want only the note whose words are within three of each other", got)
	}
}

func TestSearchTextQuotesTheWordsOfAQueryThatIsNotRaw(t *testing.T) {
	c := newCorpus(t)
	v := c.addVault(t, map[string]string{
		"notes/Close.md": "The heron waits by the reed at dawn.\n",
	})

	found, err := search.NewSearchText(c.search(nil)).Execute(t.Context(), v, search.SearchResultQuery{Query: "NEAR(heron reed, 3)", Mode: search.Lexical})
	if err != nil {
		t.Fatalf("a query with FTS5 syntax that is not raw failed: %v", err)
	}
	if len(found) != 0 {
		t.Errorf("a query that is not raw found %v, want nothing, since no note has the word near", sources(found))
	}
}
