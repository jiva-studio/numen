package search_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// recordingFinder returns a search with fixed search results and records the requests.
type recordingFinder struct {
	found  []domain.SearchResult
	vaults []domain.VaultID
	query  []string
	params []search.Parameters
}

func (f *recordingFinder) Execute(_ context.Context, v domain.Vault, query string, p search.Parameters) ([]domain.SearchResult, error) {
	f.vaults = append(f.vaults, v.ID)
	f.query = append(f.query, query)
	f.params = append(f.params, p)
	return f.found, nil
}

func getMarshVault() domain.Vault { return domain.Vault{ID: "v-marsh", Name: "Marsh"} }

func TestSearchTextRunsKeywordAndEmbeddingWithThreePerSource(t *testing.T) {
	finder := &recordingFinder{}

	_, err := search.NewSearchText(finder).Execute(t.Context(), getMarshVault(), search.SearchResultQuery{
		Query: "  heron ", Limit: 7, Kinds: []domain.SourceKind{domain.KindFile}, PathPrefix: "library/",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(finder.params) != 1 || finder.vaults[0] != getMarshVault().ID || finder.query[0] != "heron" {
		t.Fatalf("the search received %v %v, want the trimmed query once on the vault", finder.vaults, finder.query)
	}
	p := finder.params[0]
	if p.Limit != 7 || p.PathPrefix != "library/" || len(p.Kinds) != 1 || p.Kinds[0] != domain.KindFile || p.Each != 3 || p.IsGrowing {
		t.Errorf("the parameters are %+v, want limit 7, the prefix, the kind, three passages of one source and a finished query", p)
	}
	if p.Lexical == 0 || p.Sources == 0 {
		t.Errorf("the parameters are %+v, want the words and the meaning both to run", p)
	}
}

func TestAnEmptyQueryFindsNoResultsAndRunsNoSearch(t *testing.T) {
	finder := &recordingFinder{found: []domain.SearchResult{{Path: "notes/Heron.md", Text: "heron"}}}
	passages := search.NewSearchText(finder)

	for _, query := range []string{"", " \n\t"} {
		got, err := passages.Execute(t.Context(), getMarshVault(), search.SearchResultQuery{Query: query, Limit: 5})
		if err != nil || len(got) != 0 {
			t.Errorf("the query %q found %v, %v, want nothing", query, got, err)
		}
	}
	if len(finder.query) != 0 {
		t.Errorf("an empty query ran the search %d times, want none", len(finder.query))
	}
}

func TestTheLimitOfASearchOfResultsIsClamped(t *testing.T) {
	finder := &recordingFinder{}
	passages := search.NewSearchText(finder)
	for _, limit := range []int{5000, -2} {
		if _, err := passages.Execute(t.Context(), getMarshVault(), search.SearchResultQuery{Query: "heron", Limit: limit}); err != nil {
			t.Fatal(err)
		}
	}

	if finder.params[0].Limit != search.MaxSearchedResults {
		t.Errorf("a limit of 5000 requested %d passages, want the most a search returns, %d", finder.params[0].Limit, search.MaxSearchedResults)
	}
	if want := search.GetTypingParameters(search.Hybrid, 0).Limit; finder.params[1].Limit != want {
		t.Errorf("a negative limit requested %d passages, want the default of the search, %d", finder.params[1].Limit, want)
	}
}

func TestSearchTextTrimsTextAroundTheHit(t *testing.T) {
	body := strings.Repeat("filler words ", 400) + "the gray heron stands" + strings.Repeat(" trailing words", 400)
	matchOffset := strings.Index(body, "heron")
	finder := &recordingFinder{found: []domain.SearchResult{
		{Path: "library/Birds.pdf", Text: body, MatchOffset: matchOffset, Start: 4000, Length: len(body), PlaceLabel: "page 12", Selector: domain.Selector(`{"page":12}`)},
		{Path: "notes/Short.md", Text: "a short heron note", MatchOffset: 8, Start: 0, Length: 18, MatchLine: 3},
	}}

	got, err := search.NewSearchText(finder).Execute(t.Context(), getMarshVault(), search.SearchResultQuery{Query: "heron"})
	if err != nil || len(got) != 2 {
		t.Fatalf("found %v, %v, want two passages", got, err)
	}
	long := got[0]
	if !strings.Contains(long.Text, "heron") || len(long.Text) > search.MaxExcerptBytes || !strings.Contains(body, long.Text) {
		t.Errorf("the text has %d bytes of %d, want a part of at most %d bytes that contains the hit",
			len(long.Text), len(body), search.MaxExcerptBytes)
	}
	if before := strings.Index(long.Text, "heron"); before < search.MaxExcerptBytes/4-len("the gray ") || before > search.MaxExcerptBytes/4 {
		t.Errorf("the hit is %d bytes into the window, want about a quarter of %d", before, search.MaxExcerptBytes)
	}
	if long.Start != 4000 || long.Length != len(body) || long.PlaceLabel != "page 12" || string(long.Selector) != `{"page":12}` {
		t.Errorf("the range and place changed: %+v", long)
	}
	if got[1].Text != "a short heron note" || got[1].MatchLine != 3 {
		t.Errorf("a short text was cut: %+v", got[1])
	}
}

func TestTheWindowAroundAHitNeverSplitsACharacter(t *testing.T) {
	body := strings.Repeat("я", 3000) + "Ачарьи" + strings.Repeat("ю", 3000)
	matchOffset := strings.Index(body, "Ачарьи")
	finder := &recordingFinder{found: []domain.SearchResult{{Path: "library/Talks.pdf", Text: body, MatchOffset: matchOffset, Length: len(body)}}}

	got, err := search.NewSearchText(finder).Execute(t.Context(), getMarshVault(), search.SearchResultQuery{Query: "Ачарьи"})
	if err != nil || len(got) != 1 {
		t.Fatalf("found %v, %v, want one passage", got, err)
	}
	if !strings.Contains(got[0].Text, "Ачарьи") || strings.ToValidUTF8(got[0].Text, "") != got[0].Text {
		t.Errorf("the window %q does not contain the whole word or splits a character", got[0].Text[:10])
	}
}
