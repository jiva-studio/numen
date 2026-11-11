package cli_test

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/cli"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/appstate"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/filetext"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/check"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vectors"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func (s *session) parts(where cli.Locations) vaultParts {
	options := vaultconfig.Options{ServiceDir: where.ServiceDir}
	return vaultParts{
		ports:    portsOfFile(appstate.OpenAt(where.Registry)),
		readers:  vaultreader.Readers{Options: options},
		identity: vaultreader.Identity{Options: options},
		trash:    s.bin,
		now:      testsupport.NewClock(),
		rows: func(ctx context.Context) (vaultport.VaultRepository, func() error, error) {
			db, err := index.Open(ctx, where.Index)
			if err != nil {
				return nil, nil, err
			}
			return db.Vaults(), db.Close, nil
		},
	}
}

func (s *session) deps(where cli.Locations) cli.Deps {
	options := vaultconfig.Options{ServiceDir: where.ServiceDir}
	find := registry.NewFind(s.parts(where).ports.Lookup).Execute
	return cli.Deps{
		Vaults: func() (cli.Vaults, error) { return s.parts(where).build(), nil },

		Links: func(ctx context.Context) (cli.Links, error) {
			db, err := index.Open(ctx, where.Index)
			if err != nil {
				return cli.Links{}, err
			}
			return cli.Links{Show: query.NewShowLinksByName(find, query.NewShowLinks(db.NoteQueries())), Close: db.Close}, nil
		},

		Scan: func(ctx context.Context, rebuild bool) (cli.Scan, error) {
			db, err := index.Open(ctx, where.Index)
			if err != nil {
				return cli.Scan{}, err
			}
			readers := vaultreader.Readers{Options: options}

			notes := vaults.NewScan(readers, db.Vaults(),
				db.Notes(),
				db.NoteQueries(), db.Maintenance(), nil)
			notes.ShouldRebuildIndex = rebuild

			books := filetext.NewIndex(readers, db.Sources(), db.Sources(), db.Sources(), db.Sources(), testsupport.Texts{Readers: readers})
			books.ShouldRebuildIndex = rebuild

			// No vectors: a test uses no model, and a vault is searched by words alone.
			embed := vectors.NewEmbed(readers, db.Sources(), db.Sources())
			embed.Texts = testsupport.Texts{Readers: readers}

			return cli.Scan{
				Read: indexing.NewReadWholeVaultByName(find, preparingRead(indexing.NewReadWholeVault(notes, books, embed))),
				Summary: func(ctx context.Context, v domain.Vault) (domain.VaultSummary, error) {
					return db.NoteQueries().GetSummary(ctx, v.ID)
				},
				Now:   clockOf(s.parts(where).now),
				Close: db.Close,
			}, nil
		},

		// No embedder: a test uses no model, so a query is answered by words alone.
		Search: func(ctx context.Context, errorHandler window.ErrorHandler) (cli.Search, error) {
			db, err := index.Open(ctx, where.Index)
			if err != nil {
				return cli.Search{}, err
			}
			return cli.Search{
				Search: search.NewByName(find, preparing(search.New(search.Index{Passages: db.ChunkSearch(), Nearest: db.ChunkSearch()}, vaultreader.Readers{Options: options},
					testsupport.Texts{Readers: vaultreader.Readers{Options: options}}, nil, nil, errorHandler))),
				Close: db.Close,
			}, nil
		},

		Problems: func(ctx context.Context) (cli.Problems, error) {
			db, err := index.Open(ctx, where.Index)
			if err != nil {
				return cli.Problems{}, err
			}
			return cli.Problems{Problems: vaults.NewGetProblemsByName(find, vaults.NewGetProblems(check.Standard(db.NoteProblems()))), Close: db.Close}, nil
		},
	}
}

// vaultParts are the pieces the vault commands of a test are built from, so a test can replace one.
type vaultParts struct {
	ports    registry.Ports
	readers  vaultport.VaultReaders
	identity vaultport.VaultIdentity
	trash    machineport.Trash
	now      machineport.Clock
	rows     func(context.Context) (vaultport.VaultRepository, func() error, error)
}

func (p vaultParts) build() cli.Vaults {
	find := registry.NewFind(p.ports.Lookup).Execute
	unregister := func(rows vaultport.VaultRepository) vaults.Unregister {
		return vaults.NewUnregister(p.ports.Vaults, p.ports.Remover, rows)
	}
	return cli.Vaults{
		Add:        vaults.NewAdd(p.identity, p.ports.Vaults, p.ports.Lookup, p.ports.Saver, p.now),
		Known:      registry.NewGetKnownVaults(p.ports.Vaults, p.ports.Last, p.readers),
		RecordOpen: vaults.NewRecordOpenByName(find, vaults.NewRecordOpen(p.ports.Recorder)),
		OpenRename: func(ctx context.Context) (cli.Renaming, error) {
			rows, closeRows, err := p.rows(ctx)
			if err != nil {
				return cli.Renaming{}, err
			}
			return cli.Renaming{Rename: vaults.NewRenameByName(find, vaults.NewRename(p.ports.Vaults, p.ports.Saver, rows)), Close: closeRows}, nil
		},
		OpenUnregister: func(ctx context.Context) (cli.Unregistering, error) {
			rows, closeRows, err := p.rows(ctx)
			if err != nil {
				return cli.Unregistering{}, err
			}
			return cli.Unregistering{Unregister: vaults.NewUnregisterByName(find, unregister(rows)), Close: closeRows}, nil
		},
		OpenErase: func(ctx context.Context) (cli.Erasing, error) {
			rows, closeRows, err := p.rows(ctx)
			if err != nil {
				return cli.Erasing{}, err
			}
			return cli.Erasing{Erase: vaults.NewEraseByName(find, vaults.NewErase(p.identity, p.trash, unregister(rows))), Close: closeRows}, nil
		},
	}
}

func portsOfFile(list *appstate.VaultRegistry) registry.Ports {
	return registry.Ports{Vaults: list, Lookup: list, Saver: list, Remover: list, Recorder: list, Last: list}
}

func portsOfSingle(list singleVaultList) registry.Ports {
	return registry.Ports{Vaults: list, Lookup: list, Saver: list, Remover: list, Recorder: list, Last: list}
}

// singleVaultList is a list holding a single vault, answering nothing about any other.
type singleVaultList struct{ targetVault domain.Vault }

func (o singleVaultList) List() ([]domain.Vault, error)   { return []domain.Vault{o.targetVault}, nil }
func (singleVaultList) Save(domain.Vault) error           { return nil }
func (singleVaultList) Remove(domain.VaultID) error       { return nil }
func (singleVaultList) RecordOpened(domain.VaultID) error { return nil }

func (o singleVaultList) FindVault(nameOrPath string) (domain.Vault, bool, error) {
	if nameOrPath == o.targetVault.Name {
		return o.targetVault, true, nil
	}
	return domain.Vault{}, false, nil
}

func (o singleVaultList) GetLastVault() (domain.Vault, bool, error) { return o.targetVault, true, nil }

// fixedLinkQueries answers what one note points at and what points at it, out of what the
// test wrote down.
type fixedLinkQueries struct{ links, backlinks []domain.ResolvedLink }

func (j fixedLinkQueries) GetLinks(context.Context, domain.VaultID, string) ([]domain.ResolvedLink, error) {
	return j.links, nil
}

func (j fixedLinkQueries) GetBacklinks(context.Context, domain.VaultID, string) ([]domain.ResolvedLink, error) {
	return j.backlinks, nil
}

func (fixedLinkQueries) ResolveLinks(
	context.Context, domain.VaultID, string, []string,
) (map[string]domain.ResolvedLink, error) {
	return nil, nil
}

// A command runs on the values the terminal supplies, and a test that supplies values
// uses no disk, no database and no model. The openers are what make this possible.
func TestACommandRunsOnValuesAlone(t *testing.T) {
	t.Parallel()
	target := domain.Vault{ID: "vlt-01", Name: "notebook", Path: "/nowhere/notebook"}
	provided := cli.Deps{
		Vaults: func() (cli.Vaults, error) {
			return vaultParts{ports: portsOfSingle(singleVaultList{target})}.build(), nil
		},
		Links: func(context.Context) (cli.Links, error) {
			return cli.Links{Show: query.NewShowLinksByName(registry.NewFind(singleVaultList{target}).Execute, query.NewShowLinks(fixedLinkQueries{
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
			}))}, nil
		},
	}

	var out bytes.Buffer
	err := cli.Run(t.Context(), &out, io.Discard,
		[]string{"links", "notebook", "Bees.md"}, func(cli.Locations) cli.Deps { return provided })
	if err != nil {
		t.Fatalf("links: %v\n%s", err, out.String())
	}
	for _, want := range []string{"child      Hives.md", "ref        Honey.md"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("links did not say %q:\n%s", want, out.String())
		}
	}
}

func clockOf(clock machineport.Clock) func() time.Time {
	if clock == nil {
		return nil
	}
	return clock.Now
}
