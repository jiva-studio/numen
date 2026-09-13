package editor

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/wire"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/task"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/cards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// Installation is a window put together and running: the questions a client
// may ask, and the pieces anything else working the same vault needs.
//
// The index and the embedder belong to the installation and are made once. The
// passes behind a vault belong to that vault, and are taken down and built
// again when another is opened.
type Installation struct {
	API *API

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

	made     Assembly
	registry port.VaultRegistry
	tasks    *task.Tasks
	out      io.Writer
	// models is this machine's models and processor. Reading a scan and
	// listening to a recording each hold them, one run at a time.
	models *source.Lock
	// under is what every vault's passes run under.
	under context.Context
	wake  nudges
	// why says why this installation embeds nothing, when it does not. It
	// stands in the list of what is being done, and is put back there when a
	// vault going takes its own entries out.
	why error
	// stopEmbedder gives back the models the installation is holding.
	stopEmbedder func() error

	shutting shutting
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
// Going takes two steps. Settle is called while the window is still drawn: the
// clients write what only they hold and the writes in the air land. Closing
// then stops the scan, waits for it, and closes the database, in that order.
func Open(ctx context.Context, made Assembly, asked string, out io.Writer) (*Installation, error) {
	registry := made.GetRegistry()

	first, err := chooseVault(registry, asked)
	if err != nil {
		_ = made.CloseIndex()
		return nil, err
	}

	// One list of what is being done, for everything that does anything and for
	// the window that shows it. One job behind it too: what a person asked for
	// is one piece of work however they asked for it. It is made before
	// anything that reports itself into it.
	tasks := task.New()

	// Opened once, for as long as the window is. A local model is fetched and
	// compiled behind this, so the window is drawn while it arrives.
	embedder, asking, closeEmbedder, why := made.OpenEmbedders(ctx, tasks)
	if why != nil {
		fmt.Fprintf(out, "not embedding: %v\n", why)
		// A provider that made no model is an installation with no vectors for
		// as long as the window is open. It stands in the list under what
		// stopped it.
		tasks.Set(task.Task{ID: makingVectors, Doing: "Indexing", Error: why.Error()})
	}
	if closeEmbedder == nil {
		closeEmbedder = func() error { return nil }
	}

	wake := newNudges(settled)

	// Where a passage sits on the page is asked of whichever producer made the
	// text it is a place in, which is what the index records.
	highlighting := source.NewHighlight(
		made.GetVaultReaders(), made.GetSourceQueries(), made.GetDerivedStores())
	highlighting.Documents = made.GetTextExtractor()

	api := &API{
		Listeners: newChangeAudience(),
		Places:    newPlaceAudience(),
		Edits:     drawing(),
		Window:    &wire.Window{Named: wire.Editor, Tasking: tasks},
		Wrote:     func() { raise(wake.notes) },
		Readers:   made.GetVaultReaders(),
		Viewer:    newCachingViewer(made.GetPageRenderer()), //nolint:contextcheck // a document stays open for the window, not for the request that opened it
		Highlight: &highlighting,
		Notes: Notes{
			Queries: made.GetNoteQueries(),
			Links:   made.GetLinkQueries(),
		},
		Files: Files{Writers: made.GetVaultWriters()},
	}
	// The editor is open on one vault, and answers which as itself.
	api.Window.Vault = func() string { return string(api.GetShownVault().ID) }
	api.Indexing.Progress = made.GetIndexProgress()
	// Named before anything is read: it is what decides whether a chunk already
	// carries a vector, and what tells the window that something is going to
	// embed what was cut.
	if embedder != nil {
		model := embedder.Model()
		api.Indexing.Model.Store(model.String())
		api.Indexing.Recipe.Store(model.Recipe())
	}

	// The search the window offers is the search the application already does.
	// It is built once the embedder is settled, so a model that could not be
	// fitted leaves the words half to answer on its own.
	// A search short of a half is said where the person is. A window opened
	// from a desktop entry has no terminal to write to.
	finds := made.NewSearch(asking, func(err error) {
		api.say(task.Task{ID: wordsAlone, Doing: "Answering by words alone", Error: err.Error()})
	})
	api.Finds = &finds

	opened := &Installation{
		API:          api,
		Embedder:     embedder,
		Asking:       asking,
		made:         made,
		registry:     registry,
		tasks:        tasks,
		models:       &source.Lock{},
		out:          out,
		under:        ctx,
		wake:         wake,
		why:          why,
		stopEmbedder: closeEmbedder,
	}

	// Everything that acts on this installation's notes, cards and vaults, built
	// once. Whatever else works the same vault — the tools an agent calls — is
	// served these and builds none of its own.
	//
	// A note the person saves is level before the save is answered, so the vault
	// finds what it now holds without waiting on the watch.
	opened.notes = made.OpenNotes(opened.level).FollowMoves(api.GetWindow())
	opened.cards = made.OpenCards(opened.level)
	opened.vaults = made.OpenVaults(opened.notes.Move)

	notes := &opened.notes
	api.Notes.Read = &notes.Read
	api.Notes.Write = &notes.Write
	api.Notes.Create = &notes.Create
	api.Notes.Linking = &notes.Linking
	api.Notes.Rename = &notes.Rename
	api.Notes.Remove = &notes.Remove

	cutting := &opened.cards
	api.Cards = Cards{
		Read:        &cutting.Read,
		List:        &cutting.List,
		Write:       &cutting.Write,
		Create:      &cutting.Create,
		RenameField: &cutting.Rename,
	}
	// A preset is a note the editor writes key by key, and the curve beside its
	// one control is the same simulator the flashcards window runs on.
	running := made.OpenFlashcards(opened.level)
	api.Presets = &running.Presets
	api.Curves = &running.Curves
	least, most := made.GetPartsUnderANodeBounds()
	api.Configuring = SettingsPorts{
		Configured:      made.ReadSettings,
		Models:          made.GetModels,
		ChoosesSetting:  made.TurnSettings,
		ConfiguredFile:  made.ReadSettingsFile,
		WritesFile:      made.WriteSettingsFile,
		PartsUnderANode: Bounds{Least: least, Most: most},
		LatestDayStarts: made.GetLatestDayStarts(),
		Day:             running.Day,
		Now:             made.GetClock(),
	}

	held := &opened.vaults
	api.Files.Move = &held.Move
	api.Files.Import = &held.Import
	api.Files.URLs = &source.CreateURL{Writers: made.GetVaultWriters(), Index: opened.level}
	api.Vaults = Vaults{
		Registry: held.Registry,
		Add:      &held.Add,
		Rename:   &held.Rename,
		Forget:   &held.Forget,
		Erase:    &held.Erase,
	}

	api.Drops = &source.DropTranscript{
		Readers: made.GetVaultReaders(),
		Sources: made.GetSourceRepository(),
		Queries: made.GetSourceQueries(),
		Derived: made.GetDerivedStores(),
	}

	// A machine holding neither of the tools an address is reached with binds
	// no downloader, and the window is answered that this build cannot do it.
	if by := made.OpenDownloader(ctx); by != nil {
		fetching := made.OpenImportURL(ctx, by)
		api.Imports = &fetching
	}

	// Reading every file again is what this launch was asked for, and is not
	// carried to a vault opened later.
	//nolint:contextcheck // the passes behind a vault run under o.under, for as long as the window stands
	if err := opened.arrive(first, made.IsRebuildingIndex()); err != nil {
		_ = closeEmbedder()
		_ = made.CloseIndex()
		return nil, err
	}

	// A recording is played over a socket of its own, opened once everything it
	// answers through is in place. A machine that refuses one leaves the player
	// with no address, and the words are still read.
	stopped := func(why error) { fmt.Fprintf(out, "recordings will no longer play: %v\n", why) }
	if playing, why := Listen(api, stopped); why != nil {
		fmt.Fprintf(out, "recordings will not play: %v\n", why)
	} else {
		api.Playing = playing
	}
	return opened, nil
}

// Close shuts the door on every question, stops the passes behind the vault,
// waits for them, and closes the index.
func (o *Installation) Close() error {
	// First: a search, a note and a link are answered straight from the index,
	// and the index closes here. This stands until the last of them is off it.
	o.API.Shut()
	o.leave()
	if o.API.Viewer != nil {
		o.API.Viewer.close()
	}
	// The socket a recording was played over answers with the vault's files, so
	// it stops before the vault does.
	_ = o.API.Playing.Close()
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

// GetShownVault is the vault the window has open.
func (o *Installation) GetShownVault() domain.Vault { return o.API.GetShownVault() }

// Notes and Cards are the use cases this window works its vault through.
// Whatever else works the same vault in the same process is served these, so
// what one of them refuses the other refuses and neither is built short of
// something the other has. What a person does to the list of vaults is
// API.Vaults, where the window itself reaches it.
//
// A caller that is not the person draws what it is doing over the note, which
// is Notes.Drawing and is asked for there.
func (o *Installation) Notes() note.Scenarios  { return o.notes }
func (o *Installation) Cards() cards.Scenarios { return o.cards }

// Refresh brings named notes up to date. Whatever changes a note calls it, so
// that what changed is findable before the change is reported done.
func (o *Installation) Refresh() vaults.Refresh {
	if on := o.API.showing.Load(); on != nil {
		return on.refresh
	}
	return o.made.GetRefresh()
}

// GetRecognitionWorker reads a scanned document for whoever asks. It is one job
// for the window and for an agent alike, so that what a person started through
// one of them is shown by the other. Nothing while the window has no vault.
func (o *Installation) GetRecognitionWorker() *source.RecognitionWorker {
	if on := o.API.showing.Load(); on != nil {
		return on.recognising
	}
	return nil
}

// GetTranscriptionWorker hears a recording, for whoever asks and for the queue
// behind the vault. It is one job for the window and for an agent alike, so
// that what a person started through one of them is shown by the other. Nothing
// while the window has no vault.
func (o *Installation) GetTranscriptionWorker() *source.TranscriptionWorker {
	if on := o.API.showing.Load(); on != nil {
		return on.transcribing
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
	walkingNotes  = "walking the notes"
	readingBooks  = "reading the books"
	makingVectors = "making the vectors"
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
	o.API.say(task.Task{ID: wearingATheme, Doing: "Wearing a theme", Error: said})
}

// Says puts what reading the settings had to tell a person in the list of what
// is being done, which is where a person is.
func (o *Installation) Says(said []string) {
	for at, one := range said {
		o.API.say(task.Task{
			ID:    fmt.Sprintf("%s %d", readingTheSettings, at),
			Doing: "Reading the settings",
			Error: one,
		})
	}
}
