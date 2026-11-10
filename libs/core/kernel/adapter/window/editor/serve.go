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

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/tablinks"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/vaultreading"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vectors"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/markdown"

	machineport "github.com/jiva-studio/numen/libs/core/kernel/usecase/file/machinecontract"
	sourceport "github.com/jiva-studio/numen/libs/core/kernel/usecase/filetext/sourcecontract"
	indexport "github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing/indexcontract"
	problemport "github.com/jiva-studio/numen/libs/core/kernel/usecase/note/problemcontract"
	windowport "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/windowcontract"
	embeddingport "github.com/jiva-studio/numen/libs/core/kernel/usecase/vectors/embeddingcontract"
)

// Installation is a window put together and running: the pieces the services
// of its routes and anything else working the same vault need.
// The index and the embedder belong to the installation and are made once.
// The passes behind a vault belong to that vault and are rebuilt when another
// is opened.
type Installation struct {
	State *State

	Embedder embeddingport.Embedder
	Asking   embeddingport.Embedder

	notes  *note.Scenarios
	vaults *vaults.Scenarios

	sources       Sources
	pages         pageSet
	dialogs       []func(dialog machineport.PathDialog)
	out           io.Writer
	checkIdentity vaults.CheckIdentity
	closing       <-chan struct{}
	indexer       *indexing.Worker
	embedderError error
	stopEmbedder  func() error
	isRebuilding  bool

	shutdown  shutdownState
	sessionMu sync.Mutex
}

// Setup is what a window is made of before any service is mounted on it.
type Setup struct {
	Tasks         *run.Tasks
	Out           io.Writer
	CheckIdentity vaults.CheckIdentity
	Embedder      embeddingport.Embedder
	Asking        embeddingport.Embedder
	EmbedderError error
	StopEmbedder  func() error
	Progress      indexport.IndexProgress
	OpenTabsStore windowport.OpenTabsStore
	EmitEvent     func(event string)
	IsRebuilding  bool
	Closing       <-chan struct{}
}

// errNoVault is returned for a vault query on a window with no vault open.
var errNoVault = errors.New("the window has no vault")

// NewInstallation makes the state of a window and the worker that indexes its
// vaults. Nothing is served and no vault is open until Start.
func NewInstallation(in Setup, sources Sources) (*Installation, error) {
	if in.EmbedderError != nil {
		writeDiagnostic(in.Tasks.Set, in.Out, "not embedding: %v\n", in.EmbedderError)
		// Record indexing task failure when embedding provider initialization fails.
		in.Tasks.Set(vaultreading.Task{ID: makingVectors, Title: "Indexing", Error: in.EmbedderError.Error()})
	}
	state, err := NewState(in.Tasks)
	if err != nil {
		return nil, err
	}
	state.EmitEvent = in.EmitEvent
	state.OpenTabsStore = in.OpenTabsStore
	state.SetIndexing(in.Progress, in.Embedder)
	indexer := newIndexWorker(sources.ReadVault, state, in.Embedder, quietPeriod, in.Out)
	state.NotifyNoteWritten = indexer.WakeNotes

	return &Installation{
		State:         state,
		Embedder:      in.Embedder,
		Asking:        in.Asking,
		sources:       sources,
		pages:         pageSet{state: state},
		out:           in.Out,
		checkIdentity: in.CheckIdentity,
		closing:       in.Closing,
		indexer:       indexer,
		embedderError: in.EmbedderError,
		stopEmbedder:  in.StopEmbedder,
		isRebuilding:  in.IsRebuilding,
	}, nil
}

// UseScenarios sets the use cases the window works its notes and its vaults
// through. They are built once: other components working the same vault, such
// as agent tools, use these instances and build none of their own, so both
// observe the same errors.
func (o *Installation) UseScenarios(notes *note.Scenarios, target *vaults.Scenarios) {
	o.notes, o.vaults = notes, target
}

// Mount adds the services the window answers through.
func (o *Installation) Mount(routes ...Route) { o.pages.routes = append(o.pages.routes, routes...) }

// SetThemes sets the themes the window offers and puts on its pages. A window
// put together without a catalog serves none.
func (o *Installation) SetThemes(themes numenv1connect.ThemeServiceHandler) { o.pages.themes = themes }

// GetThemes is the themes the window offers, and nil for a window put together without a catalog.
func (o *Installation) GetThemes() numenv1connect.ThemeServiceHandler { return o.pages.themes }

// SetPlugins sets where the calls for the services plugins serve are forwarded,
// and where the files of a plugin are read from.
func (o *Installation) SetPlugins(services PluginServices, dir func(id string) (string, bool)) {
	o.pages.plugins, o.pages.pluginDir = services, dir
}

// OnFolderDialog registers what a service does with the native dialog when the
// application gives one.
func (o *Installation) OnFolderDialog(bind func(dialog machineport.PathDialog)) {
	o.dialogs = append(o.dialogs, bind)
}

// SetFolderDialog sets the native file and folder dialog used when a client
// requests one.
func (o *Installation) SetFolderDialog(dialog machineport.PathDialog) {
	for _, bind := range o.dialogs {
		bind(dialog)
	}
}

// Start opens the first vault, follows the model of the embedder and serves
// recordings over the loopback socket.
func (o *Installation) Start(ctx context.Context, first domain.Vault, files *FileServer) error {
	followModel(o.Embedder, o.State, o.indexer, o.rebuildSession)

	// Reindexing every file applies to this launch only; it is not carried to a
	// vault opened later.
	if err := o.startVault(ctx, first, o.isRebuilding); err != nil {
		return err
	}

	// Recordings are served over a dedicated socket, opened after every
	// dependency it answers through is in place. When the socket cannot be opened,
	// the player has no address.
	onStop := func(why error) {
		writeDiagnostic(o.State.reportTask, o.out, "recordings will no longer play: %v\n", why)
	}
	if playing, why := Listen(ctx, files, onStop); why != nil {
		writeDiagnostic(o.State.reportTask, o.out, "recordings will not play: %v\n", why)
	} else {
		o.State.Playing.Store(playing)
	}
	return nil
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
	playingErr := o.State.Playing.Load().Close()
	// The embedder goes after the work that uses it and before the database,
	// which is the order they depend on each other in.
	return errors.Join(playingErr, o.stopEmbedder(), o.sources.CloseIndex())
}

// Problems returns the window's index cache as a port; Passages does the same.
// Other components working the same vault in this process, including agent
// tools, read the same index.
func (o *Installation) Problems() problemport.ProblemQueries {
	return o.sources.GetProblemQueries()
}

// Passages is the window's index cache as a port, as Problems is.
func (o *Installation) Passages() sourceport.PassageQueries {
	return o.sources.GetPassageQueries()
}

// GetCurrentVault is the vault the window has open.
func (o *Installation) GetCurrentVault() domain.Vault { return o.State.GetCurrentVault() }

// Notes returns the use cases this window works its vault through.
// Other components working the same vault in this process, such as agent
// tools, use the same instances and share error behavior and dependencies.
// Operations on the list of vaults are in Vaults.
// Activity from callers other than the user is rendered through Notes.Drawing.
func (o *Installation) Notes() note.Scenarios { return *o.notes }

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
	return o.sources.GetRefresh()
}

// UpdateIndex brings the notes at the paths up to date in the index.
func (o *Installation) UpdateIndex(ctx context.Context, v domain.Vault, paths []string) error {
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
	wordsAlone = "search by words only"
	// applyingTheme is the task ID for a theme the settings name that the catalog does not have.
	applyingTheme = "apply theme"
	// readingTheSettings is the task ID for a settings name that this build reads under another name.
	// Each message is appended to this ID with its position.
	readingTheSettings = "read settings"
	// readingTheFloor is the task ID for a similarity floor that is stored and cannot be read.
	readingTheFloor = "read search floor"
	// updatingIndex is the task ID for a file written to the vault that the index did not follow.
	updatingIndex = "update index"
)

// SayTheme records a theme warning task, and takes the task out of the list for an empty message,
// which says the theme the settings name is found now.
func (o *Installation) SayTheme(message string) {
	if message == "" {
		o.State.finishTask(applyingTheme)
		return
	}
	o.State.reportTask(vaultreading.Task{ID: applyingTheme, Title: "Apply theme", Error: message})
}

// Report records configuration messages as tasks that are notices, not failures.
func (o *Installation) Report(messages []string) {
	for item, one := range messages {
		o.State.reportTask(vaultreading.Task{
			ID:    fmt.Sprintf("%s %d", readingTheSettings, item),
			Title: "Read settings",
			About: one,
			Kind:  vaultreading.KindNotice,
		})
	}
}

// NewState makes the state of a window that stands on no vault.
func NewState(tasks *run.Tasks) (*State, error) {
	tabLinks, err := tablinks.NewRegistry(linkGrace, tablinks.ScheduleAfter)
	if err != nil {
		return nil, err
	}
	state := &State{
		ChangeEvents: newChangeBroadcaster(),
		Places:       newPlaceBroadcaster(),
		TabIDs:       newTabIDBroadcaster(),
		TabLinks:     tabLinks,
		Edits:        newEditBroadcaster(),
		Window:       &wire.Window{Named: wire.Editor, Tasking: tasks},
	}
	// The editor is open on one vault, and answers which as itself.
	state.Window.Vault = func() string { return string(state.GetCurrentVault().ID) }
	return state, nil
}

// SetIndexing sets how far the vault has been read for meaning, and the model it is read under.
// It is set before any chunk is read: the model name and recipe determine whether
// a chunk already carries a vector, and tell the window that chunks will be
// embedded.
func (s *State) SetIndexing(progress indexport.IndexProgress, embedder embeddingport.Embedder) {
	s.Indexing.Progress = progress
	if embedder != nil {
		name, recipe := vectors.DescribeModel(embedder)
		s.Indexing.Model.Store(name)
		s.Indexing.Recipe.Store(recipe)
	}
}

// ReportWordsAlone records that the search runs on words only, with the reason.
func (s *State) ReportWordsAlone(err error) {
	s.reportTask(vaultreading.Task{ID: wordsAlone, Title: "Search by words only", Error: err.Error()})
}

// ReportFloorError records a similarity floor that is stored and cannot be read.
func (s *State) ReportFloorError(err error) {
	s.reportTask(vaultreading.Task{ID: readingTheFloor, Title: "Read search floor", Error: err.Error()})
}

// Vaults returns the use cases that act on the list of vaults, built once.
// Agent components use the same instances, so both observe the same errors.
func (o *Installation) Vaults() *vaults.Scenarios { return o.vaults }

// shutdownState is whether a flush of pending writes is running and whether the
// window has closed. Vault swaps and window close both flush, one at a time.
//
// A window that has closed shows no other vault.
type shutdownState struct {
	mu         sync.Mutex
	isBusy     bool
	isClosing  bool
	pluginOpen *PluginOpen
}

// PluginOpen tracks a running plugin-initiated vault open; a user flush cancels it. A vault switch
// given the running open shares the open's lock; a user vault switch is given nil.
type PluginOpen struct {
	cancel context.CancelCauseFunc
	end    chan struct{}
}

// errCancelledByUser is the cause a plugin's vault open is canceled with when the user closes the window,
// quits or switches its vault.
var errCancelledByUser = errors.New("the user closed the window or switched its vault")

func (s *shutdownState) tryBegin() error {
	s.mu.Lock()
	for s.pluginOpen != nil {
		current := s.pluginOpen
		s.mu.Unlock()
		current.cancel(errCancelledByUser)
		<-current.end
		s.mu.Lock()
	}
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

func (s *shutdownState) tryBeginPluginOpen(ctx context.Context) (context.Context, *PluginOpen, func(), error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch {
	case s.isClosing:
		return nil, nil, nil, errWindowClosing
	case s.isBusy:
		return nil, nil, nil, errShutdownInProgress
	}
	ctx, cancel := context.WithCancelCause(ctx)
	open := &PluginOpen{cancel: cancel, end: make(chan struct{})}
	s.isBusy, s.pluginOpen = true, open
	return ctx, open, func() {
		s.mu.Lock()
		s.isBusy, s.pluginOpen = false, nil
		s.mu.Unlock()
		cancel(nil)
		close(open.end)
	}, nil
}

func (s *shutdownState) tryBeginSwitch(open *PluginOpen) (func(), bool, error) {
	s.mu.Lock()
	isInsidePluginOpen := open != nil && s.pluginOpen == open
	s.mu.Unlock()
	if isInsidePluginOpen {
		return func() {}, true, nil
	}
	if err := s.tryBegin(); err != nil {
		return nil, false, err
	}
	return s.endKeepOpen, false, nil
}

func (s *shutdownState) isClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isClosing
}

func (s *shutdownState) endKeepOpen() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isBusy = false
}

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
// writes already taken. Calling it again returns true immediately.
// It returns false, with no write discarded, when a page holds work that needs
// user confirmation. A close that arrives while a flush runs returns false: the
// window stays, and the next close flushes again.
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

// importingFiles is what a drop is called in the list of what is being done.
const importingFiles = "import files"

// namedInAnError is how many of the files that stayed outside are named before
// the rest are counted.
const namedInAnError = 3

// Imports copies files dropped onto the window into a folder of the vault; the
// root is the empty path.
// It runs where the drop reached the application, off the page's thread, and
// returns no value: results go to all listeners.
// The watcher does not report images or archives, so Imports names what arrived itself.
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

	pos := vaultreading.Task{ID: importingFiles + " " + into, Title: "Import files", About: into}
	state.reportTask(pos)

	imported, err := o.vaults.Import.Execute(ctx, showing, into, paths)
	if direct := getDirectChildren(into, imported.Imported); len(direct) > 0 {
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

func getDirectChildren(into string, landed []string) []string {
	children := make([]string, 0, len(landed))
	for _, path := range landed {
		if pathpkg.Dir(path) == into || (into == "" && !strings.Contains(path, "/")) {
			children = append(children, path)
		}
	}
	return children
}

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
