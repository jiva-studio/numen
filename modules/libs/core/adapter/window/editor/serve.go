package editor

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/cards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/file"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/indexing"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/recognition"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/search"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/transcription"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
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

	viewer *viewer

	// Embedder fills the index, and Asking turns a query into a vector. They
	// are one object where the settings name one provider, and two providers
	// of one model where a vault indexed over a network is asked on a machine
	// that has none. Nil for an installation with none, and for Asking also
	// where the two turned out not to be one model; then a search is answered
	// by words alone.
	Embedder port.Embedder
	Asking   port.Embedder

	// notes, cards and vaults are this installation's use cases, built once.
	// Everything else working the same vault is served these, so a dependency
	// named here is named for all of them.
	notes  note.Scenarios
	cards  cards.Scenarios
	vaults vaults.Scenarios

	made  Assembly
	tasks *run.Tasks
	out   io.Writer
	// checkIdentity refuses a vault whose folder no longer carries its identity.
	checkIdentity vaults.CheckIdentity
	// models is this machine's models and processor. Recognising a scan and
	// listening to a recording each hold them, one run at a time.
	models *run.Lock
	// under is what every vault's passes run under.
	under context.Context
	// indexer brings the index of the vault in the window level with it.
	indexer *indexing.Worker
	// embedderError says why this installation embeds nothing, when it does not. It
	// stands in the list of what is being done, and is put back there when a
	// vault going takes its own entries out.
	embedderError error
	// stopEmbedder gives back the models the installation is holding.
	stopEmbedder func() error

	shutdown shutdownState
}

// errNoVault is a question about a vault, asked of a window standing on
// nothing.
var errNoVault = errors.New("the window has no vault")

// Open stands the window on what it was given: the vault it shows, the
// questions it may ask, and a scan running behind it.
//
// made is every piece of the installation this window serves. A window that
// cannot be opened gives the index back.
//
// asked is the vault a person named — a name, a path or an identity. Naming
// none opens the one shown last. An installation holding no vault opens a
// window standing on nothing, and a person makes or adds one there.
//
// The scan is started and left running, and a note is answerable before it
// ends.
//
// Going takes two steps. RequestFlush is called while the window is still open:
// the clients write what only they hold and the writes in the air land.
// Closing then stops the scan, waits for it, and closes the database, in that
// order.
func Open(ctx context.Context, made Assembly, asked string, out io.Writer) (*Installation, error) {
	registry := made.GetRegistry()

	first, err := vaults.NewGetVaultToShow(registry).Execute(asked)
	if err != nil {
		_ = made.CloseIndex()
		return nil, err
	}

	// One list of what is being done, for everything that does anything and for
	// the window that shows it. One job behind it too: what a person asked for
	// is one piece of work however they asked for it. It is made before
	// anything that reports itself into it.
	tasks := run.NewTasks()

	// Opened once, for as long as the window is. A local model is fetched and
	// compiled behind this, so the window is drawn while it arrives.
	embedder, asking, closeEmbedder, why := made.OpenEmbedders(ctx, tasks)
	if why != nil {
		fmt.Fprintf(out, "not embedding: %v\n", why)
		// A provider that made no model is an installation with no vectors for
		// as long as the window is open. It stands in the list under what
		// stopped it.
		tasks.Set(run.Task{ID: makingVectors, Title: "Indexing", Error: why.Error()})
	}
	if closeEmbedder == nil {
		closeEmbedder = func() error { return nil }
	}

	//nolint:contextcheck // a document stays open for the window, not for the request that opened it
	state, services, shown := createWindow(made, tasks, embedder)
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
	services.Search.finds = &finds

	pool := &Installation{
		State:         state,
		Services:      services,
		viewer:        shown,
		Embedder:      embedder,
		Asking:        asking,
		made:          made,
		checkIdentity: vaults.NewCheckIdentity(made.GetVaultIdentity()),
		tasks:         tasks,
		models:        &run.Lock{},
		out:           out,
		under:         ctx,
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
	stopped := func(why error) { fmt.Fprintf(out, "recordings will no longer play: %v\n", why) }
	if playing, why := Listen(services.server, stopped); why != nil {
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
	if o.viewer != nil {
		o.viewer.close()
	}
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
func (o *Installation) Queries() port.NoteQueries     { return o.made.GetNoteQueries() }
func (o *Installation) Problems() port.ProblemQueries { return o.made.GetProblemQueries() }
func (o *Installation) Passages() port.PassageQueries { return o.made.GetPassageQueries() }
func (o *Installation) SourcesKnown() port.SourceQueries {
	return o.made.GetSourceQueries()
}

// GetCurrentVault is the vault the window has open.
func (o *Installation) GetCurrentVault() domain.Vault { return o.State.GetCurrentVault() }

// Notes and Cards are the use cases this window works its vault through.
// Whatever else works the same vault in the same process is served these, so
// what one of them refuses the other refuses and neither is built short of
// something the other has. What a person does to the list of vaults is
// Services.Vaults, where the window itself reaches it.
//
// A caller that is not the person draws what it is doing over the note, which
// is Notes.Drawing and is asked for there.
func (o *Installation) Notes() note.Scenarios  { return o.notes }
func (o *Installation) Cards() cards.Scenarios { return o.cards }

// DraftingLocation locates where a stretch stands in a note as UTF-16 count offsets.
func (o *Installation) DraftingLocation(ctx context.Context, path, stood string) (int, int, bool) {
	contents, err := o.notes.Read.Execute(ctx, o.GetCurrentVault(), path)
	if err != nil || contents.Outcome != note.Ok {
		return 0, 0, false
	}
	at, _ := markdown.FindSpans(contents.Body, stood)
	if len(at) != 1 {
		return 0, 0, false
	}
	return markdown.CountUTF16(contents.Body, at[0].From),
		markdown.CountUTF16(contents.Body, at[0].To), true
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
	// reads under another one. Each sentence stands under this and its place.
	readingTheSettings = "reading the settings"
	// levellingTheIndex is a file that reached the vault and an index that did
	// not follow it.
	levellingTheIndex = "levelling the index"
)

// SayTheme puts what dressing the window had to tell a person in the list of
// what is being done. A theme the settings name that the catalogue has not is
// said there.
func (o *Installation) SayTheme(said string) {
	o.State.reportTask(run.Task{ID: wearingATheme, Title: "Wearing a theme", Error: said})
}

// Report puts what reading the settings had to tell a person in the list of
// what is being done, which is where a person is.
func (o *Installation) Report(said []string) {
	for at, one := range said {
		o.State.reportTask(run.Task{
			ID:    fmt.Sprintf("%s %d", readingTheSettings, at),
			Title: "Reading the settings",
			Error: one,
		})
	}
}

// createWindow is the state a window answers from and the handlers that read
// it, with the pieces that need no vault. The viewer is given back for the
// installation to close.
func createWindow(
	made Assembly, tasks *run.Tasks, embedder port.Embedder,
) (*State, *Services, *viewer) {
	// Where a passage sits on the page is asked of whichever producer made the
	// text it is a place in, which is what the index records.
	highlight := source.NewHighlight(
		made.GetVaultReaders(), made.GetSourceQueries(), made.GetDerivedStores())
	highlight.Documents = made.GetTextExtractor()

	state := &State{
		ChangeEvents: newChangeBroadcaster(),
		Places:       newPlaceBroadcaster(),
		Edits:        newEditBroadcaster(),
		Window:       &wire.Window{Named: wire.Editor, Tasking: tasks},
	}
	shown := newCachingViewer(made.GetPageRenderer())
	services := createServices(state)
	services.getPluginDir = made.GetPluginDir
	state.EmitEvent = made.EmitEvent
	readers := made.GetVaultReaders()
	services.readFiles(readers)
	services.readDocuments(readers, shown)
	services.Book.openClient = made.OpenBookClient
	services.readTexts(made.OpenSourceTexts())
	services.Ocr.highlight = &highlight
	services.Note.getOpeningNote = note.NewGetOpeningNote(made.GetNoteQueries())
	services.Note.getTypes = note.NewGetTypes(made.GetNoteQueries())
	services.Note.getHeadings = note.NewGetHeadings(made.GetNoteQueries())
	services.Note.resolveLinks = note.NewResolveLinks(made.GetLinkQueries())
	services.Search.getNameMatches = search.NewGetNameMatches(made.GetNoteQueries())
	services.Search.getTitles = note.NewGetTitles(made.GetNoteQueries())
	services.Search.getTypes = services.Note.getTypes
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
	return state, services, shown
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
	o.cards = made.OpenCards(o.level)
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

	cutting := &o.cards
	services.Cards.read = &cutting.Read
	services.Cards.list = &cutting.List
	services.Cards.write = &cutting.Write
	services.Cards.create = &cutting.Create
	services.Cards.renameField = &cutting.RenameField
	services.Cards.getTitle = notes.GetTitle
	// A preset is a note the editor writes key by key, and the curve beside its
	// one control is the same simulator the flashcards window runs on.
	running := made.OpenFlashcards(o.level)
	services.Presets.presets = &running.Presets
	services.Presets.curves = &running.Curves
	services.Presets.create = &cutting.Create
	services.Presets.getTitle = notes.GetTitle
	services.Settings.get = made.OpenGetSettings(running.Day)
	services.Settings.write = made.OpenWriteSettings()
	services.Settings.getFile = made.OpenGetSettingsFile()
	services.Settings.writeFile = made.OpenWriteSettingsFile()
	services.Settings.writeSecret = made.OpenWritePluginSecret()
	services.Settings.deleteSecret = made.OpenDeletePluginSecret()

	held := &o.vaults
	services.File.move = &held.Move
	services.File.urls = &source.CreateURL{Writers: made.GetVaultWriters(), Index: o.level}
	services.Vaults.list = &held.List
	services.Vaults.find = &held.Find
	services.Vaults.folderCheck = &held.FolderCheck
	services.Vaults.add = &held.Add
	services.Vaults.rename = &held.Rename
	services.Vaults.remove = &held.Remove
	services.server.find = &held.Find

	services.Artifact.drops = &transcription.Drop{
		Readers: made.GetVaultReaders(),
		Sources: made.GetSourceRepository(),
		Queries: made.GetSourceQueries(),
		Derived: made.GetDerivedStores(),
	}

	// A machine holding neither of the tools an address is reached with binds
	// no downloader, and the window is answered that this build cannot do it.
	if by := made.OpenDownloader(ctx); by != nil {
		fetching := made.OpenImportURL(ctx, by)
		services.readImports(&fetching)
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
func (o *Installation) GetImportURL() *source.ImportURL { return o.Services.Artifact.imports }

// GetSearch is the search the window offers.
func (o *Installation) GetSearch() *search.Search { return o.Services.Search.finds }
