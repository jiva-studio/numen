package container

import (
	"context"
	"io"

	indexnote "github.com/jiva-studio/numen/libs/core/kernel/adapter/index/note"
	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/version"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hosttext"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/shell"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/indexfolder"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/textfiles"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwatch"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/sharedmimeinfo"
	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/textversion"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"
)

func openIndex(c vaultWiring, ctx context.Context) (*vault.Index, error) {
	path, err := indexfolder.Prepare(c.IndexPath)
	if err != nil {
		return nil, err
	}
	db, err := index.Open(ctx, path)
	if err != nil {
		return nil, err
	}
	queries := db.NoteQueries()
	return vault.NewIndex(vault.IndexPorts{
		Closer: db,
		File:   path,
		Stores: vault.IndexStores{
			Vaults: db.Vaults(), Notes: db.Notes(), OpenTabs: db.OpenTabs(), Maintenance: db.Maintenance(),
		},
		NoteQueries: vault.IndexNoteQueries{
			FileStats: queries, NameMatches: queries, Headings: queries, Summaries: queries, NoteTypes: queries,
			TypedPaths: queries, VaultHolds: queries, NoteRefs: queries, OpeningNote: queries, NamedPaths: queries,
			Links: queries, Problems: db.NoteProblems(),
		},
		Chunks: vault.IndexChunks{
			Passages: db.ChunkSearch(), Nearest: db.ChunkSearch(), Progress: db.ChunkVectors(), Stats: db.ChunkVectors(),
			Index:   db,
			Vectors: db.Sources(), VectorQueries: db.Sources(),
		},
		Source: vault.IndexSources{
			Sources: db.Sources(), SourcesUnder: db.Sources(), RemovalLookup: db.Sources(),
			SourceRecords: db.Sources(), SourceBacklog: db.Sources(),
		},
		Search: vault.IndexSearch{
			SearchFloors: db.SearchFloors(), SearchSamples: db.SearchSamples(),
		},
		TextVersions: version.NewIndexVersions(db.TextVersions()),
	}), nil
}

func updateIndex(c vaultWiring, db *vault.Index) edit.IndexUpdater {
	refresh := vault.NewRefresh(
		vaultReaders(c),
		db.Stores.Vaults,
		db.Stores.Notes,
		db.Source.RemovalLookup,
		db.Source.Sources,
		c.Clock,
	)
	refresh.Cut = c.CutNotes
	refresh.Ledger = db.Ledger()
	return func(ctx context.Context, v domain.Vault, paths []string) error {
		_, err := refresh.Execute(ctx, v, paths)
		return err
	}
}

func vaultMoves(c vaultWiring, ctx context.Context) (<-chan struct{}, error) {
	path, err := indexfolder.Prepare(c.IndexPath)
	if err != nil {
		return nil, err
	}
	return vaultwatch.Watcher{Options: vaultOptions(c), Hub: c.Watches.GetShared()}.File(ctx, path)
}

func vaultNotes(c vaultWiring, db *vault.Index, index edit.IndexUpdater) note.Scenarios {
	names, refs, links := db.NoteQueries.NamedPaths, db.NoteQueries.NoteRefs, db.NoteQueries.Links
	sources, indexed := db.Source.Sources, db.Source.SourcesUnder
	readers := vaultReaders(c)
	writers := vaultWriters(c)
	now := c.Clock

	// One note.Move settles every note that traveled, whether a rename sent it
	// or a move did.
	mover := note.NewMove(note.MoveDeps{
		Readers: readers, Writers: writers, Links: links, Names: names,
		Sources: sources, Index: index, Now: now,
	})
	mover.Sync = c.SyncSetting

	linking := note.NewEditLinks(readers, writers, index, now)
	remove := note.NewRemove(writers, links, indexed, index)
	addresses := query.NewGetLinkAddress(names)
	create := note.NewCreate(writers, names, index, now)

	return note.Scenarios{
		Edit: note.Editing{
			Read:    note.NewRead(readers),
			Write:   note.NewWrite(readers, writers, index, now),
			Create:  create,
			Replace: note.NewReplace(readers, writers, index, now),
			Linking: linking,
		},
		Place: note.Placing{
			Move:       mover,
			Rename:     note.NewRename(mover),
			Remove:     remove,
			RemoveNote: note.NewRemoveNote(remove, readers),
			EmptyTrash: note.NewEmptyTrash(writers),
		},
		Connect: note.Connecting{
			SaveConnection:   note.NewSaveConnection(readers, writers, index, now, links, addresses),
			RemoveSide:       note.NewRemoveSide(readers, writers, index, now, links, addresses),
			RemoveConnection: note.NewRemoveConnection(readers, writers, index, now, links, addresses),
			Join:             note.NewJoin(readers, writers, index, now, links, addresses),
			CreateInRelation: note.NewCreateInRelation(create, addresses),
		},
		Refer: note.Referring{
			AddLinks:       note.NewAddLinks(linking),
			Links:          query.NewShowLinks(links),
			Backlinks:      query.NewGetBacklinks(links),
			ResolveLinks:   query.NewResolveLinks(links),
			GetLinkAddress: addresses,
		},
		Browse: note.Browsing{
			Connection:     query.NewGetConnection(links),
			GetTitles:      query.NewGetTitles(refs),
			GetTitle:       query.NewGetTitle(refs),
			IsVaultIndexed: query.NewIsVaultIndexed(db.NoteQueries.VaultHolds),
		},
	}
}

func vaultFocusNote(_ vaultWiring, refs noteport.NoteRefs, view query.PlaceFocus) *query.FocusNote {
	return query.NewFocusNote(refs, view)
}

func vaultOpener(c vaultWiring, db *vault.Index) *vault.VaultOpener {
	return vaultOpenerWith(c, db, vaultReaders(c), vaultWatcher(c))
}

func vaultOpenerWith(c vaultWiring,
	db *vault.Index, walking vaultport.VaultReaders, watcher vaultport.VaultWatcher,
) *vault.VaultOpener {
	scan := vault.NewScan(
		walking,
		db.Stores.Vaults,
		db.Stores.Notes,
		db.NoteQueries.FileStats,
		db.Stores.Maintenance,
	)
	if c.SetNoteCut != nil {
		c.SetNoteCut(&scan, db)
	}
	scan.Walks = db.Walks()

	buffered := &indexnote.Buffered{NoteRepository: db.Stores.Notes}
	return vault.ConfigureVaultOpener(watcher, scan, buffered, vaultMakeRefresh(c, db, buffered))
}

func vaultMakeRefresh(c vaultWiring, db *vault.Index, notes noteport.NoteRepository) vault.Refresh {
	refresh := vault.NewRefresh(vaultReaders(c), db.Stores.Vaults, notes, db.Source.RemovalLookup, db.Source.Sources, c.Clock)
	refresh.Cut = c.CutNotes
	refresh.Ledger = db.Ledger()
	return refresh
}

func newTextVersions(c vaultWiring, db *vault.Index) (*textversion.Versions, error) {
	return textversion.NewVersions(db.TextVersions, textfiles.Stores{Options: vaultOptions(c)}, c.RunningHost.Runtime, vaultReaders(c), note.NewReadFrontmatter(vaultReaders(c)), c.Clock)
}

func getTextReaders(c vaultWiring) textdomain.Readers {
	if c.TextReaders != nil {
		return c.TextReaders
	}
	if c.TextVersions == nil {
		return nil
	}
	return c.TextVersions.Texts
}

func setPluginTexts(c vaultWiring) {
	if c.TextVersions == nil {
		c.PluginHost.Endpoint.SetPluginTexts(nil)
		return
	}
	c.PluginHost.Endpoint.SetPluginTexts(hosttext.NewTexts(c.TextVersions))
}

func openPluginData(c vaultWiring) *textversion.PluginData {
	if c.TextVersions == nil {
		return nil
	}
	data, err := textversion.NewPluginData(c.TextVersions, derivedfile.PluginStores{Options: vaultOptions(c)})
	if err != nil {
		vaultHandleError(c, err)
		return nil
	}
	return data
}

func openFileText(c vaultWiring) *textversion.FileText {
	if c.TextVersions == nil {
		return nil
	}
	text, err := textversion.NewFileText(c.TextVersions)
	if err != nil {
		vaultHandleError(c, err)
		return nil
	}
	return text
}

func openPluginFile(c vaultWiring) *file.OpenPluginFile {
	return file.NewOpenPluginFile(derivedfile.PluginStores{Options: vaultOptions(c)})
}

func getMediaType(_ vaultWiring) *file.GetMediaType {
	return file.NewGetMediaType(sharedmimeinfo.Open())
}

func openVaults(c Config, db *vault.Index, out io.Writer, label string) *vault.OpenVaults[*vault.Index] {
	return &vault.OpenVaults[*vault.Index]{Opener: func(i *vault.Index) *vault.VaultOpener { return vaultOpener(vaultOf(c), i) }, Index: db, Out: out, Label: label}
}

// SetShellNotes gives a shell the notes it answers about: whether a vault is indexed and what a link
// resolves to. A write updates the index through updateIndex.
func SetShellNotes(c Config, api *shell.API, db *vault.Index, updateIndex edit.IndexUpdater) {
	notes := vaultNotes(vaultOf(c), db, updateIndex)
	api.IsVaultIndexed, api.ResolveLinks = notes.Browse.IsVaultIndexed, notes.Refer.ResolveLinks
}

// BindVaultIndexer has a shell read a vault into the index through vaults, and report a change each
// time a vault is read.
func BindVaultIndexer(_ Config, ctx context.Context, api *shell.API, vaults *vault.OpenVaults[*vault.Index]) {
	vaults.RecordVault = func(v domain.Vault) {
		api.ClearIndexFailure(v.ID)
		api.ReportChange()
	}
	api.SetVaultIndexer(ctx, vaults.IndexVault)
}
