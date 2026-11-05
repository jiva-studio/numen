package hostsearch

import (
	"context"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"

	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
	"github.com/jiva-studio/numen/libs/core/usecase/search"
)

// recordingFinder answers a search with fixed passages and records what it was asked.
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

func TestSearchPassagesRunsWindowSearchOnVaultAndQuery(t *testing.T) {
	finder := &recordingFinder{}
	s := Search{Finds: finder}

	_, err := s.SearchPassages(t.Context(), getVault(), pluginport.PluginPassageQuery{
		Query: "  heron ", Limit: 7, Kinds: []domain.SourceKind{domain.KindFile}, PathPrefix: "library/",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(finder.params) != 1 || finder.vaults[0] != getVault().ID || finder.query[0] != "heron" {
		t.Fatalf("the search was asked %v %v", finder.vaults, finder.query)
	}
	p := finder.params[0]
	if p.Limit != 7 || p.PathPrefix != "library/" || len(p.Kinds) != 1 || p.Kinds[0] != domain.KindFile || p.Each != 3 || p.IsGrowing {
		t.Errorf("the parameters are %+v, want limit 7, the prefix, the kind, three passages of one source and a finished query", p)
	}
	if p.Lexical == 0 || p.Dense == 0 {
		t.Errorf("the parameters are %+v, want the words and the meaning both to run", p)
	}
}

func TestSearchPassagesWithoutModelUsesWordsAndSkipsBlankQuery(t *testing.T) {
	finder := &recordingFinder{found: []domain.Passage{{Source: "a.md", Text: "heron"}}}
	s := Search{Finds: finder, Embedder: nil}

	got, err := s.SearchPassages(t.Context(), getVault(), pluginport.PluginPassageQuery{Query: "heron"})
	if err != nil || len(got) != 1 {
		t.Errorf("an installation with no model answered %v, %v, want the words to answer", got, err)
	}
	blank, err := s.SearchPassages(t.Context(), getVault(), pluginport.PluginPassageQuery{Query: " \n"})
	if err != nil || len(blank) != 0 || len(finder.query) != 1 {
		t.Errorf("a blank query answered %v, %v after %d searches, want none and no search", blank, err, len(finder.query))
	}
}

func TestSearchPassagesCapsTheLimit(t *testing.T) {
	finder := &recordingFinder{}
	s := Search{Finds: finder}
	if _, err := s.SearchPassages(t.Context(), getVault(), pluginport.PluginPassageQuery{Query: "heron", Limit: 5000}); err != nil {
		t.Fatal(err)
	}
	if finder.params[0].Limit != pluginport.MaxSearchedPassages {
		t.Errorf("asked for %d passages, want the cap of %d", finder.params[0].Limit, pluginport.MaxSearchedPassages)
	}
	if _, err := s.SearchPassages(t.Context(), getVault(), pluginport.PluginPassageQuery{Query: "heron", Limit: -2}); err != nil {
		t.Fatal(err)
	}
	if finder.params[1].Limit != 0 {
		t.Errorf("a negative limit asked for %d, want the default of the search", finder.params[1].Limit)
	}
}

func TestSearchPassagesTrimsTextToWindowAroundMatch(t *testing.T) {
	body := strings.Repeat("filler words ", 400) + "the grey heron stands" + strings.Repeat(" trailing words", 400)
	matchAt := strings.Index(body, "heron")
	finder := &recordingFinder{found: []domain.Passage{
		{Source: "library/Birds.pdf", Text: body, HitAt: matchAt, Start: 4000, Length: len(body), Location: "page 12", Line: 0, Selector: domain.Selector(`{"page":12}`)},
		{Source: "notes/Short.md", Text: "a short heron note", HitAt: 8, Start: 0, Length: 18, Line: 3},
	}}
	got, err := Search{Finds: finder}.SearchPassages(t.Context(), getVault(), pluginport.PluginPassageQuery{Query: "heron"})
	if err != nil || len(got) != 2 {
		t.Fatalf("answered %v, %v", got, err)
	}
	long := got[0]
	if !strings.Contains(long.Text, "heron") || len(long.Text) >= len(body)/4 || len(long.Text) > pluginport.MaxPassageBytes {
		t.Errorf("the text holds %d bytes of %d and %q around the hit, want a window of at most %d bytes that holds the hit",
			len(long.Text), len(body), long.Text[:20], pluginport.MaxPassageBytes)
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
	matchAt := strings.Index(body, "Ачарьи")
	finder := &recordingFinder{found: []domain.Passage{{Source: "a.pdf", Text: body, HitAt: matchAt, Length: len(body)}}}
	got, err := Search{Finds: finder}.SearchPassages(t.Context(), getVault(), pluginport.PluginPassageQuery{Query: "Ачарьи"})
	if err != nil || len(got) != 1 {
		t.Fatalf("answered %v, %v", got, err)
	}
	if !strings.Contains(got[0].Text, "Ачарьи") || strings.ToValidUTF8(got[0].Text, "") != got[0].Text {
		t.Errorf("the window %q does not hold the word whole or splits a character", got[0].Text[:10])
	}
}
