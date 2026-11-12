package query_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/note/domain"
)

type headings map[string][]domain.Heading

func (h headings) GetHeadings(_ context.Context, _ domain.VaultID, paths []string) (map[string][]domain.Heading, error) {
	out := map[string][]domain.Heading{}
	for _, path := range paths {
		if found, ok := h[path]; ok {
			out[path] = found
		}
	}
	return out, nil
}

type brokenHeadings struct{}

func (brokenHeadings) GetHeadings(context.Context, domain.VaultID, []string) (map[string][]domain.Heading, error) {
	return nil, errors.New("index closed")
}

func TestHeadingsAreListedInTheOrderOfThePathsAndLeaveOutWhatNamesNoNote(t *testing.T) {
	index := headings{
		"Heat.md":    {{Text: "Work", Line: 3, Level: 2}},
		"Entropy.md": {{Text: "Entropy", Line: 1, Level: 1}, {Text: "Order", Line: 8, Level: 2}},
	}
	got, err := query.NewListHeadings(index).Execute(t.Context(), domain.Vault{ID: "v"}, []string{"Heat.md", "Missing.md", "Entropy.md"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Path != "Heat.md" || got[1].Path != "Entropy.md" || len(got[1].Headings) != 2 {
		t.Errorf("listed %+v, want Heat.md then Entropy.md", got)
	}
}

func TestAListingThatTheIndexCannotAnswerIsAnError(t *testing.T) {
	if _, err := query.NewListHeadings(brokenHeadings{}).Execute(t.Context(), domain.Vault{ID: "v"}, []string{"Heat.md"}); err == nil {
		t.Error("a closed index answered")
	}
}
