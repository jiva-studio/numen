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

func openIndex(c vaultWiring, ctx context.Context) (*Index, error) {
	path, err := indexfolder.Prepare(c.IndexPath)
	if err != nil {
		return nil, err
	}
	db, err := index.Open(ctx, path)
	if err != nil {
		return nil, err
	}
	return &Index{DB: db, File: path, Ledger: vault.NewLedger(), Versions: version.NewIndexVersions(db.TextVersions())}, nil
}

func updateIndex(c vaultWiring, db *Index) edit.IndexUpdater {
	refresh := vault.NewRefresh(
		vaultReaders(c),
		db.Vaults(),
		db.Notes(),
		db.Sources(),
		db.Sources(),
		c.Clock,
	)
	refresh.Cut = c.CutNotes
	refresh.Ledger = db.Ledger
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

// noteViews are the viewers that receive what a person or a program does to the notes. A nil view receives
// no one.
type noteViews struct {
	Moves note.TellMove
}

func vaultNotes(c vaultWiring, db *Index, index edit.IndexUpdater, views noteViews) note.Scenarios {
	names, links := db.NoteQueries(), db.NoteQueries()
	indexed := db.Sources()
	readers := vaultReaders(c)
	writers := vaultWriters(c)
	now := c.Clock

	// One note.Move settles every note that traveled, whether a rename sent it
	// or a move did.
	moveDeps := note.MoveDeps{
		Readers: readers, Writers: writers, Links: links, Names: names,
		Sources: db.Sources(), Index: index, Now: now,
		Drawing: views.Moves, Sync: c.SyncSetting,
	}
	mover := note.NewMove(moveDeps)

	linking := note.NewEditLinks(readers, writers, index, now)
	remove := note.NewRemove(writers, links, indexed, index)
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
			Rename:     note.NewRename(moveDeps, mover),
			Remove:     remove,
			RemoveNote: note.NewRemoveNote(remove, readers),
			EmptyTrash: note.NewEmptyTrash(writers),
		},
		Connect: note.Connecting{
			SaveConnection:   note.NewSaveConnection(readers, writers, index, now, links, names),
			RemoveSide:       note.NewRemoveSide(readers, writers, index, now, links, names),
			RemoveConnection: note.NewRemoveConnection(readers, writers, index, now, links, names),
			Join:             note.NewJoin(readers, writers, index, now, links, names),
			CreateInRelation: note.NewCreateInRelation(create, names),
		},
		Refer: note.Referring{AddLinks: note.NewAddLinks(linking)},
	}
}

func vaultQueries(db *Index) query.Scenarios {
	links, refs := db.NoteQueries(), db.NoteQueries()
	return query.Scenarios{
		Refer: query.Referring{
			Links:          query.NewShowLinks(links),
			Backlinks:      query.NewGetBacklinks(links),
			ResolveLinks:   query.NewResolveLinks(links),
			GetLinkAddress: query.NewGetLinkAddress(db.NoteQueries()),
		},
		Browse: query.Browsing{
			Connection:     query.NewGetConnection(links),
			Neighborhood:   query.NewShowNeighborhood(links, refs),
			GetTitles:      query.NewGetTitles(refs),
			GetTitle:       query.NewGetTitle(refs),
			IsVaultIndexed: query.NewIsVaultIndexed(db.NoteQueries()),
		},
	}
}

func vaultFocusNote(_ vaultWiring, refs noteport.NoteRefs, view query.PlaceFocus) *query.FocusNote {
	return query.NewFocusNote(refs, view)
}

func vaultOpener(c vaultWiring, db *Index) *vault.VaultOpener {
	return vaultOpenerWith(c, db, vaultReaders(c), vaultWatcher(c))
}

func vaultOpenerWith(c vaultWiring,
	db *Index, walking vaultport.VaultReaders, watcher vaultport.VaultWatcher,
) *vault.VaultOpener {
	scan := vault.NewScan(
		walking,
		db.Vaults(),
		db.Notes(),
		db.NoteQueries(),
		db.Maintenance(),
	)
	if c.SetNoteCut != nil {
		c.SetNoteCut(&scan, db)
	}
	scan.Walks = &db.Walks

	buffered := &indexnote.Buffered{NoteRepository: db.Notes()}
	return vault.ConfigureVaultOpener(watcher, scan, buffered, vaultMakeRefresh(c, db, buffered))
}

func vaultMakeRefresh(c vaultWiring, db *Index, notes noteport.NoteRepository) vault.Refresh {
	refresh := vault.NewRefresh(vaultReaders(c), db.Vaults(), notes, db.Sources(), db.Sources(), c.Clock)
	refresh.Cut = c.CutNotes
	refresh.Ledger = db.Ledger
	return refresh
}

func newTextVersions(c vaultWiring, db *Index) (*textversion.Versions, error) {
	return textversion.NewVersions(db.Versions, textfiles.Stores{Options: vaultOptions(c)}, c.RunningHost.Runtime, vaultReaders(c), note.NewReadFrontmatter(vaultReaders(c)), c.Clock)
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

func openVaults(c Config, db *Index, out io.Writer, label string) *vault.OpenVaults[*Index] {
	return &vault.OpenVaults[*Index]{Opener: func(i *Index) *vault.VaultOpener { return vaultOpener(vaultOf(c), i) }, Index: db, Out: out, Label: label}
}

// SetShellNotes gives a shell the notes it answers about: whether a vault is indexed and what a link
// resolves to.
func SetShellNotes(api *shell.API, db *Index) {
	queries := vaultQueries(db)
	api.IsVaultIndexed, api.ResolveLinks = queries.Browse.IsVaultIndexed, queries.Refer.ResolveLinks
}

// BindVaultIndexer has a shell read a vault into the index through vaults, and report a change each
// time a vault is read.
func BindVaultIndexer(_ Config, ctx context.Context, api *shell.API, vaults *vault.OpenVaults[*Index]) {
	vaults.RecordVault = func(v domain.Vault) {
		api.ClearIndexFailure(v.ID)
		api.ReportChange()
	}
	api.SetVaultIndexer(ctx, vaults.IndexVault)
}

func showMovesIn(showing func() (domain.Vault, bool), view MoveView) note.TellMove {
	return func(ctx context.Context, v domain.Vault, went domain.Move) error {
		if current, isShowing := showing(); !isShowing || current.ID != v.ID {
			return nil
		}
		return view.ShowMove(ctx, went)
	}
}
