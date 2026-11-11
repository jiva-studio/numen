package note_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func TestGetLinkAddressWritesTheNameOfTheNote(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Heat.md": "# Heat\n"})

	got, err := query.NewGetLinkAddress(c.db.NoteQueries()).Execute(t.Context(), c.vault, "Heat.md")
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

	got, err := query.NewGetLinkAddress(nil).Execute(t.Context(), c.vault, "physics/Heat.md")
	if err != nil {
		t.Fatal(err)
	}
	if got != (domain.Address{Scheme: domain.SchemeName, Value: "Heat"}) {
		t.Errorf("the address is %v", got)
	}
}
