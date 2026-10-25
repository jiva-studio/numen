package note_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

func TestGetLinkAddressWritesTheNameOfTheNote(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Heat.md": "# Heat\n"})

	got, err := note.NewGetLinkAddress(c.db.Queries()).Execute(t.Context(), c.vault, "Heat.md")
	if err != nil {
		t.Fatal(err)
	}
	if got != (domain.Address{Scheme: domain.SchemeName, Value: "Heat"}) {
		t.Errorf("the address is %v", got)
	}
}

func TestGetLinkAddressWritesTheNameWhereNoIndexIsBound(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Heat.md": "# Heat\n"})

	got, err := note.NewGetLinkAddress(nil).Execute(t.Context(), c.vault, "physics/Heat.md")
	if err != nil {
		t.Fatal(err)
	}
	if got != (domain.Address{Scheme: domain.SchemeName, Value: "Heat"}) {
		t.Errorf("the address is %v", got)
	}
}
