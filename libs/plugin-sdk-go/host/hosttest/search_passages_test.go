package hosttest_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
)

func indexedPassages() []host.Passage {
	return []host.Passage{
		{Path: "library/Birds.pdf", Text: "The grey heron stands in the reeds", Location: "page 12", Start: 400, Length: 700},
		{Path: "library/Birds.pdf", Text: "A heron fishes at dusk", Location: "page 40", Start: 9000, Length: 500},
		{Path: "notes/Heron.md", Text: "Herons are patient", Start: 20, Length: 40, Line: 4},
		{Path: "notes/Frogs.md", Text: "Frogs sing", Start: 0, Length: 10, Line: 1},
	}
}

func TestTheSearchFindsThePassagesWithTheWordsOfTheQuery(t *testing.T) {
	search := newSearch()
	search.SetIndexedPassages(indexedPassages())

	got, err := search.SearchPassages(t.Context(), host.PassageQuery{Query: "heron"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].Path != "library/Birds.pdf" || got[0].Location != "page 12" {
		t.Errorf("found %+v, want the three passages that hold the word, in the order of the index", got)
	}
}

func TestTheSearchAppliesThePathPrefixBeforeTheLimitAndTheKinds(t *testing.T) {
	search := newSearch()
	search.SetIndexedPassages(indexedPassages())

	under, err := search.SearchPassages(t.Context(), host.PassageQuery{Query: "heron", Limit: 1, PathPrefix: "notes/"})
	if err != nil || len(under) != 1 || under[0].Path != "notes/Heron.md" {
		t.Errorf("one passage under notes/ is %+v, %v", under, err)
	}
	files, err := search.SearchPassages(t.Context(), host.PassageQuery{Query: "heron", Kinds: []domain.SourceKind{domain.KindFile}})
	if err != nil || len(files) != 2 || files[0].Path != "library/Birds.pdf" {
		t.Errorf("the files that answer are %+v, %v", files, err)
	}
	notes, err := search.SearchPassages(t.Context(), host.PassageQuery{Query: "heron", Kinds: []domain.SourceKind{domain.KindNote}})
	if err != nil || len(notes) != 1 || notes[0].Path != "notes/Heron.md" {
		t.Errorf("the notes that answer are %+v, %v", notes, err)
	}
}

func TestTheSearchOfPassagesNeedsNoModelButNeedsThePermissionAndAVault(t *testing.T) {
	search := hosttest.NewSearch("physics")
	search.SetIndexedPassages(indexedPassages())
	if got, err := search.SearchPassages(t.Context(), host.PassageQuery{Query: "heron"}); err != nil || len(got) != 3 {
		t.Errorf("a host with no model found %d, %v, want the words alone to answer", len(got), err)
	}
	if got, err := search.SearchPassages(t.Context(), host.PassageQuery{Query: "  "}); err != nil || len(got) != 0 {
		t.Errorf("a blank query found %d, %v", len(got), err)
	}

	search.Deny(host.PermissionIndexRead)
	if _, err := search.SearchPassages(t.Context(), host.PassageQuery{Query: "heron"}); !errors.Is(err, host.ErrPermissionDenied) {
		t.Errorf("without the permission it gave %v", err)
	}
	hidden := hosttest.NewSearch("physics")
	hidden.HideVault()
	if _, err := hidden.SearchPassages(t.Context(), host.PassageQuery{Query: "heron"}); !errors.Is(err, host.ErrNoVault) {
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
