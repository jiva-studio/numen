package editor

import (
	"context"
	"errors"
	"fmt"
	"io"
	pathpkg "path"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/core/adapter/window/editor/assembly"
	"github.com/jiva-studio/numen/libs/core/adapter/window/editor/command"
	"github.com/jiva-studio/numen/libs/core/adapter/window/editor/textmenu"
	"github.com/jiva-studio/numen/libs/core/adapter/window/editor/texts"
	"github.com/jiva-studio/numen/libs/core/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/usecase/file"
	"github.com/jiva-studio/numen/libs/core/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/core/usecase/run"
	"github.com/jiva-studio/numen/libs/core/usecase/search"
	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/usecase/vectors"
	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/markdown"

	sourceport "github.com/jiva-studio/numen/libs/core/usecase/filetext/sourcecontract"
	noteport "github.com/jiva-studio/numen/libs/core/usecase/note/notecontract"
	problemport "github.com/jiva-studio/numen/libs/core/usecase/note/problemcontract"
	embeddingport "github.com/jiva-studio/numen/libs/core/usecase/vectors/embeddingcontract"
)

// Installation is a window put together and running: the questions a client
// may ask, and the pieces anything else working the same vault needs.
//
// The index and the embedder belong to the installation and are made once. The
// passes behind a vault belong to that vault, and are taken down and built
// again when another is opened.
type Installation struct {
	// State is what every handler reads, and Services the handlers themselves.
	State    *State
	Services *Services

	// Embedder fills the index and Asking converts a query into a vector. They
	// are one object when the settings name one provider, and two providers of
	// one model when a vault indexed over a network is queried on a machine
	// without that provider. Nil when the installation has no embedder; Asking
	// is also nil when the two are not one model. Search then uses words only.
	Embedder embeddingport.Embedder
	Asking   embeddingport.Embedder

	// notes and vaults are this installation's use cases, built once.
	// Everything else working the same vault is served these, so a dependency
	// named here is named for all of them.
	notes  note.Scenarios
	vaults vaults.Scenarios

	made  Assembly
	tasks *run.Tasks
	out   io.Writer
	// checkIdentity rejects a vault whose folder no longer carries its identity.
	checkIdentity vaults.CheckIdentity
	// models is this machine's models and processor. Recognising a scan and
	// listening to a recording each hold them, one run at a time.
	models *run.Lock
	// closing signals when the window root context ends.
	closing <-chan struct{}
	// indexer brings the index of the vault in the window level with it.
	indexer *indexing.Worker
	// embedderError records the error if embedding model initialization failed.
	embedderError error
	// stopEmbedder gives back the models the installation is holding.
	stopEmbedder func() error

	shutdown shutdownState
}

// errNoVault is returned for a vault query on a window with no vault open.
var errNoVault = errors.New("the window has no vault")

// Open initializes the editor window session for the specified vault.
// When targetVault is empty, opens the most recently viewed vault.
func Open(ctx context.Context, made Assembly, targetVault string, out io.Writer) (*Installation, error) {
	registry := made.GetRegistry()

	first, err := vaults.NewGetVaultToShow(registry).Execute(targetVault)
	if err != nil {
		_ = made.CloseIndex()
		return nil, err
	}

	// One task list serves every operation and the window that displays it.
	// A request is one task regardless of how it was issued. The list is created
	// before any component that reports into it.
	tasks := run.NewTasks()
	models := &run.Lock{}
	made.ReportPluginFailures(tasks)
	made.SetPluginWork(tasks, models)

	// Opened once, for as long as the window is. A local model is fetched and
	// compiled behind this, so the window is drawn while it arrives.
	embedder, asking, closeEmbedder, why := made.OpenEmbedders(ctx, tasks)
	if why != nil {
		fmt.Fprintf(out, "not embedding: %v\n", why)
		// Record indexing task failure when embedding provider initialization fails.
		tasks.Set(run.Task{ID: makingVectors, Title: "Indexing", Error: why.Error()})
	}
	if closeEmbedder == nil {
		closeEmbedder = func() error { return nil }
	}

	state, services := createWindow(made, tasks, embedder)
	indexer := newIndexWorker(made, state, embedder, quietPeriod, out)
	state.NotifyNoteWritten = indexer.WakeNotes

	// The window's search is the application's search. It is built after the
	// embedder is resolved; when the model could not be loaded, search uses words
	// only. The degraded state is reported as a task because a window opened
	// from a desktop entry has no terminal.
	finds := made.NewSearch(asking, func(err error) {
		state.reportTask(run.Task{ID: wordsAlone, Title: "Answering by words alone", Error: err.Error()})
	})
	services.Search.Finds = &finds
	floors := made.OpenFloors(embedder, newFloorErrorReporter(state))
	services.Search.Floors = &floors

	pool := &Installation{
		State:         state,
		Services:      services,
		Embedder:      embedder,
		Asking:        asking,
		made:          made,
		checkIdentity: vaults.NewCheckIdentity(made.GetVaultIdentity()),
		tasks:         tasks,
		models:        models,
		out:           out,
		closing:       ctx.Done(),
		indexer:       indexer,
		embedderError: why,
		stopEmbedder:  closeEmbedder,
	}

	pool.wireUseCases()

	// Reindexing every file applies to this launch only; it is not carried to a
	// vault opened later.
	if err := pool.startVault(ctx, first, made.IsRebuildingIndex()); err != nil {
		_ = closeEmbedder()
		_ = made.CloseIndex()
		return nil, err
	}

	// Recordings are served over a dedicated socket, opened after every
	// dependency it answers through is in place. When the socket cannot be opened,
	// the player has no address and the transcript is still readable.
	onStop := func(why error) { fmt.Fprintf(out, "recordings will no longer play: %v\n", why) }
	if playing, why := Listen(services.server, onStop); why != nil {
		fmt.Fprintf(out, "recordings will not play: %v\n", why)
	} else {
		services.openPlaying(playing)
	}
	return pool, nil
}

// Close shuts the door on every question, stops the passes behind the vault,
// waits for them, and closes the index.
func (o *Installation) Close() error {
	// Searches, note reads and link lookups read the index directly; the index
	// closes here, after the last of them has finished.
	o.State.Close()
	o.stopVault()
	// The socket a recording was played over answers with the vault's files, so
	// it stops before the vault does.
	_ = o.Services.Playing.Close()
	// The embedder goes after the work that uses it and before the database,
	// which is the order they depend on each other in.
	err := o.stopEmbedder()
	if closed := o.made.CloseIndex(); err == nil {
		err = closed
	}
	return err
}

// Queries returns the window's index cache as a port; Problems, Passages and
// SourcesKnown do the same. Other components working the same vault in this
// process, including agent tools, read the same index.
func (o *Installation) Queries() noteport.NoteQueries        { return o.made.GetNoteQueries() }
func (o *Installation) Problems() problemport.ProblemQueries { return o.made.GetProblemQueries() }
func (o *Installation) Passages() sourceport.PassageQueries  { return o.made.GetPassageQueries() }
func (o *Installation) SourcesKnown() sourceport.SourceQueries {
	return o.made.GetSourceQueries()
}

// GetCurrentVault is the vault the window has open.
func (o *Installation) GetCurrentVault() domain.Vault { return o.State.GetCurrentVault() }

// Notes returns the use cases this window works its vault through.
// Other components working the same vault in this process, such as agent
// tools, use the same instances, so they share error behavior and
// dependencies. Operations on the list of vaults are in Services.Vaults.
//
// Activity from callers other than the user is rendered over the note through
// Notes.Drawing.
func (o *Installation) Notes() note.Scenarios { return o.notes }

// DraftingLocation locates where target text occurs in a note as UTF-16 count offsets.
func (o *Installation) DraftingLocation(ctx context.Context, path, text string) (int, int, bool) {
	contents, err := o.notes.Read.Execute(ctx, o.GetCurrentVault(), path)
	if err != nil || contents.Outcome != note.Ok {
		return 0, 0, false
	}
	pos, _ := markdown.FindRanges(contents.Body, text)
	if len(pos) != 1 {
		return 0, 0, false
	}
	return markdown.CountUTF16(contents.Body, pos[0].Start),
		markdown.CountUTF16(contents.Body, pos[0].End), true
}

// Refresh brings named notes up to date. Whatever changes a note calls it, so
// that what changed is findable before the change is reported done.
func (o *Installation) Refresh() vaults.Refresh {
	if on := o.State.session.Load(); on != nil {
		return on.refresh
	}
	return o.made.GetRefresh()
}

// level brings named notes up to date in the index, through whatever is
// following the vault they are in. A note the window makes is level before the
// answer comes back, so it is drawn as soon as it exists.
func (o *Installation) level(ctx context.Context, v domain.Vault, paths []string) error {
	_, err := o.Refresh().Execute(ctx, v, paths)
	return err
}

// What each pass behind the window is called in the list of what is being done.
// One name each, so a pass that reports itself again replaces itself.
const (
	walkingNotes  = indexing.WalkingNotes
	indexingTexts = indexing.IndexingTexts
	makingVectors = indexing.MakingVectors
	// wordsAlone is the task ID for a search that runs without semantic retrieval.
	wordsAlone = "answering by words alone"
	// wearingATheme is a theme the settings name that the catalogue has not.
	wearingATheme = "wearing a theme"
	// readingTheSettings is a name the settings file holds that this build
	// reads under another one. Each sentence is placed under this and its place.
	readingTheSettings = "reading the settings"
	// readingTheFloor is a similarity floor the index holds and cannot read.
	readingTheFloor = "reading the search floor"
	// levellingTheIndex is a file that reached the vault and an index that did
	// not follow it.
	levellingTheIndex = "levelling the index"
)

// SayTheme records a theme warning task.
func (o *Installation) SayTheme(message string) {
	o.State.reportTask(run.Task{ID: wearingATheme, Title: "Wearing a theme", Error: message})
}

// Report records configuration messages as tasks.
func (o *Installation) Report(messages []string) {
	for item, one := range messages {
		o.State.reportTask(run.Task{
			ID:    fmt.Sprintf("%s %d", readingTheSettings, item),
			Title: "Reading the settings",
			Error: one,
		})
	}
}

// createWindow is the state a window answers from and the handlers that read
// it, with the pieces that need no vault.
func createWindow(
	made Assembly, tasks *run.Tasks, embedder embeddingport.Embedder,
) (*State, *Services) {
	state := &State{
		ChangeEvents: newChangeBroadcaster(),
		Places:       newPlaceBroadcaster(),
		Edits:        newEditBroadcaster(),
		Window:       &wire.Window{Named: wire.Editor, Tasking: tasks},
	}
	services := createServices(state)
	services.getPluginDir = made.GetPluginDir
	state.EmitEvent = made.EmitEvent
	state.OpenTabsStore = made.GetOpenTabsStore()
	readers := made.GetVaultReaders()
	services.readFiles(readers, made.OpenProxy(), made.GetMediaType())
	services.Note.getOpeningNote = note.NewGetOpeningNote(made.GetNoteQueries())
	services.Note.getTypes = note.NewGetTypes(made.GetNoteQueries())
	services.Note.getHeadings = note.NewGetHeadings(made.GetNoteQueries())
	services.Note.resolveLinks = note.NewResolveLinks(made.GetLinkQueries())
	services.Search.GetNameMatches = search.NewGetNameMatches(made.GetNoteQueries())
	services.Search.GetTitles = note.NewGetTitles(made.GetNoteQueries())
	services.Search.GetTypes = services.Note.getTypes
	services.File.getFolderEntries = file.NewGetFolderEntries(readers, made.GetNoteQueries())
	services.File.getFileKinds = file.NewGetFileKinds(readers, made.GetNoteQueries())
	services.File.createFolder = file.NewCreateFolder(made.GetVaultWriters())

	// The editor is open on one vault, and answers which as itself.
	state.Window.Vault = func() string { return string(state.GetCurrentVault().ID) }
	state.Indexing.Progress = made.GetIndexProgress()
	// Set before any chunk is read: the model name and recipe determine whether
	// a chunk already carries a vector, and tell the window that chunks will be
	// embedded.
	if embedder != nil {
		name, recipe := vectors.DescribeModel(embedder)
		state.Indexing.Model.Store(name)
		state.Indexing.Recipe.Store(recipe)
	}
	return state, services
}

// wireUseCases builds the use cases the window and everything else working its
// vault are served, and hands each handler the ones it answers through.
func (o *Installation) wireUseCases() {
	made, services := o.made, o.Services

	// Use cases for this installation's notes and vaults, built once. Other
	// components working the same vault, such as agent tools, use these instances
	// and build none of their own.
	//
	// A note saved by the user is refreshed in the index before the save returns,
	// so the vault reflects its content without waiting for the watcher.
	o.notes = made.OpenNotes(o.level).FollowMoves(o.State.GetWindow())
	o.vaults = made.OpenVaults(o.notes.Move)

	notes := &o.notes
	services.Note.read = &notes.Read
	services.Note.write = &notes.Write
	services.Note.create = &notes.Create
	services.Note.editLinks = &notes.Linking
	services.Note.replaceLinks = &notes.ReplaceLinks
	services.Note.rename = &notes.Rename
	services.Note.showNeighbourhood = &notes.Neighbourhood
	services.Note.getLinkAddress = notes.GetLinkAddress
	services.File.remove = &notes.Remove

	services.Settings.Get = made.OpenGetSettings()
	services.PluginServices = made.GetPluginServices()
	services.Playing.setPluginServices(services.PluginServices)
	services.Command = command.NewHandler(o.State, made.GetPluginCommands())
	if menu := made.OpenFileMenu(); menu != nil {
		services.TextVersion = textmenu.NewHandler(o.State, menu)
	}
	if text := made.OpenFileText(); text != nil {
		services.Text = texts.NewHandler(o.State, text)
	}
	services.Agent.GetAgent = made.GetAgent
	services.Settings.Write = made.OpenWriteSettings()
	services.Settings.WriteSecret = made.OpenWritePluginSecret()
	services.Settings.DeleteSecret = made.OpenDeletePluginSecret()
	services.Settings.PluginRetry = made.OpenRetryPlugin()
	services.Settings.OnPluginChange = func() {
		if o.Embedder != nil {
			name, recipe := vectors.DescribeModel(o.Embedder)
			o.State.Indexing.Model.Store(name)
			o.State.Indexing.Recipe.Store(recipe)
		}
		if o.indexer != nil {
			o.indexer.WakeSources()
		}
	}
	if data := made.OpenPluginData(); data != nil {
		services.Settings.PluginData, services.Settings.Vaults = data, o.State
	}
	services.openChoosePath = made.OpenChoosePluginPath

	target := &o.vaults
	services.File.move = &target.Move
	services.Vaults.list = &target.List
	services.Vaults.find = &target.Find
	services.Vaults.folderCheck = &target.FolderCheck
	services.Vaults.add = &target.Add
	services.Vaults.rename = &target.Rename
	services.Vaults.remove = &target.Remove
	services.server.find = &target.Find
}

// Vaults returns the use cases that act on the list of vaults, built once.
// Agent components use the same instances, so both observe the same errors.
func (o *Installation) Vaults() *vaults.Scenarios { return &o.vaults }

// GetSearch is the search the window offers.
func (o *Installation) GetSearch() *search.Search { return o.Services.Search.Finds }

// shutdownState is whether a flush of pending writes is running and whether the
// window has closed. Vault swaps and window close both flush, one at a time.
//
// A window that has closed shows no other vault.
type shutdownState struct {
	mu        sync.Mutex
	isBusy    bool
	isClosing bool
}

// tryBegin starts a flush, or returns the reason one cannot start.
func (s *shutdownState) tryBegin() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch {
	case s.isClosing:
		return errWindowClosing
	case s.isBusy:
		return errShutdownInProgress
	}
	s.isBusy = true
	return nil
}

// endKeepOpen ends the flush and leaves the window open.
func (s *shutdownState) endKeepOpen() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isBusy = false
}

// end ends the flush. closed is whether the window is now closed.
func (s *shutdownState) end(closed bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isBusy = false
	s.isClosing = closed
}

// errWindowClosing is the error for a vault requested in a closed window.
var errWindowClosing = errors.New("this window is closing")

// errShutdownInProgress is the error for a vault requested while a flush is
// running.
var errShutdownInProgress = errors.New("the window is flushing pending writes")

// FlushPending has every page write what only it holds, then waits for the
// writes already taken. It is called while the window is still showing, and
// calling it again returns true immediately. It returns false when a page holds
// work that needs user confirmation; no write has been discarded.
//
// A vault swap flushes too. A close that arrives while a flush runs returns
// false: the window stays, and the next close flushes again.
func (o *Installation) FlushPending(ctx context.Context) bool {
	switch err := o.shutdown.tryBegin(); {
	case errors.Is(err, errWindowClosing):
		return true
	case err != nil:
		return false
	}

	flushed := flushPending(ctx, o.State.Window, &o.State.Writing)
	o.shutdown.end(flushed)
	return flushed
}

// WaitForFlush waits for every page to write its pending data. The wait is on
// ctx ended or the vault was requested again.
func (o *Installation) WaitForFlush(ctx context.Context) bool {
	return o.State.Window.WaitForFlush(ctx)
}

// flushPending has every client write what only it holds, then waits for the
// writes already taken. It returns whether the flush completed.
//
// A silent client is bounded by ctx. A client with a pending user prompt ends the
// flush with false: the vault stays open until the user responds. The writes
// themselves have no bound.
func flushPending(ctx context.Context, pages *wire.Window, writes *inflight) bool {
	if !pages.RequestFlush(ctx) {
		return false
	}
	<-writes.seal()
	return true
}

// FlushTimeout bounds how long a page may take to write its unsaved data when
// the vault closes. A page with a pending user prompt counts as responded;
// further waiting depends on the user.
const FlushTimeout = 3 * time.Second

type Assembly = assembly.Assembly
type VaultAssembly = assembly.VaultAssembly
type IndexAssembly = assembly.IndexAssembly
type PluginAssembly = assembly.PluginAssembly
type SettingsAssembly = assembly.SettingsAssembly
type WorkerAssembly = assembly.WorkerAssembly
type ScenarioAssembly = assembly.ScenarioAssembly

// importingFiles is what a drop is called in the list of what is being done.
const importingFiles = "importing files"

// namedInAnError is how many of the files that stayed outside are named before
// the rest are counted.
const namedInAnError = 3

// Imports copies files dropped onto the window into a folder of the vault; the
// root is the empty path.
//
// It runs where the drop reached the application, off the thread the page is
// served on, and returns no value: results are reported to all listeners.
//
// The watcher does not report images or archives, so Imports names what
// arrived to the listeners itself.
func (o *Installation) Imports(ctx context.Context, into string, paths []string) {
	state := o.State
	if len(paths) == 0 {
		return
	}
	showing := state.GetCurrentVault()
	if showing.ID == "" {
		return
	}
	if !state.Writing.begin() {
		return
	}
	defer state.Writing.finish()

	pos := run.Task{ID: importingFiles + " " + into, Title: "Bringing files in", About: into}
	state.reportTask(pos)

	imported, err := o.vaults.Import.Execute(ctx, showing, into, paths)
	if direct := getDirectChildren(into, imported.Landed); len(direct) > 0 {
		state.ChangeEvents.publish(change{paths: direct})
	}
	if err != nil {
		pos.Error = err.Error()
		state.reportTask(pos)
		return
	}
	if failureText := describeImportFailures(imported.Errors); failureText != "" {
		pos.Error = failureText
		state.reportTask(pos)
		return
	}
	state.finishTask(pos.ID)
}

// getDirectChildren returns items in the target folder.
func getDirectChildren(into string, landed []string) []string {
	children := make([]string, 0, len(landed))
	for _, path := range landed {
		if pathpkg.Dir(path) == into || (into == "" && !strings.Contains(path, "/")) {
			children = append(children, path)
		}
	}
	return children
}

// describeImportFailures formats import failures into a single descriptive message.
func describeImportFailures(errs []vaults.ImportFailure) string {
	if len(errs) == 0 {
		return ""
	}
	messages := make([]string, 0, namedInAnError)
	for _, one := range errs[:min(len(errs), namedInAnError)] {
		messages = append(messages, fmt.Sprintf("%s: %v", one.Name, one.Why))
	}
	if rest := len(errs) - len(messages); rest > 0 {
		messages = append(messages, fmt.Sprintf("and %d more", rest))
	}
	return strings.Join(messages, "; ")
}
