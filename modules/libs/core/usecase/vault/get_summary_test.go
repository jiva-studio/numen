package vault_test

import (
	"testing"

	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

func TestGetSummaryCountsWhatTheIndexContains(t *testing.T) {
	t.Parallel()
	_, db, v := newRefresh(t, map[string]string{"A.md": "# A\n\n## One\n", "B.md": "# B\n"})

	got, err := vaults.NewGetSummary(db.Queries()).Execute(t.Context(), v)
	if err != nil {
		t.Fatal(err)
	}
	if got.Notes != 2 {
		t.Errorf("the summary counts %d notes, want 2", got.Notes)
	}
}

func TestGetSummaryAnswersNothingWhereNoIndexIsBound(t *testing.T) {
	t.Parallel()
	if got, err := vaults.NewGetSummary(nil).Execute(t.Context(), domain.Vault{}); err != nil || got != (domain.VaultSummary{}) {
		t.Errorf("an unbound index answered %+v, %v", got, err)
	}
	var unmade *vaults.GetSummary
	if got, err := unmade.Execute(t.Context(), domain.Vault{}); err != nil || got != (domain.VaultSummary{}) {
		t.Errorf("a use case not made answered %+v, %v", got, err)
	}
}
