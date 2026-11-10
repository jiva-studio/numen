package container

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"

	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/version"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/shell"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/chunking"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/appstate"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/bufferednotes"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/textfiles"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwatch"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/hosttext"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/textbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/sharedmimeinfo"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/trash"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/problemport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/textport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/windowport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/filetext"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/textversion"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/check"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/picker"
	vaultregistry "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// Registry opens the vault registry for this installation.
func (c Config) Registry() (vaultregistry.Ports, error) {
	return vaultRegistry(c.vault())
}

// VaultReaders opens vaults for reading.
func (c Config) VaultReaders() vaultport.VaultReaders {
	return vaultReaders(c.vault())
}

// NoteWriters opens vaults for writing notes.
func (c Config) NoteWriters() vaultport.NoteWriters {
	return noteWriters(c.vault())
}

// FileMovers opens vaults for moving and removing files.
func (c Config) FileMovers() vaultport.FileMovers {
	return fileMovers(c.vault())
}

// VaultLocks holds the write locks of the vaults.
func (c Config) VaultLocks() vaultport.VaultLocks {
	return writeLocks(c.vault())
}

// GetImportedFiles opens files imported from outside the vault.
func (c Config) GetImportedFiles() sourceport.ImportedFiles {
	return getImportedFiles(c.vault())
}

// VaultWatcher monitors vaults for external filesystem changes.
func (c Config) VaultWatcher() vaultport.VaultWatcher {
	return vaultWatcher(c.vault())
}

// VaultIdentity provides vault identification.
func (c Config) VaultIdentity() vaultport.VaultIdentity {
	return vaultIdentity(c.vault())
}

// Trash provides trash and recycling operations.
func (c Config) Trash() machineport.Trash {
	return vaultTrash(c.vault())
}

func (c Config) indexPath() (string, error) {
	return resolveIndexPath(c.vault())
}

// Vaults builds the list of vaults an installation holds, and the vault's own
// tree, against this installation's list and index.
func (c Config) Vaults(ports vaultregistry.Ports, db *vault.Index, notes note.Move) vault.Scenarios {
	return vaultVaults(c.vault(), ports, db, notes)
}

// AddVault registers a new vault in the registry with the given root path and name.
func (c Config) AddVault(ports vaultregistry.Ports, root, name string) (domain.Vault, error) {
	return addVault(c.vault(), ports, root, name)
}

// GetKnownVaults builds the use case that lists known vaults and their folder state.
func (c Config) GetKnownVaults(ports vaultregistry.Ports) *vaultregistry.GetKnownVaults {
	return getKnownVaults(c.vault(), ports)
}

// GetProblems holds the vault to the standard checks.
func (c Config) GetProblems(problems problemport.ProblemQueries) *vault.GetProblems {
	return getProblems(c.vault(), problems)
}

// GetSummary builds the vault summary query use case.
func (c Config) GetSummary(summaries noteport.VaultSummaries) *vaultregistry.GetSummary {
	return getSummary(c.vault(), summaries)
}

// PickFolder builds the folder picker use case when a dialog is provided.
func (c Config) PickFolder(dialog machineport.FolderDialog) *picker.PickFolder {
	return pickFolder(c.vault(), dialog)
}

// OpenIndex opens the cache. The caller closes it.
func (c Config) OpenIndex(ctx context.Context) (*vault.Index, error) {
	return openIndex(c.vault(), ctx)
}

// UpdateIndex updates the notes at the paths given up to date in the index.
func (c Config) UpdateIndex(db *vault.Index) note.IndexUpdater {
	return updateIndex(c.vault(), db)
}

// Moves reports each time the index file changes.
func (c Config) Moves(ctx context.Context) (<-chan struct{}, error) {
	return vaultMoves(c.vault(), ctx)
}

// Notes builds everything that acts on the notes of a vault, against this
// installation's vault readers and writers.
func (c Config) Notes(db *vault.Index, index note.IndexUpdater) note.Scenarios {
	return vaultNotes(c.vault(), db, index)
}

// FocusNote builds the use case that shows a note in the window.
func (c Config) FocusNote(refs noteport.NoteRefs, view windowport.Window) *query.FocusNote {
	return vaultFocusNote(c.vault(), refs, view)
}

// GetNamedPaths builds the note named paths query use case.
func (c Config) GetNamedPaths(names noteport.NamedPaths) *query.GetNamedPaths {
	return getNamedPaths(c.vault(), names)
}

// Read builds the vault file reader use case.
func (c Config) Read() file.Read {
	return vaultRead(c.vault())
}

// VaultOpener is how this installation opens a vault, the way the settings say
// one is read and watched.
func (c Config) VaultOpener(db *vault.Index) *vault.VaultOpener {
	return vaultOpener(c.vault(), db)
}

// VaultOpenerWith is the same, with the watcher, and the readers the walk reads
// the vault through, in place of the installation's own.
func (c Config) VaultOpenerWith(
	db *vault.Index, walking vaultport.VaultReaders, watcher vaultport.VaultWatcher,
) *vault.VaultOpener {
	return vaultOpenerWith(c.vault(), db, walking, watcher)
}

// GetRefresh brings named notes up to date in the index, over the notes of the cache.
func (c Config) GetRefresh(db *vault.Index) vault.Refresh {
	return makeRefresh(c, db, db.Notes())
}

func makeRefresh(c Config, db *vault.Index, notes noteport.NoteRepository) vault.Refresh {
	return vaultMakeRefresh(c.vault(), db, notes)
}

// NewTextVersions is the text versions of the vaults' files.
func (c Config) NewTextVersions(db *vault.Index) (*textversion.Versions, error) {
	return newTextVersions(c.vault(), db)
}

// SetPluginTexts provides the text versions of the vaults to the plugins through
// the host.
func (c Config) SetPluginTexts() {
	setPluginTexts(c.vault())
}

// vaultWiring wires vaults, opening, notes, text versions, index and open vaults.
type vaultWiring struct {
	IndexPath    string
	RegistryPath string
	ServiceDir   string
	SafeMode     bool
	Clock        machineport.Clock
	PluginHost   *PluginHost
	TextVersions *textversion.Versions
	TextReaders  textdomain.Readers
	VaultLocks   *vaultwriter.Locks
	Watches      *vaultwatch.Hub
	ErrorHandler func(error)
	SyncSetting  query.SyncSetting
	CutNotes     chunking.CutNotes
	SetNoteCut   func(*vault.Scan, *vault.Index)
}

func vaultRegistry(c vaultWiring) (vaultregistry.Ports, error) {
	list, err := openRegistry(c)
	if err != nil {
		return vaultregistry.Ports{}, err
	}
	return vaultregistry.Ports{
		Vaults: list, Lookup: list, Saver: list, Remover: list, Recorder: list, Last: list,
	}, nil
}

func openRegistry(c vaultWiring) (*appstate.VaultRegistry, error) {
	if c.RegistryPath != "" {
		return appstate.OpenAt(c.RegistryPath), nil
	}
	return appstate.Open()
}

func vaultReaders(c vaultWiring) vaultport.VaultReaders {
	return vaultreader.Readers{Options: vaultOptions(c)}
}

func vaultWriters(c vaultWiring) vaultwriter.VaultWriters {
	return vaultwriter.VaultWriters{Options: vaultOptions(c), Locks: c.VaultLocks}
}

func noteWriters(c vaultWiring) vaultport.NoteWriters { return vaultWriters(c) }

func folderWriters(c vaultWiring) vaultport.FolderWriters { return vaultWriters(c) }

func fileMovers(c vaultWiring) vaultport.FileMovers { return vaultWriters(c) }

func writeLocks(c vaultWiring) vaultport.VaultLocks { return vaultWriters(c) }

func getImportedFiles(_ vaultWiring) sourceport.ImportedFiles {
	return filesystem.ImportedFiles{}
}

func vaultWatcher(c vaultWiring) vaultport.VaultWatcher {
	return vaultwatch.Watcher{Options: vaultOptions(c), Hub: c.Watches.GetShared()}
}

func vaultIdentity(c vaultWiring) vaultport.VaultIdentity {
	return vaultreader.Identity{Options: vaultOptions(c)}
}

func vaultTrash(c vaultWiring) machineport.Trash {
	return trash.New(c.Clock)
}

func vaultOptions(c vaultWiring) vaultconfig.Options {
	return vaultconfig.Options{
		ServiceDir: c.ServiceDir,
	}
}

func vaultHandleError(c vaultWiring, err error) {
	if c.ErrorHandler != nil {
		c.ErrorHandler(err)
	}
}

func vaultVaults(c vaultWiring, registry vaultregistry.Ports, db *vault.Index, notes note.Move) vault.Scenarios {
	// Erase is Unregister and a folder that goes, so the two hold one Unregister.
	unregister := vault.NewUnregister(registry.Vaults, registry.Remover, db.Vaults())
	erase := vault.NewErase(vaultIdentity(c), vaultTrash(c), unregister)
	find := vaultregistry.NewFind(registry.Lookup)
	return vault.Scenarios{
		Registry: vault.Listing{
			List:        vaultregistry.NewList(registry.Vaults),
			Find:        find,
			FolderCheck: check.NewFolderCheck(vaultReaders(c)),
		},
		Change: vault.Changing{
			Add:        vault.NewAdd(vaultIdentity(c), registry.Vaults, registry.Lookup, registry.Saver, c.Clock),
			Rename:     vault.NewRename(registry.Vaults, registry.Saver, db.Vaults()),
			Remove:     vault.NewRemove(find, unregister, erase),
			Unregister: unregister,
			Erase:      erase,
		},
		Tree: vault.Tree{
			Move:   vault.NewMove(vaultWriters(c), db.Links(), db.SourcesUnder(), db.Sources(), notes),
			Import: vault.NewImport(vaultWriters(c), getImportedFiles(c)),
		},
		RecordOpen: vault.NewRecordOpen(registry.Recorder),
	}
}

func addVault(c vaultWiring, registry vaultregistry.Ports, root, name string) (domain.Vault, error) {
	return vault.NewAdd(vaultIdentity(c), registry.Vaults, registry.Lookup, registry.Saver, c.Clock).Execute(root, name)
}

func getKnownVaults(c vaultWiring, registry vaultregistry.Ports) *vaultregistry.GetKnownVaults {
	recognized := vaultregistry.NewGetKnownVaults(registry.Vaults, registry.Last, vaultReaders(c))
	return &recognized
}

func getProblems(_ vaultWiring, problems problemport.ProblemQueries) *vault.GetProblems {
	return vault.NewStandardGetProblems(problems)
}

func getSummary(_ vaultWiring, summaries noteport.VaultSummaries) *vaultregistry.GetSummary {
	return vaultregistry.NewGetSummary(summaries)
}

func pickFolder(_ vaultWiring, dialog machineport.FolderDialog) *picker.PickFolder {
	if dialog == nil {
		return nil
	}
	pick := picker.NewPickFolder(dialog)
	return &pick
}

// ErrNoIndexPath is a configuration that names no index database.
var ErrNoIndexPath = errors.New("the configuration names no index database: the application sets IndexPath")

func resolveIndexPath(c vaultWiring) (string, error) {
	if c.IndexPath == "" {
		return "", ErrNoIndexPath
	}
	if err := os.MkdirAll(filepath.Dir(c.IndexPath), 0o755); err != nil {
		return "", err
	}
	return c.IndexPath, nil
}

func openIndex(c vaultWiring, ctx context.Context) (*vault.Index, error) {
	path, err := resolveIndexPath(c)
	if err != nil {
		return nil, err
	}
	db, err := index.Open(ctx, path)
	if err != nil {
		return nil, err
	}
	queries := db.NoteQueries()
	return vault.NewIndex(vault.IndexPorts{
		Backend:       db,
		Path:          path,
		Vaults:        db.Vaults(),
		Notes:         db.Notes(),
		FileStats:     queries,
		NameMatches:   queries,
		Headings:      queries,
		Summaries:     queries,
		NoteTypes:     queries,
		TypedPaths:    queries,
		VaultHolds:    queries,
		NoteRefs:      queries,
		OpeningNote:   queries,
		NamedPaths:    queries,
		Passages:      db.ChunkQueries(),
		Progress:      db.ChunkQueries(),
		Stats:         db.ChunkQueries(),
		Sources:       db.Sources(),
		SourcesUnder:  db.Sources(),
		RemovalLookup: db.Sources(),
		SourceRecords: db.Sources(),
		SourceBacklog: db.Sources(),
		Vectors:       db.Sources(),
		VectorQueries: db.Sources(),
		Maintenance:   db.Maintenance(),
		Links:         queries,
		Problems:      queries,
		OpenTabs:      db.OpenTabs(),
		TextVersions:  version.NewIndexVersions(db.TextVersions()),
		SearchFloors:  db.SearchFloors(),
		SearchSamples: db.SearchSamples(),
		Nearest:       db.ChunkQueries(),
	}), nil
}

func updateIndex(c vaultWiring, db *vault.Index) note.IndexUpdater {
	refresh := vault.NewRefresh(
		vaultReaders(c),
		db.Vaults(),
		db.Notes(),
		db.RemovalLookup(),
		db.Sources(),
		c.Clock,
	)
	refresh.FollowRenames = filetext.NewFollowRenames(db.RemovalLookup(), db.Sources())
	refresh.Cut = c.CutNotes
	refresh.Ledger = db.Ledger()
	return func(ctx context.Context, v domain.Vault, paths []string) error {
		_, err := refresh.Execute(ctx, v, paths)
		return err
	}
}

func vaultMoves(c vaultWiring, ctx context.Context) (<-chan struct{}, error) {
	path, err := resolveIndexPath(c)
	if err != nil {
		return nil, err
	}
	return vaultwatch.Watcher{Options: vaultOptions(c), Hub: c.Watches.GetShared()}.File(ctx, path)
}

func vaultNotes(c vaultWiring, db *vault.Index, index note.IndexUpdater) note.Scenarios {
	names, refs, links := db.NamedPaths(), db.NoteRefs(), db.Links()
	sources, indexed := db.Sources(), db.SourcesUnder()
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
			Neighborhood:   query.NewShowNeighbourhood(links, refs),
			GetTitles:      query.NewGetTitles(refs),
			GetTitle:       query.NewGetTitle(refs),
			IsVaultIndexed: query.NewIsVaultIndexed(db.VaultHolds()),
		},
	}
}

func vaultFocusNote(_ vaultWiring, refs noteport.NoteRefs, view windowport.Window) *query.FocusNote {
	return query.NewFocusNote(refs, view)
}

func getNamedPaths(_ vaultWiring, names noteport.NamedPaths) *query.GetNamedPaths {
	return query.NewGetNamedPaths(names)
}

func vaultRead(c vaultWiring) file.Read {
	return file.Read{Readers: vaultReaders(c)}
}

func vaultOpener(c vaultWiring, db *vault.Index) *vault.VaultOpener {
	return vaultOpenerWith(c, db, vaultReaders(c), vaultWatcher(c))
}

func vaultOpenerWith(c vaultWiring,
	db *vault.Index, walking vaultport.VaultReaders, watcher vaultport.VaultWatcher,
) *vault.VaultOpener {
	scan := vault.NewScan(
		walking,
		db.Vaults(),
		db.Notes(),
		db.FileStats(),
		db.Maintenance(),
	)
	if c.SetNoteCut != nil {
		c.SetNoteCut(&scan, db)
	}
	scan.Walks = db.Walks()

	buffered := &bufferednotes.Repository{NoteRepository: db.Notes()}
	return vault.ConfigureVaultOpener(watcher, scan, buffered, vaultMakeRefresh(c, db, buffered))
}

func vaultMakeRefresh(c vaultWiring, db *vault.Index, notes noteport.NoteRepository) vault.Refresh {
	refresh := vault.NewRefresh(vaultReaders(c), db.Vaults(), notes, db.RemovalLookup(), db.Sources(), c.Clock)
	refresh.FollowRenames = filetext.NewFollowRenames(db.RemovalLookup(), db.Sources())
	refresh.Cut = c.CutNotes
	refresh.Ledger = db.Ledger()
	return refresh
}

func newTextVersions(c vaultWiring, db *vault.Index) (*textversion.Versions, error) {
	var plugins textport.TextPlugins = textbridge.NoPlugins{}
	if c.PluginHost != nil && !c.SafeMode {
		plugins = textbridge.NewPlugins(c.PluginHost)
	}
	return textversion.NewVersions(db.TextVersions(), textfiles.Stores{Options: vaultOptions(c)}, plugins, vaultReaders(c), note.NewReadFrontmatter(vaultReaders(c)), c.Clock)
}

func getTextReaders(c vaultWiring) textdomain.Readers {
	if c.TextReaders != nil {
		return c.TextReaders
	}
	if c.TextVersions == nil {
		return nil
	}
	return c.TextVersions
}

func setPluginTexts(c vaultWiring) {
	if c.PluginHost == nil {
		return
	}
	if c.TextVersions == nil {
		c.PluginHost.SetPluginTexts(nil)
		return
	}
	c.PluginHost.SetPluginTexts(hosttext.NewTexts(c.TextVersions, c.TextVersions))
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

func setTextVersionListener(c vaultWiring, listener textdomain.VersionListener) {
	if c.TextVersions != nil {
		c.TextVersions.SetListener(listener)
	}
}

func reportTextFilesChanged(c vaultWiring, ctx context.Context, v domain.Vault, paths []string) error {
	if c.TextVersions == nil {
		return nil
	}
	return c.TextVersions.ReportFilesChanged(ctx, v, paths)
}

func openPluginFile(c vaultWiring) *file.OpenPluginFile {
	return file.NewOpenPluginFile(derivedfile.PluginStores{Options: vaultOptions(c)})
}

func getMediaType(_ vaultWiring) *file.GetMediaType {
	return file.NewGetMediaType(sharedmimeinfo.Open())
}

// OpenVaults is every vault a window has open over db: each is read into the index, and what it
// reports goes to out under label.
func (c Config) OpenVaults(db *vault.Index, out io.Writer, label string) *vault.OpenVaults[Config, *vault.Index] {
	return &vault.OpenVaults[Config, *vault.Index]{Config: c, Index: db, Out: out, Label: label}
}

// SetShellNotes gives a shell the notes it answers about: whether a vault is indexed and what a link
// resolves to. A write updates the index through updateIndex.
func (c Config) SetShellNotes(api *shell.API, db *vault.Index, updateIndex note.IndexUpdater) {
	notes := c.Notes(db, updateIndex)
	api.IsVaultIndexed, api.ResolveLinks = notes.Browse.IsVaultIndexed, notes.Refer.ResolveLinks
}

// BindVaultIndexer has a shell read a vault into the index through vaults, and report a change each
// time a vault is read.
func (c Config) BindVaultIndexer(ctx context.Context, api *shell.API, vaults *vault.OpenVaults[Config, *vault.Index]) {
	vaults.RecordVault = func(v domain.Vault) {
		api.ClearIndexFailure(v.ID)
		api.ReportChange()
	}
	api.SetVaultIndexer(ctx, vaults.IndexVault)
}
