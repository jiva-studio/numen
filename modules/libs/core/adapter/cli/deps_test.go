package cli_test

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/cli"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/index"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/appstate"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/embedding"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/search"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"

	"github.com/jiva-studio/numen/modules/libs/core/port/windowport"
)

// getDerived is the shelf this installation keeps its own files on inside a
// vault: what a recognition wrote, and what a transcription wrote.
func getDerived(options vaultconfig.Options) derivedfile.DerivedStores {
	return derivedfile.DerivedStores{
		Options: options,
		Area:    derivedfile.OCRDir,
		Areas:   []string{derivedfile.TranscriptDir},
	}
}

// deps is what the commands of one session run through. The test assembles it
// itself: a registry and an index of its own, the folders it made, and a trash
// nothing of this machine's is sent to.
func (s *session) deps(where cli.Locations) cli.Deps {
	options := vaultconfig.Options{ServiceDir: where.ServiceDir}
	return cli.Deps{
		Vaults: func() (cli.Vaults, error) {
			return cli.Vaults{
				Registry: appstate.OpenAt(where.Registry),
				Readers:  vaultreader.Readers{Options: options},
				Identity: vaultreader.Identity{Options: options},
				Trash:    s.bin,
				Now:      time.Now,
				Rows: func(ctx context.Context) (cli.VaultRows, error) {
					db, err := index.Open(ctx, where.Index)
					if err != nil {
						return cli.VaultRows{}, err
					}
					return cli.VaultRows{Vaults: db.Vaults(), Close: db.Close}, nil
				},
			}, nil
		},

		Links: func(ctx context.Context) (cli.Links, error) {
			db, err := index.Open(ctx, where.Index)
			if err != nil {
				return cli.Links{}, err
			}
			return cli.Links{Show: note.NewShowLinks(db.NoteQueries()), Close: db.Close}, nil
		},

		Scan: func(ctx context.Context, v domain.Vault, rebuild bool) (cli.Scan, error) {
			db, err := index.Open(ctx, where.Index)
			if err != nil {
				return cli.Scan{}, err
			}
			readers := vaultreader.Readers{Options: options}
			store, err := getDerived(options).Open(v)
			if err != nil {
				_ = db.Close()
				return cli.Scan{}, err
			}

			notes := vaults.NewScan(readers, db.Vaults(),
				db.Notes(),
				db.NoteQueries(), db.Maintenance())
			notes.ShouldRebuildIndex = rebuild

			books := source.NewExtract(readers, db.Sources(), db.Sources())
			books.Derived, books.Documents, books.ShouldRebuildIndex = store, testsupport.Documents{}, rebuild

			// No vectors: a test reaches no model, and a vault answers by its
			// words alone.
			vectors := embedding.NewEmbed(readers, db.Sources(), db.Sources())
			vectors.Derived, vectors.Documents = store, testsupport.Documents{}

			return cli.Scan{
				Read: vaults.NewReadWholeVault(notes, books, vectors),
				Summary: func(ctx context.Context) (domain.VaultSummary, error) {
					return db.NoteQueries().GetSummary(ctx, v.ID)
				},
				Close: db.Close,
			}, nil
		},

		// No embedder: a test reaches no model, so a question is answered by its
		// words alone.
		Search: func(ctx context.Context, errorHandler windowport.ErrorHandler) (cli.Search, error) {
			db, err := index.Open(ctx, where.Index)
			if err != nil {
				return cli.Search{}, err
			}
			return cli.Search{
				Search: search.New(db.ChunkQueries(), vaultreader.Readers{Options: options},
					getDerived(options), testsupport.Documents{}, nil, 0, errorHandler),
				Close: db.Close,
			}, nil
		},

		Problems: func(ctx context.Context) (cli.Problems, error) {
			db, err := index.Open(ctx, where.Index)
			if err != nil {
				return cli.Problems{}, err
			}
			return cli.Problems{Problems: db.NoteQueries(), Close: db.Close}, nil
		},
	}
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

// A command runs on what the terminal is handed, and a test that hands it
// values reaches no disk, no database and no model. The openers are what make
func TestACommandRunsOnValuesAlone(t *testing.T) {
	t.Parallel()
	target := domain.Vault{ID: "vlt-01", Name: "notebook", Path: "/nowhere/notebook"}
	provided := cli.Deps{
		Vaults: func() (cli.Vaults, error) {
			return cli.Vaults{Registry: singleVaultList{target}}, nil
		},
		Links: func(context.Context) (cli.Links, error) {
			return cli.Links{Show: note.NewShowLinks(fixedLinkQueries{
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
