package container

import (
	"context"
	"io"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"

	indexnote "github.com/jiva-studio/numen/libs/core/kernel/adapter/index/note"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/version"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hosttext"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/contenttype"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/indexfolder"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwatch"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/textport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/textversion"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/jiva-studio/numen/libs/shared/note/edit"
)

func openIndex(c vaultWiring, ctx context.Context) (*vaultIndex, error) {
	return openIndexAt(ctx, c.IndexPath, index.WithQueryLog(c.Clock, c.RecordLog))
}

func openIndexAt(ctx context.Context, indexPath string, queryLog index.Option) (*vaultIndex, error) {
	path, err := indexfolder.Prepare(indexPath)
	if err != nil {
		return nil, err
	}
	db, err := index.Open(ctx, path, queryLog)
	if err != nil {
		return nil, err
	}
	return &vaultIndex{DB: db, File: path, Ledger: vault.NewLedger(), Versions: version.NewIndexVersions(db.TextVersions())}, nil
}

func vaultMoves(ctx context.Context, indexPath string, watcher vaultwatch.Watcher) (<-chan struct{}, error) {
	path, err := indexfolder.Prepare(indexPath)
	if err != nil {
		return nil, err
	}
	return watcher.File(ctx, path)
}

// noteViews configures move notifications, write admission and index status for note operations.
type noteViews struct {
	Moves     note.TellMove
	Admission vaultport.WriteAdmission
	Follow    note.IndexFollow
}

func vaultNotes(c vaultWiring, db *vaultIndex, index edit.IndexUpdater, views noteViews) noteUseCases {
	names, links := db.NoteQueries(), db.NoteQueries()
	indexed := db.Sources()
	readers := c.Readers
	writers := c.Writers
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
	remove := note.NewRemove(writers, links, indexed, indexed, index, views.Admission).WithAbsent(c.AbsentFiles)
	create := note.NewCreate(writers, names, index, now, views.Admission)
	connecting := note.ConnectionDeps{
		Readers: readers, Writers: writers, Index: index, Now: now, Links: links, Names: names,
		Admission: views.Admission, Follow: views.Follow,
	}

	return noteUseCases{
		Edit: noteEditing{
			Read:    note.NewRead(readers),
			Write:   note.NewWrite(readers, writers, index, now, views.Admission),
			Create:  create,
			Linking: linking,
		},
		Properties: noteProperties{Frontmatter: note.NewWriteFrontmatter(readers, writers, index, now)},
		Place: notePlacing{
			Move:       mover,
			Rename:     note.NewRename(moveDeps, mover, views.Admission),
			Remove:     remove,
			RemoveNote: note.NewRemoveNote(remove, readers),
			EmptyTrash: note.NewEmptyTrash(writers, textversion.NewTrashVersions(db.Versions, derivedfile.TextStores{Options: c.Options}), views.Admission),
		},
		Connect: noteConnecting{
			SaveConnection:   note.NewSaveConnection(connecting),
			RemoveSide:       note.NewRemoveSide(connecting),
			RemoveConnection: note.NewRemoveConnection(connecting),
			Join:             note.NewJoin(connecting),
			CreateInRelation: note.NewCreateInRelation(create, names),
		},
	}
}

func vaultQueries(db *vaultIndex) query.UseCases {
	links := db.NoteQueries()
	return query.UseCases{
		Refer: query.Referring{
			Links:        query.NewShowLinks(links),
			Backlinks:    query.NewGetBacklinks(links),
			ResolveLinks: query.NewResolveLinks(links),
		},
		Browse: query.Browsing{
			Connection: query.NewGetConnection(links),
		},
	}
}

func vaultOpener(c vaultWiring, db *vaultIndex) *vault.VaultOpener {
	return vaultOpenerWith(c, db, c.Readers, c.Watcher)
}

func vaultOpenerWith(c vaultWiring,
	db *vaultIndex, walking vaultport.VaultReaders, watcher vault.VaultWatcher,
) *vault.VaultOpener {
	scan := newScan(walking, db)
	if c.SetNoteCut != nil {
		c.SetNoteCut(&scan)
	}

	buffered := &indexnote.Buffered{NoteRepository: db.Notes()}
	return vault.ConfigureVaultOpener(watcher, scan, buffered, vaultMakeRefresh(c, db, buffered))
}

func newScan(readers vaultport.VaultReaders, db *vaultIndex) vault.Scan {
	scan := vault.NewScan(readers, db.Vaults(), db.Notes(), db.NoteQueries(), db.Maintenance(), db.Sources())
	scan.Walks = &db.Walks
	return scan
}

func vaultMakeRefresh(c vaultWiring, db *vaultIndex, notes noteport.NoteRepository) vault.Refresh {
	return vault.NewRefreshFrom(vault.RefreshDeps{
		Readers: c.Readers, Vaults: db.Vaults(), Notes: notes, Queries: db.Sources(), Sources: db.Sources(),
		Now: c.Clock, Cut: c.CutNotes, Ledger: db.Ledger,
	})
}

func newTextVersions(options vaultconfig.Options, plugins textversion.TextPlugins, clock machineport.Clock, db *vaultIndex) *textVersions {
	files := derivedfile.TextStores{Options: options, Decoded: derivedfile.NewDecodedTexts(derivedfile.DefaultDecodedBudget)}
	stores, readers := db.Versions, vaultreader.Readers{Options: options}
	texts := textversion.NewTexts(stores, files, readers, clock)
	changes := textversion.NewChanges(texts)
	versions := textversion.NewVersions(stores, files, plugins, clock, changes)
	return &textVersions{Versions: versions, Texts: texts, Changes: changes}
}

func getTextReaders(readers textport.Readers, versions *textVersions) textport.Readers {
	if readers != nil {
		return readers
	}
	if versions == nil {
		return nil
	}
	return versions.Texts
}

func setPluginTexts(lender *hostendpoint.Lender, versions *textVersions) {
	if versions == nil {
		lender.Update(func(l *hostendpoint.Lent) { l.Texts = nil })
		return
	}
	texts := hosttext.NewTexts(versions.Versions, versions.Texts)
	lender.Update(func(l *hostendpoint.Lent) { l.Texts = texts })
}

func openPluginData(options vaultconfig.Options, versions *textVersions) *textversion.PluginData {
	if versions == nil {
		return nil
	}
	return textversion.NewPluginData(versions.Versions, derivedfile.PluginStores{Options: options})
}

func openFileText(versions *textVersions) *textversion.FileText {
	if versions == nil {
		return nil
	}
	return textversion.NewFileText(versions.Texts, versions.Versions)
}

func openPluginFile(options vaultconfig.Options) *file.OpenPluginFile {
	return file.NewOpenPluginFile(derivedfile.PluginStores{Options: options})
}

func getMediaType() *file.GetMediaType {
	return file.NewGetMediaType(contenttype.Open())
}

func openVaults(c vaultWiring, db *vaultIndex, out io.Writer, label string, record func(domain.Vault)) *vault.OpenVaults[*vaultIndex] {
	return &vault.OpenVaults[*vaultIndex]{
		Opener: func(i *vaultIndex) *vault.VaultOpener { return vaultOpener(c, i) }, Index: db, Out: out, Label: label,
		RecordVault: record,
	}
}
