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
	"github.com/jiva-studio/numen/libs/core/kernel/domain/contenttype"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/indexfolder"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwatch"
	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
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

// noteViews are the viewers that receive what a person or a program does to the notes, the gate the writes
// pass, and the report of whether the index followed a write. A nil view receives no one; a nil Follow takes
// the index to follow every write.
// index to follow every write.
type noteViews struct {
	Moves  note.TellMove
	Writes note.WriteGate
	Follow note.IndexFollow
}

func vaultNotes(c vaultWiring, db *Index, index edit.IndexUpdater, views noteViews) NoteScenarios {
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
	remove := note.NewRemove(writers, links, indexed, index, views.Writes)
	create := note.NewCreate(writers, names, index, now, views.Writes)
	connecting := note.ConnectionDeps{
		Readers: readers, Writers: writers, Index: index, Now: now, Links: links, Names: names,
		Gate: views.Writes, Follow: views.Follow,
	}

	return NoteScenarios{
		Edit: NoteEditing{
			Read:    note.NewRead(readers),
			Write:   note.NewWrite(readers, writers, index, now, views.Writes),
			Create:  create,
			Replace: note.NewReplace(readers, writers, index, now),
			Linking: linking,
		},
		Place: NotePlacing{
			Move:       mover,
			Rename:     note.NewRename(moveDeps, mover, views.Writes),
			Remove:     remove,
			RemoveNote: note.NewRemoveNote(remove, readers),
			EmptyTrash: note.NewEmptyTrash(writers, views.Writes),
		},
		Connect: NoteConnecting{
			SaveConnection:   note.NewSaveConnection(connecting),
			RemoveSide:       note.NewRemoveSide(connecting),
			RemoveConnection: note.NewRemoveConnection(connecting),
			Join:             note.NewJoin(connecting),
			CreateInRelation: note.NewCreateInRelation(create, names),
		},
		Refer: NoteReferring{AddLinks: note.NewAddLinks(linking)},
	}
}

func vaultQueries(db *Index) query.Scenarios {
	links, refs := db.NoteQueries(), db.NoteQueries()
	return query.Scenarios{
		Refer: query.Referring{
			Links:        query.NewShowLinks(links),
			Backlinks:    query.NewGetBacklinks(links),
			ResolveLinks: query.NewResolveLinks(links),
		},
		Browse: query.Browsing{
			Connection:     query.NewGetConnection(links),
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
		db.Sources(),
	)
	if c.SetNoteCut != nil {
		c.SetNoteCut(&scan)
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

func newTextVersions(c vaultWiring, db *Index) *TextVersions {
	stores, files, readers := db.Versions, derivedfile.TextStores{Options: vaultOptions(c)}, vaultReaders(c)
	texts := textversion.NewTexts(stores, files, readers, note.NewReadFrontmatter(readers), c.Clock)
	changes := textversion.NewChanges(texts)
	versions := textversion.NewVersions(stores, files, c.RunningHost.Runtime, c.Clock, changes)
	return &TextVersions{Versions: versions, Texts: texts, Changes: changes}
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
	c.PluginHost.Endpoint.SetPluginTexts(hosttext.NewTexts(c.TextVersions.Versions, c.TextVersions.Texts))
}

func openPluginData(c vaultWiring) *textversion.PluginData {
	if c.TextVersions == nil {
		return nil
	}
	return textversion.NewPluginData(c.TextVersions.Versions, derivedfile.PluginStores{Options: vaultOptions(c)})
}

func openFileText(c vaultWiring) *textversion.FileText {
	if c.TextVersions == nil {
		return nil
	}
	return textversion.NewFileText(c.TextVersions.Texts, c.TextVersions.Versions)
}

func openPluginFile(c vaultWiring) *file.OpenPluginFile {
	return file.NewOpenPluginFile(derivedfile.PluginStores{Options: vaultOptions(c)})
}

func getMediaType(_ vaultWiring) *file.GetMediaType {
	return file.NewGetMediaType(contenttype.Open())
}

func openVaults(c Config, db *Index, out io.Writer, label string, record func(domain.Vault)) *vault.OpenVaults[*Index] {
	return &vault.OpenVaults[*Index]{
		Opener: func(i *Index) *vault.VaultOpener { return vaultOpener(vaultOf(c), i) }, Index: db, Out: out, Label: label,
		RecordVault: record,
	}
}

// SetShellNotes gives a shell the notes it answers about: whether a vault is indexed.
func SetShellNotes(api *shell.API, db *Index) {
	queries := vaultQueries(db)
	api.Ensure = indexing.NewEnsureIndexed(&api.Runs, queries.Browse.IsVaultIndexed, api.Window, api)
}
