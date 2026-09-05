package webui

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/wire"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/task"
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
	API   *API
	Index *container.Index

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
	notes  container.Notes
	cards  container.Cards
	vaults container.Vaults

	cfg      container.Config
	registry port.VaultRegistry
	tasks    *task.Tasks
	out      io.Writer
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

// Open puts together everything the window needs: the vault it shows, the
// questions it may ask, and a scan running behind it.
//
// asked is the vault a person named — a name, a path or an identity. Naming
// none opens the one shown last. An installation holding no vault opens a
// window standing on nothing, and a person makes or adds one there.
//
// The scan is started and left running. A vault of a hundred thousand notes
// takes a minute and a half, and the first note is answerable long before that.
//
// Going takes two steps. Settle is called while the window is still drawn: the
// clients write what only they hold and the writes in the air land. Closing
// then stops the scan, waits for it, and closes the database — in that order,
// because the database is what the scan writes to.
func Open(ctx context.Context, cfg container.Config, asked string, out io.Writer) (*Installation, error) {
	registry, err := cfg.Registry()
	if err != nil {
		return nil, err
	}
	first, err := chosen(registry, asked)
	if err != nil {
		return nil, err
	}

	db, err := cfg.OpenIndex(ctx)
	if err != nil {
		return nil, err
	}

	// One list of what is being done, for everything that does anything and for
	// the window that shows it. One job behind it too: what a person asked for
	// is one piece of work however they asked for it. It is made before
	// anything that reports itself into it.
	tasks := task.New()

	// Opened once, for as long as the window is. A local model is fetched and
	// compiled behind this, so the window is drawn while it arrives.
	embedder, asking, closeEmbedder, why := cfg.Embedders(ctx, tasks)
	if why != nil {
		fmt.Fprintf(out, "not embedding: %v\n", why)
		// A provider that made no model is an installation with no vectors for
		// as long as the window is open. It stands in the list under what
		// stopped it.
		tasks.Set(task.Task{ID: makingVectors, Doing: "Indexing", Failed: why.Error()})
	}
	if closeEmbedder == nil {
		closeEmbedder = func() error { return nil }
	}

	wake := waking(settled)

	// Where a passage sits on the page is asked of whichever producer made the
	// text it is a place in, which is what the index records.
	highlighting := source.NewHighlight(cfg.VaultReaders(), db.SourcesKnown(), cfg.DerivedStores())
	highlighting.Documents = cfg.TextExtractor()

	api := &API{
		Listeners: following(),
		Places:    focusing(),
		Edits:     drawing(),
		Window:    &wire.Window{Named: wire.Editor, Tasking: tasks},
		Wrote:     func() { raise(wake.notes) },
		Readers:   cfg.VaultReaders(),
		Viewer:    keepingDrawings(cfg.PageRenderer()),
		Highlight: &highlighting,
		Notes: Notes{
			Queries: db.Queries(),
			Links:   db.Links(),
		},
		Files: Files{Writers: cfg.VaultWriters()},
	}
	// The editor is open on one vault, and answers which as itself.
	api.Window.Vault = func() string { return string(api.Showing().ID) }
	api.Indexing.Progress = db.Progress()
	// Named before anything is read: it is what decides whether a chunk already
	// carries a vector, and what tells the window that something is going to
	// embed what was cut.
	if embedder != nil {
		model := embedder.Model()
		api.Indexing.Model.Store(model.String())
		api.Indexing.Recipe.Store(model.Recipe())
	}

	// The themes are the installation's, and a folder that could not be made
	// leaves the ones this binary ships. A theme the settings name that the
	// catalogue has not is said where the person is.
	themes, why := cfg.Themes(func(said string) {
		api.say(task.Task{ID: wearingATheme, Doing: "Wearing a theme", Failed: said})
	})
	if why != nil {
		fmt.Fprintf(out, "themes: %v\n", why)
	}
	api.Themes = themes

	// The search the window offers is the search the application already does.
	// It is built once the embedder is settled, so a model that could not be
	// fitted leaves the words half to answer on its own.
	// A search short of a half is said where the person is. A window opened
	// from a desktop entry has no terminal to write to.
	finds := cfg.Searching(db, asking, func(err error) {
		api.say(task.Task{ID: wordsAlone, Doing: "Answering by words alone", Failed: err.Error()})
	})
	api.Finds = &finds

	opened := &Installation{
		API:          api,
		Index:        db,
		Embedder:     embedder,
		Asking:       asking,
		cfg:          cfg,
		registry:     registry,
		tasks:        tasks,
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
	opened.notes = cfg.Notes(db.Queries(), db.Links(), db.Sources(), db.SourcesKnown(),
		opened.level).Following(api.Viewing())
	opened.cards = cfg.Cards(db.Queries(), db.Links(), opened.level)
	opened.vaults = cfg.Vaults(registry, db, opened.notes.Move)

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
	running := cfg.Flashcards(db.Queries(), db.Links(), opened.level)
	api.Presets = &running.Presets
	api.Curves = &running.Curves
	api.Configuring = Configuring{
		Configured:     cfg.Configured(),
		Models:         cfg.Models(),
		ChoosesSetting: cfg.TurnsSetting(),
		ConfiguredFile: cfg.ConfiguredFile(),
		WritesFile:     cfg.WritesConfiguredFile(),
		PartsUnderANode: Bounds{
			Least: cfg.PartsUnderANodeBounds().Least,
			Most:  cfg.PartsUnderANodeBounds().Most,
		},
		LatestDayStarts: cfg.LatestDayStarts(),
	}

	held := &opened.vaults
	api.Files.Move = &held.Move
	api.Files.Import = &held.Import
	api.Vaults = Vaults{
		Registry: held.Registry,
		Add:      &held.Add,
		Rename:   &held.Rename,
		Forget:   &held.Forget,
		Erase:    &held.Erase,
	}

	api.Drops = &source.DropTranscript{
		Readers: cfg.VaultReaders(),
		Sources: db.Sources(),
		Known:   db.SourcesKnown(),
		Derived: cfg.DerivedStores(),
	}

	// Reading every file again is what this launch was asked for, and is not
	// carried to a vault opened later.
	if err := opened.arrive(first, cfg.RebuildIndex); err != nil {
		_ = closeEmbedder()
		_ = db.Close()
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
	if closed := o.Index.Close(); err == nil {
		err = closed
	}
	return err
}

// Showing is the vault the window has open.
func (o *Installation) Showing() domain.Vault { return o.API.Showing() }

// Notes and Cards are the use cases this window works its vault through.
// Whatever else works the same vault in the same process is served these, so
// what one of them refuses the other refuses and neither is built short of
// something the other has. What a person does to the list of vaults is
// API.Vaults, where the window itself reaches it.
//
// A caller that is not the person draws what it is doing over the note, which
// is Notes.Drawing and is asked for there.
func (o *Installation) Notes() container.Notes { return o.notes }
func (o *Installation) Cards() container.Cards { return o.cards }

// Refresh brings named notes up to date. Whatever changes a note calls it, so
// that what changed is findable before the change is reported done.
func (o *Installation) Refresh() vaults.Refresh {
	if on := o.API.showing.Load(); on != nil {
		return on.opening.Refreshing()
	}
	return vaults.NewRefresh(
		o.cfg.VaultReaders(),
		o.Index.Vaults(),
		o.Index.NotesCutAt(o.cfg.Chunking(), o.cfg.Legibility()),
		o.Index.SourcesKnown(),
		o.Index.Sources(),
	)
}

// Recognising reads a scanned document for whoever asks. It is one job for the
// window and for an agent alike, so that what a person started through one of
// them is shown by the other. Nothing while the window has no vault.
func (o *Installation) Recognising() *source.Recognising {
	if on := o.API.showing.Load(); on != nil {
		return on.recognising
	}
	return nil
}

// Transcribing hears a recording, for whoever asks and for the queue behind the
// vault. It is one job for the window and for an agent alike, so that what a
// person started through one of them is shown by the other. Nothing while the
// window has no vault.
func (o *Installation) Transcribing() *source.Transcribing {
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

// Says puts what reading the settings had to tell a person in the list of what
// is being done, which is where a person is.
func (o *Installation) Says(said []string) {
	for at, one := range said {
		o.API.say(task.Task{
			ID:     fmt.Sprintf("%s %d", readingTheSettings, at),
			Doing:  "Reading the settings",
			Failed: one,
		})
	}
}
