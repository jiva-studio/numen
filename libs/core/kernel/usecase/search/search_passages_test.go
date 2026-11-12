package search_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// recordingFinder returns a search with fixed passages and records the requests.
type recordingFinder struct {
	found  []domain.Passage
	vaults []domain.VaultID
	query  []string
	params []search.Parameters
}

func (f *recordingFinder) Execute(_ context.Context, v domain.Vault, query string, p search.Parameters) ([]domain.Passage, error) {
	f.vaults = append(f.vaults, v.ID)
	f.query = append(f.query, query)
	f.params = append(f.params, p)
	return f.found, nil
}

func getMarshVault() domain.Vault { return domain.Vault{ID: "v-marsh", Name: "Marsh"} }

func TestSearchPassagesRunsKeywordAndEmbeddingWithThreePerSource(t *testing.T) {
	finder := &recordingFinder{}

	_, err := search.NewSearchPassages(finder).Execute(t.Context(), getMarshVault(), search.PassageQuery{
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
	if p.Lexical == 0 || p.Dense == 0 {
		t.Errorf("the parameters are %+v, want the words and the meaning both to run", p)
	}
}

func TestAnEmptyQueryFindsNoPassagesAndRunsNoSearch(t *testing.T) {
	finder := &recordingFinder{found: []domain.Passage{{Source: "notes/Heron.md", Text: "heron"}}}
	passages := search.NewSearchPassages(finder)

	for _, query := range []string{"", " \n\t"} {
		got, err := passages.Execute(t.Context(), getMarshVault(), search.PassageQuery{Query: query, Limit: 5})
		if err != nil || len(got) != 0 {
			t.Errorf("the query %q found %v, %v, want nothing", query, got, err)
		}
	}
	if len(finder.query) != 0 {
		t.Errorf("an empty query ran the search %d times, want none", len(finder.query))
	}
}

func TestTheLimitOfASearchOfPassagesIsClamped(t *testing.T) {
	finder := &recordingFinder{}
	passages := search.NewSearchPassages(finder)
	for _, limit := range []int{5000, -2} {
		if _, err := passages.Execute(t.Context(), getMarshVault(), search.PassageQuery{Query: "heron", Limit: limit}); err != nil {
			t.Fatal(err)
		}
	}

	if finder.params[0].Limit != search.MaxSearchedPassages {
		t.Errorf("a limit of 5000 requested %d passages, want the most a search returns, %d", finder.params[0].Limit, search.MaxSearchedPassages)
	}
	if want := search.GetTypingParameters(search.Hybrid, 0).Limit; finder.params[1].Limit != want {
		t.Errorf("a negative limit requested %d passages, want the default of the search, %d", finder.params[1].Limit, want)
	}
}

func TestSearchPassagesTrimsTextAroundTheHit(t *testing.T) {
	body := strings.Repeat("filler words ", 400) + "the gray heron stands" + strings.Repeat(" trailing words", 400)
	hitAt := strings.Index(body, "heron")
	finder := &recordingFinder{found: []domain.Passage{
		{Source: "library/Birds.pdf", Text: body, HitAt: hitAt, Start: 4000, Length: len(body), Location: "page 12", Selector: domain.Selector(`{"page":12}`)},
		{Source: "notes/Short.md", Text: "a short heron note", HitAt: 8, Start: 0, Length: 18, Line: 3},
	}}

	got, err := search.NewSearchPassages(finder).Execute(t.Context(), getMarshVault(), search.PassageQuery{Query: "heron"})
	if err != nil || len(got) != 2 {
		t.Fatalf("found %v, %v, want two passages", got, err)
	}
	long := got[0]
	if !strings.Contains(long.Text, "heron") || len(long.Text) > search.MaxPassageBytes || !strings.Contains(body, long.Text) {
		t.Errorf("the text has %d bytes of %d, want a part of at most %d bytes that contains the hit",
			len(long.Text), len(body), search.MaxPassageBytes)
	}
	if before := strings.Index(long.Text, "heron"); before < search.MaxPassageBytes/4-len("the gray ") || before > search.MaxPassageBytes/4 {
		t.Errorf("the hit is %d bytes into the window, want about a quarter of %d", before, search.MaxPassageBytes)
	}
	if long.Start != 4000 || long.Length != len(body) || long.Location != "page 12" || string(long.Selector) != `{"page":12}` {
		t.Errorf("the range and place changed: %+v", long)
	}
	if got[1].Text != "a short heron note" || got[1].Line != 3 {
		t.Errorf("a short text was cut: %+v", got[1])
	}
}

func TestTheWindowAroundAHitNeverSplitsACharacter(t *testing.T) {
	body := strings.Repeat("я", 3000) + "Ачарьи" + strings.Repeat("ю", 3000)
	hitAt := strings.Index(body, "Ачарьи")
	finder := &recordingFinder{found: []domain.Passage{{Source: "library/Talks.pdf", Text: body, HitAt: hitAt, Length: len(body)}}}

	got, err := search.NewSearchPassages(finder).Execute(t.Context(), getMarshVault(), search.PassageQuery{Query: "Ачарьи"})
	if err != nil || len(got) != 1 {
		t.Fatalf("found %v, %v, want one passage", got, err)
	}
	if !strings.Contains(got[0].Text, "Ачарьи") || strings.ToValidUTF8(got[0].Text, "") != got[0].Text {
		t.Errorf("the window %q does not contain the whole word or splits a character", got[0].Text[:10])
	}
}
