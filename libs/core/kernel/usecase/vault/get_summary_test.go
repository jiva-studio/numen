package vault_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/none"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func TestGetSummaryCountsWhatTheIndexContains(t *testing.T) {
	t.Parallel()
	_, db, v := newRefresh(t, map[string]string{"A.md": "# A\n\n## One\n", "B.md": "# B\n"})

	got, err := registry.NewGetSummary(db.NoteQueries()).Execute(t.Context(), v)
	if err != nil {
		t.Fatal(err)
	}
	if got.Notes != 2 {
		t.Errorf("the summary counts %d notes, want 2", got.Notes)
	}
}

func TestGetSummaryAnswersNothingWhereNoIndexIsBound(t *testing.T) {
	t.Parallel()
	if got, err := registry.NewGetSummary(none.NoteQueries{}).Execute(t.Context(), domain.Vault{}); err != nil || got != (domain.VaultSummary{}) {
		t.Errorf("an unbound index answered %+v, %v", got, err)
	}
	var unmade *registry.GetSummary
	if got, err := unmade.Execute(t.Context(), domain.Vault{}); err != nil || got != (domain.VaultSummary{}) {
		t.Errorf("a use case not made answered %+v, %v", got, err)
	}
}
