package note_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/none"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func TestIsVaultIndexedReturnsWhetherTheIndexCarriesTheVault(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"Heat.md": "# Heat\n"})
	indexed := query.NewIsVaultIndexed(c.db.NoteQueries())

	if !indexed.Execute(t.Context(), c.vault) {
		t.Error("a vault in the index is not carried")
	}
	if indexed.Execute(t.Context(), domain.Vault{ID: "unknown"}) {
		t.Error("a vault the index never read is carried")
	}
}

func TestIsVaultIndexedAnswersFalseWhereNoIndexIsBound(t *testing.T) {
	t.Parallel()
	if query.NewIsVaultIndexed(none.NoteQueries{}).Execute(t.Context(), domain.Vault{ID: "any"}) {
		t.Error("a build with no index carries a vault")
	}
	var unset *query.IsVaultIndexed
	if unset.Execute(t.Context(), domain.Vault{ID: "any"}) {
		t.Error("an unset use case carries a vault")
	}
}
