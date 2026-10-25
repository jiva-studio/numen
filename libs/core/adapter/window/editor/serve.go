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
	"github.com/jiva-studio/numen/libs/core/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/usecase/file"
	"github.com/jiva-studio/numen/libs/core/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/core/usecase/recognition"
	"github.com/jiva-studio/numen/libs/core/usecase/run"
	"github.com/jiva-studio/numen/libs/core/usecase/search"
	"github.com/jiva-studio/numen/libs/core/usecase/source"
	"github.com/jiva-studio/numen/libs/core/usecase/transcription"
	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/markdown"

	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/port/noteport"
	"github.com/jiva-studio/numen/libs/core/port/recognitionport"
	"github.com/jiva-studio/numen/libs/core/port/sourceport"
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

	importURL *source.ImportURL

	// Embedder fills the index, and Asking turns a query into a vector. They
	// are one object where the settings name one provider, and two providers
	// of one model where a vault indexed over a network is asked on a machine
	// that has none. Nil for an installation with none, and for Asking also
	// where the two turned out not to be one model; then a search is answered
	// by words alone.
	Embedder embeddingport.Embedder
	Asking   embeddingport.Embedder

	// notes, cards and vaults are this installation's use cases, built once.
	// Everything else working the same vault is served these, so a dependency
	// named here is named for all of them.
	notes  note.Scenarios
	vaults vaults.Scenarios

	made  Assembly
	tasks *run.Tasks
	out   io.Writer
	// checkIdentity refuses a vault whose folder no longer carries its identity.
	checkIdentity vaults.CheckIdentity
	// models is this machine's models and processor. Recognising a scan and
	// listening to a recording each hold them, one run at a time.
	models *run.Lock
	// ctx is the root context for vault sessions.
	ctx context.Context
	// indexer brings the index of the vault in the window level with it.
	indexer *indexing.Worker
	// embedderError records the error if embedding model initialization failed.
	embedderError error
	// stopEmbedder gives back the models the installation is holding.
	stopEmbedder func() error

	shutdown shutdownState
}

// errNoVault is a question about a vault, asked of a window standing on
// nothing.
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

	// One list of what is being done, for everything that does anything and for
	// the window that shows it. One job behind it too: what a person asked for
	// is one piece of work however they asked for it. It is made before
	// anything that reports itself into it.
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

	// The search the window offers is the search the application already does.
	// It is built once the embedder is settled, so a model that could not be
	// fitted leaves the words half to answer on its own.
	// A search short of a half is said where the person is. A window opened
	// from a desktop entry has no terminal to write to.
	finds := made.NewSearch(asking, func(err error) {
		state.reportTask(run.Task{ID: wordsAlone, Title: "Answering by words alone", Error: err.Error()})
	})
	services.Search.Finds = &finds

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
		ctx:           ctx,
		indexer:       indexer,
		embedderError: why,
		stopEmbedder:  closeEmbedder,
	}

	pool.wireUseCases(ctx)

	// Indexing every file again is what this launch was asked for, and is not
	// carried to a vault opened later.
	//nolint:contextcheck // the passes behind a vault run under o.under, for as long as the window stands
	if err := pool.startVault(first, made.IsRebuildingIndex()); err != nil {
		_ = closeEmbedder()
		_ = made.CloseIndex()
		return nil, err
	}

	// A recording is played over a socket of its own, opened once everything it
	// answers through is in place. A machine that refuses one leaves the player
	// with no address, and the words are still read.
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
	// First: a search, a note and a link are answered straight from the index,
	// and the index closes here. This stands until the last of them is off it.
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

// The cache this window answers out of, as the ports it hands on. Whatever else
// works the same vault in the same process asks it here, so the window and the
// tools an agent calls read one index.
func (o *Installation) Queries() noteport.NoteQueries            { return o.made.GetNoteQueries() }
func (o *Installation) Problems() recognitionport.ProblemQueries { return o.made.GetProblemQueries() }
func (o *Installation) Passages() sourceport.PassageQueries      { return o.made.GetPassageQueries() }
func (o *Installation) SourcesKnown() sourceport.SourceQueries {
	return o.made.GetSourceQueries()
}

// GetCurrentVault is the vault the window has open.
func (o *Installation) GetCurrentVault() domain.Vault { return o.State.GetCurrentVault() }

// Notes is the use cases this window works its vault through.
// Whatever else works the same vault in the same process is served these, so
// what one of them refuses the other refuses and neither is built short of
// something the other has. What a person does to the list of vaults is
// Services.Vaults, where the window itself reaches it.
//
// A caller that is not the person draws what it is doing over the note, which
// is Notes.Drawing and is asked for there.
func (o *Installation) Notes() note.Scenarios { return o.notes }

// DraftingLocation locates where target text occurs in a note as UTF-16 count offsets.
func (o *Installation) DraftingLocation(ctx context.Context, path, text string) (int, int, bool) {
	contents, err := o.notes.Read.Execute(ctx, o.GetCurrentVault(), path)
	if err != nil || contents.Outcome != note.Ok {
		return 0, 0, false
	}
	pos, _ := markdown.FindSpans(contents.Body, text)
	if len(pos) != 1 {
		return 0, 0, false
	}
	return markdown.CountUTF16(contents.Body, pos[0].From),
		markdown.CountUTF16(contents.Body, pos[0].To), true
}

// Refresh brings named notes up to date. Whatever changes a note calls it, so
// that what changed is findable before the change is reported done.
func (o *Installation) Refresh() vaults.Refresh {
	if on := o.State.session.Load(); on != nil {
		return on.refresh
	}
	return o.made.GetRefresh()
}

// GetRecognitionWorker reads a scanned document for whoever asks. It is one job
// for the window and for an agent alike, so that what a person started through
// one of them is shown by the other. Nothing while the window has no vault.
func (o *Installation) GetRecognitionWorker() *recognition.Worker {
	if on := o.State.session.Load(); on != nil {
		return on.recognitionWorker
	}
	return nil
}

// GetTranscriptionWorker hears a recording, for whoever asks and for the queue
// behind the vault. It is one job for the window and for an agent alike, so
// that what a person started through one of them is shown by the other. Nothing
// while the window has no vault.
func (o *Installation) GetTranscriptionWorker() *transcription.Worker {
	if on := o.State.session.Load(); on != nil {
		return on.transcriptionWorker
	}
	return nil
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
	walkingNotes    = indexing.WalkingNotes
	extractingBooks = indexing.ExtractingBooks
	makingVectors   = indexing.MakingVectors
	// wordsAlone is a search answered short of the half that asks by meaning.
	wordsAlone = "answering by words alone"
	// wearingATheme is a theme the settings name that the catalogue has not.
	wearingATheme = "wearing a theme"
	// readingTheSettings is a name the settings file holds that this build
	// reads under another one. Each sentence is placed under this and its place.
	readingTheSettings = "reading the settings"
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
	readers := made.GetVaultReaders()
	services.readFiles(readers)
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
	// Named before anything is read: it is what decides whether a chunk already
	// carries a vector, and what tells the window that something is going to
	// embed what was cut.
	if embedder != nil {
		model := embedder.GetModel()
		state.Indexing.Model.Store(model.String())
		state.Indexing.Recipe.Store(model.Recipe())
	}
	return state, services
}

// wireUseCases builds the use cases the window and everything else working its
// vault are served, and hands each handler the ones it answers through.
func (o *Installation) wireUseCases(ctx context.Context) {
	made, services := o.made, o.Services

	// Everything that acts on this installation's notes, cards and vaults, built
	// once. Whatever else works the same vault — the tools an agent calls — is
	// served these and builds none of its own.
	//
	// A note the person saves is level before the save is answered, so the vault
	// finds what it now holds without waiting on the watch.
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
	services.Agent.getAgent = made.GetAgent
	services.Settings.Write = made.OpenWriteSettings()
	services.Settings.GetFile = made.OpenGetSettingsFile()
	services.Settings.WriteFile = made.OpenWriteSettingsFile()
	services.Settings.WriteSecret = made.OpenWritePluginSecret()
	services.Settings.DeleteSecret = made.OpenDeletePluginSecret()
	services.Settings.PluginRetry = made.OpenRetryPlugin()

	target := &o.vaults
	services.File.move = &target.Move
	services.File.urls = &source.CreateURL{Writers: made.GetVaultWriters(), Index: o.level}
	services.Vaults.list = &target.List
	services.Vaults.find = &target.Find
	services.Vaults.folderCheck = &target.FolderCheck
	services.Vaults.add = &target.Add
	services.Vaults.rename = &target.Rename
	services.Vaults.remove = &target.Remove
	services.server.find = &target.Find

	// A machine holding neither of the tools an address is reached with binds
	// no downloader, and the window is answered that this build cannot do it.
	if delta := made.OpenDownloader(ctx); delta != nil {
		fetching := made.OpenImportURL(ctx, delta)
		o.importURL = &fetching
	}
}

// Vaults is the use cases that act on the list of vaults, built once. An agent
// that works the list is served these, so what one of them refuses the other
// refuses.
func (o *Installation) Vaults() *vaults.Scenarios { return &o.vaults }

// GetCreateURL is the use case that makes the file a web address is kept in.
func (o *Installation) GetCreateURL() *source.CreateURL { return o.Services.File.urls }

// GetImportURL is what fetches the text at an address, or nil where this build
// holds neither of the tools that reach one.
func (o *Installation) GetImportURL() *source.ImportURL { return o.importURL }

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
// calling it again returns true at once. It returns false where a page holds
// work a person has to answer for, and then nothing has been taken away.
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
// A silent client is bounded by ctx. A client raising a question ends the
// flush with false: the vault stays open and the wait is on a person. The
// writes themselves have no bound.
func flushPending(ctx context.Context, pages *wire.Window, writes *inflight) bool {
	if !pages.RequestFlush(ctx) {
		return false
	}
	<-writes.seal()
	return true
}

// FlushTimeout is how long a page has to write what only it holds when the
// vault goes. A page raising a question has answered, and the wait from there
// is on a person.
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

// Imports copies files a person release over the window into a folder of the
// vault, the root being the empty path.
//
// It runs where the drop reached the application, which is off the thread the
// page is served on, and it answers nothing: what arrived is said to everyone
//
// The watcher reports a picture or an archive to nobody, so what arrived is
// named to the listeners here.
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
