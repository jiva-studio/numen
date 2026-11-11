package hosttest_test

import (
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
)

func getIndexedPassages() []host.Passage {
	return []host.Passage{
		{Path: "library/Birds.pdf", Text: "The gray heron stands in the reeds", Location: "page 12", Start: 400, Length: 700},
		{Path: "library/Birds.pdf", Text: "A heron fishes at dusk", Location: "page 40", Start: 9000, Length: 500},
		{Path: "notes/Heron.md", Text: "A heron is patient", Start: 20, Length: 40, Line: 4},
		{Path: "notes/Frogs.md", Text: "Frogs sing", Start: 0, Length: 10, Line: 1},
	}
}

func TestTheSearchFindsThePassagesWithTheWordsOfTheQuery(t *testing.T) {
	search := newSearch()
	search.SetIndexedPassages(getIndexedPassages())

	got, err := search.SearchPassages(t.Context(), host.PassageQuery{Query: "heron"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].Path != "library/Birds.pdf" || got[0].Location != "page 12" {
		t.Errorf("found %+v, want the three passages that contain the word, in the order of the index", got)
	}
}

func TestTheSearchAppliesThePathPrefixBeforeTheLimitAndTheKinds(t *testing.T) {
	search := newSearch()
	search.SetIndexedPassages(getIndexedPassages())

	withinPrefix, err := search.SearchPassages(t.Context(), host.PassageQuery{Query: "heron", Limit: 1, PathPrefix: "notes/"})
	if err != nil || len(withinPrefix) != 1 || withinPrefix[0].Path != "notes/Heron.md" {
		t.Errorf("one passage under notes/ is %+v, %v", withinPrefix, err)
	}
	files, err := search.SearchPassages(t.Context(), host.PassageQuery{Query: "heron", Kinds: []domain.SourceKind{domain.KindFile}})
	if err != nil || len(files) != 2 || files[0].Path != "library/Birds.pdf" {
		t.Errorf("the files that match are %+v, %v", files, err)
	}
	notes, err := search.SearchPassages(t.Context(), host.PassageQuery{Query: "heron", Kinds: []domain.SourceKind{domain.KindNote}})
	if err != nil || len(notes) != 1 || notes[0].Path != "notes/Heron.md" {
		t.Errorf("the notes that match are %+v, %v", notes, err)
	}
}

func TestTheSearchMatchesThePathPrefixInEitherNormalForm(t *testing.T) {
	composed, decomposed := "B\u00fccher", "Bu\u0308cher"
	search := newSearch()
	search.SetIndexedPassages([]host.Passage{
		{Path: composed + "/Teebuch.pdf", Text: "black tea from the hills", Start: 0, Length: 24},
		{Path: decomposed + "/Kaffee.pdf", Text: "dark tea and coffee", Start: 0, Length: 19},
		{Path: "library/Tea.pdf", Text: "green tea leaves", Start: 0, Length: 16},
	})

	for _, prefix := range []string{composed + "/", decomposed + "/"} {
		got, err := search.SearchPassages(t.Context(), host.PassageQuery{Query: "tea", PathPrefix: prefix})
		if err != nil || len(got) != 2 || got[0].Path != composed+"/Teebuch.pdf" || got[1].Path != decomposed+"/Kaffee.pdf" {
			t.Errorf("the passages under %+q are %+v, %v, want the two of the shelf", prefix, got, err)
		}
	}
}
func TestSearchPassagesNeedsPermissionAndVaultButNoModel(t *testing.T) {
	search := hosttest.NewSearch("physics")
	search.SetIndexedPassages(getIndexedPassages())
	if got, err := search.SearchPassages(t.Context(), host.PassageQuery{Query: "heron"}); err != nil || len(got) != 3 {
		t.Errorf("a host with no model found %d, %v, want the words alone to match", len(got), err)
	}
	if got, err := search.SearchPassages(t.Context(), host.PassageQuery{Query: "  "}); err != nil || len(got) != 0 {
		t.Errorf("a blank query found %d, %v", len(got), err)
	}

	search.Deny(host.PermissionIndexRead)
	if _, err := search.SearchPassages(t.Context(), host.PassageQuery{Query: "heron"}); !errors.Is(err, host.ErrPermissionDenied) {
		t.Errorf("without the permission it gave %v", err)
	}
	noVault := hosttest.NewSearch("physics")
	noVault.HideVault()
	if _, err := noVault.SearchPassages(t.Context(), host.PassageQuery{Query: "heron"}); !errors.Is(err, host.ErrNoVault) {
		t.Errorf("with no vault shown it gave %v", err)
	}
	if _, err := hosttest.NewSearch("physics").With("nowhere").SearchPassages(t.Context(), host.PassageQuery{Query: "heron"}); !errors.Is(err, host.ErrUnknownVault) {
		t.Errorf("an unknown vault gave %v", err)
	}
}

func TestTheSearchKeepsAtMostThreePassagesOfOneSource(t *testing.T) {
	search := newSearch()
	var many []host.Passage
	for i := range 5 {
		many = append(many, host.Passage{Path: "library/Birds.pdf", Text: "heron", Start: i * 10})
	}
	search.SetIndexedPassages(many)
	got, err := search.SearchPassages(t.Context(), host.PassageQuery{Query: "heron"})
	if err != nil || len(got) != host.MaxSearchedPassagesPerSource {
		t.Errorf("found %d passages of one source, %v, want %d", len(got), err, host.MaxSearchedPassagesPerSource)
	}
	if !slices.IsSortedFunc(got, func(a, b host.Passage) int { return a.Start - b.Start }) {
		t.Errorf("the passages are not in the order of the index: %+v", got)
	}
}

func TestTheSearchMatchesEveryWordOfTheQueryAsAWholeWord(t *testing.T) {
	search := newSearch()
	search.SetIndexedPassages(getIndexedPassages())

	both, err := search.SearchPassages(t.Context(), host.PassageQuery{Query: "Gray HERON"})
	if err != nil || len(both) != 1 || both[0].Location != "page 12" {
		t.Errorf("gray heron found %+v, %v, want only the passage that contains both words", both, err)
	}
	part, err := search.SearchPassages(t.Context(), host.PassageQuery{Query: "her"})
	if err != nil || len(part) != 0 {
		t.Errorf("a part of a word found %+v, %v, want nothing", part, err)
	}
	punctuated, err := search.SearchPassages(t.Context(), host.PassageQuery{Query: "reeds"})
	if err != nil || len(punctuated) != 1 {
		t.Errorf("a word at the end of the text found %+v, %v", punctuated, err)
	}
}

func TestSearchPassagesAppliesDefaultAndMaximumLimit(t *testing.T) {
	search := newSearch()
	var many []host.Passage
	for i := range host.MaxSearchedPassages + 10 {
		many = append(many, host.Passage{Path: fmt.Sprintf("notes/Heron %03d.md", i), Text: "heron", Line: 1})
	}
	search.SetIndexedPassages(many)

	byDefault, err := search.SearchPassages(t.Context(), host.PassageQuery{Query: "heron"})
	if err != nil || len(byDefault) != host.DefaultSearchedPassages {
		t.Errorf("a zero limit found %d, %v, want %d", len(byDefault), err, host.DefaultSearchedPassages)
	}
	capped, err := search.SearchPassages(t.Context(), host.PassageQuery{Query: "heron", Limit: host.MaxSearchedPassages * 2})
	if err != nil || len(capped) != host.MaxSearchedPassages {
		t.Errorf("a limit above the most found %d, %v, want %d", len(capped), err, host.MaxSearchedPassages)
	}
}
