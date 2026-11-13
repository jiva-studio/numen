package search

import (
	"context"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
)

type shownVault struct{ v domain.Vault }

func (s shownVault) GetCurrentVault() domain.Vault { return s.v }

// names is the names of a vault, found by what they contain.
type names []domain.NameMatch

func (n names) GetNameMatches(_ context.Context, _ domain.VaultID, typed string, limit int) ([]domain.NameMatch, error) {
	var found []domain.NameMatch
	for _, one := range n {
		if strings.Contains(strings.ToLower(one.Path), strings.ToLower(typed)) && len(found) < limit {
			found = append(found, one)
		}
	}
	return found, nil
}

// types is the type of each note of a vault.
type types map[string]domain.NoteType

func (t types) GetTypes(_ context.Context, _ domain.VaultID, paths []string) (map[string]domain.NoteType, error) {
	out := map[string]domain.NoteType{}
	for _, path := range paths {
		if kind, ok := t[path]; ok {
			out[path] = kind
		}
	}
	return out, nil
}

// titles is the notes of a vault.
type titles map[string]domain.NoteRef

func (t titles) GetNotes(_ context.Context, _ domain.VaultID, paths []string) (map[string]domain.NoteRef, error) {
	out := map[string]domain.NoteRef{}
	for _, path := range paths {
		if ref, ok := t[path]; ok {
			out[path] = ref
		}
	}
	return out, nil
}

// tabKinds is the kind of tab each note opens in, by path.
type tabKinds map[string]string

func (k tabKinds) Execute(_ context.Context, _ domain.Vault, paths []string) (map[string]string, error) {
	out := map[string]string{}
	for _, path := range paths {
		if kind, ok := k[path]; ok {
			out[path] = kind
		}
	}
	return out, nil
}

func newHandler(vault domain.Vault, found names, kinds tabKinds) *Handler {
	return NewHandler(
		shownVault{v: vault},
		search.NewFindNames(found, kinds),
		search.NewFindPassages(search.Search{}, titles{}, types{}, kinds),
		testsupport.NewClock(),
	)
}

func TestSearchNamesAnswersTheTitleAndTheHeadingWithWhereTheyMatched(t *testing.T) {
	handler := newHandler(domain.Vault{ID: "v1"}, names{
		{Path: "Entropy.md", Title: "Entropy", Ranges: []domain.UnitRange{{Start: 0, End: 5}}},
		{Path: "Heat.md", Title: "Heat", Type: "sample", Heading: "Heat and work", Line: 6},
	}, tabKinds{"Entropy.md": "reader", "Heat.md": "reader"})

	answer, err := handler.SearchNames(t.Context(), connect.NewRequest(&v1.SearchNamesRequest{Query: ".md"}))

	if err != nil {
		t.Fatal(err)
	}
	found := answer.Msg.GetFound()
	if len(found) != 2 {
		t.Fatalf("%d answers, want 2: %v", len(found), found)
	}
	if found[0].GetNote().GetTitle() != "Entropy" || len(found[0].GetRanges()) != 1 || found[0].GetHeading() != nil || found[0].GetType() != "" {
		t.Errorf("the title is answered as %v", found[0])
	}
	if found[1].GetHeading().GetText() != "Heat and work" || found[1].GetHeading().GetLine() != 6 || found[1].GetType() != "sample" ||
		found[1].GetTabKind() != "reader" {
		t.Errorf("the heading is answered as %v", found[1])
	}
}

func TestSearchNamesInAWindowShowingNoVaultFindsNothing(t *testing.T) {
	handler := newHandler(domain.Vault{}, names{{Path: "Entropy.md", Title: "Entropy"}}, nil)

	answer, err := handler.SearchNames(t.Context(), connect.NewRequest(&v1.SearchNamesRequest{Query: "entropy"}))

	if err != nil || len(answer.Msg.GetFound()) != 0 {
		t.Fatalf("answered %v (%v)", answer.Msg, err)
	}
}

func TestSearchNamesAsksForAtMostTheLimitOfTheWindow(t *testing.T) {
	var all names
	for i := range 30 {
		all = append(all, domain.NameMatch{Path: "note-" + string(rune('a'+i)) + ".md"})
	}
	handler := newHandler(domain.Vault{ID: "v1"}, all, nil)

	byDefault, err := handler.SearchNames(t.Context(), connect.NewRequest(&v1.SearchNamesRequest{Query: "note"}))
	if err != nil {
		t.Fatal(err)
	}
	asked, err := handler.SearchNames(t.Context(), connect.NewRequest(&v1.SearchNamesRequest{Query: "note", Limit: 3}))
	if err != nil {
		t.Fatal(err)
	}

	if len(byDefault.Msg.GetFound()) != search.MostFound || len(asked.Msg.GetFound()) != 3 {
		t.Errorf("answered %d by default and %d when asked for 3", len(byDefault.Msg.GetFound()), len(asked.Msg.GetFound()))
	}
}

func TestSearchPassagesWithoutAModeIsRefused(t *testing.T) {
	handler := newHandler(domain.Vault{ID: "v1"}, nil, nil)

	_, err := handler.SearchPassages(t.Context(), connect.NewRequest(&v1.SearchPassagesRequest{Query: "engine"}))

	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("answered %v", err)
	}
}
