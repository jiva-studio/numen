package cli_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/cli"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/none"
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

// stubSession is the session of a test: what each command opens, written as functions, and the closers of what
// they opened.
type stubSession struct {
	Vaults         func() (cli.Vaults, error)
	Indexed        func(ctx context.Context, options cli.IndexOptions) (cli.Indexed, error)
	PluginCommands func() cli.PluginCommands
	closes         *closeList
	failures       []string
}

// closeList is what a test session closes, last opened first.
type closeList struct{ fns []func() error }

func (l *closeList) add(closer func() error) { l.fns = append(l.fns, closer) }

func (d stubSession) session() cli.Session {
	return cli.NewSession(d.OpenVaults, d.OpenIndexed, d.GetPluginCommands, d.Close)
}

func (d stubSession) OpenVaults() (cli.Vaults, error) {
	if d.Vaults == nil {
		return cli.Vaults{}, errNotProvided
	}
	return d.Vaults()
}

func (d stubSession) OpenIndexed(ctx context.Context, options cli.IndexOptions) (cli.Indexed, error) {
	if d.Indexed == nil {
		return cli.Indexed{}, errNotProvided
	}
	return d.Indexed(ctx, options)
}

func (d stubSession) GetPluginCommands() cli.PluginCommands {
	if d.PluginCommands == nil {
		return cli.PluginCommands{List: newScenarios(noCommands{}, noCommands{}, nil, nil, nil).List}
	}
	return d.PluginCommands()
}

func (d stubSession) Close() ([]string, error) {
	var err error
	if d.closes == nil {
		return d.failures, nil
	}
	for _, closeOne := range slices.Backward(d.closes.fns) {
		err = errors.Join(err, closeOne())
	}
	return d.failures, err
}

var errNotProvided = errors.New("the test provided no such part")

func (s *session) deps(where cli.Locations) cli.Session {
	return s.depsOver(where, s.parts(where)).session()
}

func (s *session) depsOver(where cli.Locations, parts vaultParts) stubSession {
	closes := &closeList{}
	return stubSession{
		Vaults: func() (cli.Vaults, error) { return parts.build(), nil },
		Indexed: func(ctx context.Context, options cli.IndexOptions) (cli.Indexed, error) {
			return s.openIndexedSession(ctx, where, parts, closes, options)
		},
		closes: closes,
	}
}

func (s *session) openIndexedSession(
	ctx context.Context, where cli.Locations, parts vaultParts, closes *closeList, options cli.IndexOptions,
) (cli.Indexed, error) {
	rows, closeRows, err := parts.rows(ctx)
	if err != nil {
		return cli.Indexed{}, err
	}
	closes.add(closeRows)
	db, err := index.Open(ctx, where.Index)
	if err != nil {
		return cli.Indexed{}, err
	}
	closes.add(db.Close)
	vaultOptions := vaultconfig.Options{ServiceDir: where.ServiceDir}
	readers := vaultreader.Readers{Options: vaultOptions}
	find := registry.NewFind(parts.ports.Lookup).Execute

	notes := vaults.NewScan(readers, db.Vaults(), db.Notes(), db.NoteQueries(), db.Maintenance(), none.SourceBacklog{})
	notes = notes.Rebuilding(options.Rebuild)
	books := filetext.NewIndex(filetext.IndexDeps{
		Readers: readers, Sources: db.Sources(), Records: db.Sources(), Backlog: db.Sources(), Lookup: db.Sources(),
		Texts: testsupport.Texts{Readers: readers}, ShouldRebuildIndex: options.Rebuild,
	})
	// No vectors: a test uses no model, and a vault is searched by words alone.
	embed := vectors.NewEmbed(readers, db.Sources(), db.Sources(), none.IndexClaim{}, testsupport.Texts{Readers: readers}, none.SearchSource{})

	indexed := parts.indexedOver(rows)
	indexed.Scan = cli.Scan{
		Scan: indexing.NewScanVault(indexing.NewReadWholeVaultByName(find, preparingRead(indexing.NewReadWholeVault(notes, books, embed))), db.NoteQueries()),
		Now:  clockOf(parts.now),
	}
	// No embedder: a test uses no model, so a query is answered by words alone.
	indexed.Search = cli.Search{
		Search: search.NewByName(find, preparing(search.New(db.ChunkSearch(), vaultreader.Readers{Options: vaultOptions},
			testsupport.Texts{Readers: vaultreader.Readers{Options: vaultOptions}}, nil, options.ErrorHandler, nil))),
	}
	indexed.Links = query.NewShowLinksByName(find, query.NewShowLinks(db.NoteQueries()))
	indexed.Problems = vaults.NewGetProblemsByName(find, vaults.NewGetProblems(check.Standard(db.NoteProblems())))
	return indexed, nil
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
	return cli.Vaults{
		Add:        vaults.NewAdd(p.identity, p.ports.Vaults, p.ports.Lookup, p.ports.Saver, p.now),
		Known:      registry.NewGetKnownVaults(p.ports.Vaults, p.ports.Last, p.readers),
		RecordOpen: vaults.NewRecordOpenByName(find, vaults.NewRecordOpen(p.ports.Recorder)),
	}
}

func (p vaultParts) indexedOver(rows vaultport.VaultRepository) cli.Indexed {
	find := registry.NewFind(p.ports.Lookup).Execute
	unregister := vaults.NewUnregister(p.ports.Vaults, p.ports.Remover, rows)
	return cli.Indexed{
		Rename:     vaults.NewRenameByName(find, vaults.NewRename(p.ports.Vaults, p.ports.Saver, rows)),
		Unregister: vaults.NewUnregisterByName(find, unregister),
		Erase:      vaults.NewEraseByName(find, vaults.NewErase(p.identity, p.trash, unregister)),
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
	provided := stubSession{
		Vaults: func() (cli.Vaults, error) {
			return vaultParts{ports: portsOfSingle(singleVaultList{target})}.build(), nil
		},
		Indexed: func(context.Context, cli.IndexOptions) (cli.Indexed, error) {
			return cli.Indexed{Links: query.NewShowLinksByName(registry.NewFind(singleVaultList{target}).Execute, query.NewShowLinks(fixedLinkQueries{
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
		[]string{"links", "notebook", "Bees.md"}, func(cli.Locations) cli.Session { return provided.session() })
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
