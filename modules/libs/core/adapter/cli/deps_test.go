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
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/appstate"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/pdf"
	"github.com/jiva-studio/numen/modules/libs/core/internal/chunking"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/search"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// derived is the shelf this installation keeps its own files on inside a vault:
// what a reading wrote, and what a transcription wrote.
func derived(options filesystem.Options) filesystem.DerivedStores {
	return filesystem.DerivedStores{
		Options: options,
		Area:    filesystem.OCRDir,
		Areas:   []string{filesystem.TranscriptDir},
	}
}

// deps is what the commands of one session run through. The test assembles it
// itself: a registry and an index of its own, the folders it made, and a trash
// nothing of this machine's is sent to.
func (s *session) deps(where cli.Locations) cli.Deps {
	options := filesystem.Options{ServiceDir: where.ServiceDir}
	return cli.Deps{
		Vaults: func() (cli.Vaults, error) {
			return cli.Vaults{
				Registry: appstate.At(where.Registry),
				Readers:  filesystem.VaultReaders{Options: options},
				Identity: filesystem.VaultIdentity{Options: options},
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
			readers := filesystem.VaultReaders{Options: options}
			store, err := derived(options).Open(v)
			if err != nil {
				_ = db.Close()
				return cli.Scan{}, err
			}

			notes := vaults.NewScan(readers, db.Vaults(),
				db.Notes().Cut(chunking.Sizes{}, chunking.Legibility{}),
				db.NoteQueries(), db.Maintenance())
			notes.RebuildIndex = rebuild

			books := source.NewExtract(readers, db.Sources(), db.Sources())
			books.Derived, books.Documents, books.RebuildIndex = store, pdf.Documents{}, rebuild

			// No vectors: a test reaches no model, and a vault answers by its
			// words alone.
			vectors := source.NewEmbed(readers, db.Sources(), db.Sources())
			vectors.Derived, vectors.Documents = store, pdf.Documents{}

			return cli.Scan{
				Read: vaults.NewReadWholeVault(notes, books, vectors),
				Summary: func(ctx context.Context) (domain.VaultSummary, error) {
					return db.NoteQueries().Summary(ctx, v.ID)
				},
				Close: db.Close,
			}, nil
		},

		// No embedder: a test reaches no model, so a question is answered by its
		// words alone.
		Search: func(ctx context.Context, trouble port.Trouble) (cli.Search, error) {
			db, err := index.Open(ctx, where.Index)
			if err != nil {
				return cli.Search{}, err
			}
			return cli.Search{
				Search: search.New(db.ChunkQueries(), filesystem.VaultReaders{Options: options},
					derived(options), pdf.Documents{}, nil, 0, trouble),
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
