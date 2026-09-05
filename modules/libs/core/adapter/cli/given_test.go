package cli_test

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/cli"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// oneVault is a list holding a single vault, answering nothing about any other.
type oneVault struct{ held domain.Vault }

func (o oneVault) All() ([]domain.Vault, error) { return []domain.Vault{o.held}, nil }
func (oneVault) Save(domain.Vault) error        { return nil }
func (oneVault) Remove(domain.VaultID) error    { return nil }
func (oneVault) Opened(domain.VaultID) error    { return nil }

func (o oneVault) Find(nameOrPath string) (domain.Vault, bool, error) {
	if nameOrPath == o.held.Name {
		return o.held, true, nil
	}
	return domain.Vault{}, false, nil
}

func (o oneVault) Last() (domain.Vault, bool, error) { return o.held, true, nil }

// joined answers what one note points at and what points at it, out of what the
// test wrote down.
type joined struct{ links, backlinks []domain.ResolvedLink }

func (j joined) Links(context.Context, domain.VaultID, string) ([]domain.ResolvedLink, error) {
	return j.links, nil
}

func (j joined) Backlinks(context.Context, domain.VaultID, string) ([]domain.ResolvedLink, error) {
	return j.backlinks, nil
}

func (joined) Resolve(
	context.Context, domain.VaultID, string, []string,
) (map[string]domain.ResolvedLink, error) {
	return nil, nil
}

// A command runs on what the terminal is handed, and a test that hands it
// values reaches no disk, no database and no model. The openers are what make
// that possible: what stands behind one is the caller's to choose.
func TestACommandRunsOnValuesAlone(t *testing.T) {
	t.Parallel()
	held := domain.Vault{ID: "vlt-01", Name: "notebook", Path: "/nowhere/notebook"}
	given := cli.Deps{
		Vaults: func() (cli.Vaults, error) {
			return cli.Vaults{Registry: oneVault{held}}, nil
		},
		Links: func(context.Context) (cli.Links, error) {
			return cli.Links{Show: note.NewShowLinks(joined{
				links: []domain.ResolvedLink{{
					Link: domain.Link{
						Target: domain.Address{Scheme: domain.SchemeName, Value: "Hives"},
						Role:   domain.RoleChild,
					},
					From: "Bees.md",
					To:   "Hives.md",
				}},
				backlinks: []domain.ResolvedLink{{
					Link: domain.Link{
						Target: domain.Address{Scheme: domain.SchemeName, Value: "Bees"},
						Role:   domain.RoleRef,
					},
					From: "Honey.md",
					To:   "Bees.md",
				}},
			})}, nil
		},
	}

	var out bytes.Buffer
	err := cli.Run(t.Context(), &out, io.Discard,
		[]string{"links", "notebook", "Bees.md"}, func(cli.Locations) cli.Deps { return given })
	if err != nil {
		t.Fatalf("links: %v\n%s", err, out.String())
	}
	for _, want := range []string{"child      Hives.md", "ref        Honey.md"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("links did not say %q:\n%s", want, out.String())
		}
	}
}
