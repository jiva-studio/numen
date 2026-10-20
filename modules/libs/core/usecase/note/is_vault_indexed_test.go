package note_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

func TestIsVaultIndexedAnswersWhetherTheIndexCarriesTheVault(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Heat.md": "# Heat\n"})
	indexed := note.NewIsVaultIndexed(c.db.Queries())

	if !indexed.Execute(t.Context(), c.vault) {
		t.Error("a vault in the index is not carried")
	}
	if indexed.Execute(t.Context(), domain.Vault{ID: "unknown"}) {
		t.Error("a vault the index never read is carried")
	}
}

func TestIsVaultIndexedAnswersFalseWhereNoIndexIsBound(t *testing.T) {
	t.Parallel()
	if note.NewIsVaultIndexed(nil).Execute(t.Context(), domain.Vault{ID: "any"}) {
		t.Error("a build with no index carries a vault")
	}
	var unset *note.IsVaultIndexed
	if unset.Execute(t.Context(), domain.Vault{ID: "any"}) {
		t.Error("an unset use case carries a vault")
	}
}
