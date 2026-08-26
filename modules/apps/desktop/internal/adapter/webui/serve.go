package webui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/task"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/note"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/source"
	usecase "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/vault"
)

// Opened is a window put together and running: the questions a client may ask,
// and the pieces anything else working the same vault needs.
//
// The index and the embedder belong to the installation and are made once. The
// passes behind a vault belong to that vault, and are taken down and built
// again when another is opened.
type Opened struct {
	API   *API
	Index *container.Index

	// Embedder fills the index, and Asking turns a query into a vector. They
	// are one object where the settings name one station, and two stations
	// of one model where a vault indexed over a network is asked on a machine
	// that has none. Nil for an installation with none, and for Asking also
	// where the two turned out not to be one model; then a search is answered
	// by words alone.
	Embedder port.Embedder
	Asking   port.Embedder

	cfg      container.Config
	registry port.VaultRegistry
	tasks    *task.Tasks
	out      io.Writer
	// under is what every vault's passes run under.
	under context.Context
	wake  nudges
	// vectors is why this installation embeds nothing, when it does not. It
	// stands in the list of what is being done, and is put back there when a
	// vault going takes its own entries out.
	vectors error
	// stopEmbedder gives back the models the installation is holding.
	stopEmbedder func() error

	// on is the half of the window that belongs to the vault it is showing, and
	// is nothing while that vault is being changed.
	on atomic.Pointer[showing]

	// One settling runs at a time, and the second to arrive is refused.
	mu    sync.Mutex
	busy  bool
	going bool
}

// showing is the half of the window that belongs to one vault: the passes
// running behind it and what ends them.
type showing struct {
	scan        usecase.Scan
	refresh     usecase.Refresh
	recognising *container.Recognising
	// stop ends every pass this vault started, and ended waits for them.
	stop  context.CancelFunc
	ended func()
}

// errGoing is a vault asked for in a window that has settled to close.
var errGoing = errors.New("the window is closing")

// errSettling is a vault asked for while the window is already settling what it
// owes.
var errSettling = errors.New("the window is settling what it owes")

// errAsking is a vault asked for while a page holds work a person is being
// asked about.
var errAsking = errors.New("a page is holding work a person has to answer for")

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
func Open(ctx context.Context, cfg container.Config, asked string, out io.Writer) (*Opened, error) {
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
		// A station that made no model is an installation with no vectors for
		// as long as the window is open. It stands in the list under what
		// stopped it.
		tasks.Set(task.Task{ID: makingVectors, Doing: "Indexing", Failed: why.Error()})
	}
	if closeEmbedder == nil {
		closeEmbedder = func() error { return nil }
	}

	wake := waking(settled)

	api := &API{
		Notes:     db.Queries(),
		Links:     db.Links(),
		Listeners: following(),
		Watching:  focusing(),
		Drawing:   drawing(),
		Progress:  db.Progress(),
		Tasking:   tasks,
		Reads:     &note.Read{Readers: cfg.VaultReaders()},
		Saves:     &note.Write{Readers: cfg.VaultReaders(), Writers: cfg.VaultWriters()},
		Wrote:     func() { raise(wake.notes) },
		Readers:   cfg.VaultReaders(),
		Writers:   cfg.VaultWriters(),
		Viewer:    keepingDrawings(cfg.Documents()),
		// Where a passage sits on the page is asked of whichever producer made
		// the text it is a place in, which is what the index records.
		Marking: &source.Marks{
			Readers:   cfg.VaultReaders(),
			Sources:   db.SourcesKnown(),
			Derived:   cfg.DerivedStores(),
			Documents: cfg.Documents(),
		},
	}
	// Named before anything is read: it is what decides whether a chunk already
	// carries a vector, and what tells the window that something is going to
	// embed what was cut.
	if embedder != nil {
		model := embedder.Model()
		api.Model.Store(model.String())
		api.Recipe.Store(model.Recipe())
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

	opened := &Opened{
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
		vectors:      why,
		stopEmbedder: closeEmbedder,
	}
	api.Scan = opened.scanning

	api.Makes = &note.Create{
		Writers:   cfg.VaultWriters(),
		Names:     db.Queries(),
		Index:     opened.level,
		Extension: filedUnder(cfg),
	}
	api.Joins = &note.Linking{
		Readers: cfg.VaultReaders(),
		Writers: cfg.VaultWriters(),
		Index:   opened.level,
	}
	// One note.Move settles every note that travelled, whether a rename sent it
	// or a move did.
	moving := note.Move{
		Readers: cfg.VaultReaders(),
		Writers: cfg.VaultWriters(),
		Links:   api.Links,
		Sources: db.Sources(),
		Index:   opened.level,
		Moving: func(ctx context.Context, went domain.Went) {
			_ = api.Viewing().Moved(ctx, went)
		},
		Sync: cfg.Syncing(),
	}
	api.Sync = cfg.Syncing()
	api.Chooses = cfg.Turns()
	api.Renames = &note.Rename{Move: moving}
	api.Moves = &usecase.Move{
		Writers: cfg.VaultWriters(),
		Links:   api.Links,
		Known:   db.SourcesKnown(),
		Sources: db.Sources(),
		Notes:   moving,
	}
	api.Removes = &note.Remove{
		Writers: cfg.VaultWriters(),
		Links:   api.Links,
		Known:   db.SourcesKnown(),
		Index:   opened.level,
	}

	// The vaults this installation holds, beside the one the window is showing.
	// Erase is Forget and a folder that goes, so the two hold one Forget.
	forget := usecase.Forget{Registry: registry, Index: db.Vaults()}
	api.Vaults = registry
	api.Adding = &usecase.Add{
		Identity: cfg.VaultIdentity(),
		Registry: registry,
		Now:      time.Now,
	}
	api.Renaming = &usecase.Rename{Registry: registry, Index: db.Vaults()}
	api.Forgetting = &forget
	api.Erasing = &usecase.Erase{
		Identity: cfg.VaultIdentity(),
		Trash:    cfg.Trash(),
		Forget:   forget,
	}

	// Reading every file again is what this launch was asked for, and is not
	// carried to a vault opened later.
	if err := opened.arrive(first, cfg.RebuildIndex); err != nil {
		_ = closeEmbedder()
		_ = db.Close()
		return nil, err
	}
	return opened, nil
}

// chosen is the vault this window opens: the one a person named, else the one
// shown last, else the first this installation holds. An installation holding
// none answers with no vault at all.
//
// A vault named and not on the list is refused, and the window does not open.
func chosen(registry port.VaultRegistry, asked string) (domain.Vault, error) {
	if asked != "" {
		return usecase.Find{Registry: registry}.Execute(asked)
	}
	last, found, err := registry.Last()
	if err != nil {
		return domain.Vault{}, err
	}
	if found {
		return last, nil
	}
	held, err := usecase.List{Registry: registry}.Execute()
	if err != nil {
		return domain.Vault{}, err
	}
	if len(held) > 0 {
		return held[0], nil
	}
	return domain.Vault{}, nil
}

// Show puts a vault in the window: another in place of the one it has, or the
// first where it is standing on nothing. The index and the embedder belong to
// the installation and stay; what belongs to the vault is taken down and built
// again.
//
// Nothing is taken away until the vault asked for reads as a vault and every
// page has written what only it holds. A page holding text a person has to
// answer for calls the swap off, and the window stays on the vault it had.
//
// A vault that will not come up leaves the window on the one it was showing. A
// window neither of them comes up in stands on nothing and says so.
func (o *Opened) Show(ctx context.Context, v domain.Vault) error {
	if v.ID == o.API.Showing().ID {
		return nil
	}
	if err := readable(o.cfg, v); err != nil {
		return err
	}
	if err := o.alone(); err != nil {
		return err
	}
	defer o.free()

	// A page that says nothing is waited for HandedOverIn and no longer.
	held, spent := context.WithTimeout(ctx, HandedOverIn)
	defer spent()

	if !settling(held, &o.API.Leaving, &o.API.Writing) {
		return errAsking
	}

	was := o.API.Showing()
	o.leave()
	o.forget()

	err := o.arrive(v, false)
	if err != nil {
		if back := o.arrive(was, false); back != nil {
			// The window is standing on nothing: it says so, and the door on
			// writes stays shut.
			o.API.show(domain.Vault{})
			o.API.Failed.Store(back.Error())
			return errors.Join(err, back)
		}
	}
	// Writes are taken again: there is a vault to write in.
	o.API.Writing.open()
	// The round the settling was is over, and what a page holds from here is
	// this vault's.
	o.API.Leaving.over()
	// Everything a page is holding was read in a vault that is no longer in
	// front of it.
	o.API.Listeners.tell(changed{reload: true})
	return err
}

// arrive puts a vault in the window and builds everything that belongs to it.
//
// The zero vault is a window standing on nothing: it shows no vault, and no
// pass runs behind it.
func (o *Opened) arrive(v domain.Vault, rebuild bool) error {
	o.API.show(v)
	if v.ID == "" {
		// Nothing is being read, so nothing is waited for.
		o.API.Ready.Store(true)
		return nil
	}
	// Recorded before the vault is built, so the next window opens on it. A
	// list that could not be written is said and nothing more.
	if err := o.registry.Opened(v.ID); err != nil {
		fmt.Fprintf(o.out, "not recording %s as the vault opened: %v\n", v.Name, err)
	}
	on, err := o.begins(v, rebuild)
	if err != nil {
		return err
	}
	o.on.Store(on)
	return nil
}

// begins builds the half of the window that belongs to one vault: the scan and
// the watch behind it, the reading of the documents it holds, and the batches
// left with a proofreader.
func (o *Opened) begins(v domain.Vault, rebuild bool) (*showing, error) {
	known, err := usecase.List{Registry: o.registry}.Execute()
	if err != nil {
		return nil, err
	}

	watching, stop := context.WithCancel(o.under)
	recognising := o.cfg.Recognising(watching, o.Index.Sources(), o.tasks)

	// What a recognition writes down is cut where every other cut happens. A
	// document being read and a vault being scanned are then never two passes
	// over the index at once.
	owed := &pending{}
	recognising.Cut = func(_ context.Context, of domain.Vault, path string) error {
		owed.put(of, path)
		raise(o.wake.read)
		return nil
	}

	// A batch left with a proofreader outlives the run that left it, so one
	// left before the application closed is collected when it opens. Every
	// vault this installation holds is asked after.
	go recognising.Collecting(watching, o.Index.SourcesKnown(), collectedEvery, known...)

	scan := usecase.Scan{
		Readers:      o.cfg.VaultReaders(),
		Vaults:       o.Index.Vaults(),
		Notes:        o.Index.NotesCutAt(o.cfg.Cutting()),
		Known:        o.Index.Queries(),
		Maintenance:  o.Index.Maintenance(),
		RebuildIndex: rebuild,
	}

	// Following the vault is a use case; this adapter only says who hears about
	// it. Whatever a change turns out to mean is decided in one place, so a
	// second way of showing a vault does not decide it again.
	held := &holding{NoteRepository: o.Index.NotesCutAt(o.cfg.Cutting())}
	refresh := usecase.Refresh{Readers: o.cfg.VaultReaders(), Notes: held}
	follow := usecase.Follow{
		Watcher: o.cfg.VaultWatcher(),
		Refresh: refresh,
		Scan:    scan,
	}

	ended := begin(watching, v, o.cfg, o.Index, o.API, scan, follow,
		held, o.cfg.VaultReaders(), o.Embedder, o.wake, owed, o.out)

	return &showing{
		scan:        scan,
		refresh:     refresh,
		recognising: recognising,
		stop:        stop,
		ended:       ended,
	}, nil
}

// leave takes down the half of the window that belongs to the vault it is
// showing.
func (o *Opened) leave() {
	on := o.on.Swap(nil)
	if on == nil {
		return
	}
	on.stop()
	on.ended()
	// A reading writes to the index, so it ends before anything reads what it
	// wrote.
	on.recognising.Wait()
	// The documents held open go with the vault, and each gives back the worker
	// it was holding.
	if o.API.Viewer != nil {
		o.API.Viewer.empty()
	}
}

// forget is the vault that went leaving nothing of itself behind: what was said
// about reading it, and the entries its passes left in the list of what is
// being done.
//
// What stopped this installation from embedding at all is put back. It stands
// for as long as the window is open.
func (o *Opened) forget() {
	o.API.Ready.Store(false)
	o.API.Failed.Store("")
	o.API.Unwatched.Store("")

	for _, pass := range []string{walkingNotes, readingBooks, makingVectors, wordsAlone} {
		o.API.finished(pass)
	}
	if o.vectors != nil {
		o.API.say(task.Task{ID: makingVectors, Doing: "Indexing", Failed: o.vectors.Error()})
	}
}

// alone takes the window for one settling. A swap and a window closing both
// settle, one settling runs at a time, and the second to arrive is told so.
func (o *Opened) alone() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	switch {
	case o.going:
		return errGoing
	case o.busy:
		return errSettling
	}
	o.busy = true
	return nil
}

func (o *Opened) free() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.busy = false
}

// Settle is everything owed landing before anything is taken away. It is called
// while the window is still drawn, and calling it again is free. It answers
// false where a page is holding work a person is being asked about, and then
// nothing has been taken away and the vault is as it was.
//
// A vault being opened settles too, and the close that arrives while it is
// running is answered false: the window stays, and the next ask settles again.
func (o *Opened) Settle(ctx context.Context) bool {
	o.mu.Lock()
	if o.going {
		o.mu.Unlock()
		return true
	}
	if o.busy {
		o.mu.Unlock()
		return false
	}
	o.busy = true
	o.mu.Unlock()

	settled := settling(ctx, &o.API.Leaving, &o.API.Writing)

	o.mu.Lock()
	defer o.mu.Unlock()
	o.busy = false
	// A window that settled to go shows no other vault.
	o.going = settled
	return settled
}

// Answered is every page having written what it owes. It is what the window
// waits on while a person answers a question, and that wait is on a person and
// is not measured. It answers false where ctx ended or the vault was asked
// again.
func (o *Opened) Answered(ctx context.Context) bool { return answering(ctx, &o.API.Leaving) }

// Close stops the passes behind the vault, waits for them, and closes the
// index.
func (o *Opened) Close() error {
	o.leave()
	if o.API.Viewer != nil {
		o.API.Viewer.close()
	}
	// The embedder goes after the work that uses it and before the database,
	// which is the order they depend on each other in.
	err := o.stopEmbedder()
	if closed := o.Index.Close(); err == nil {
		err = closed
	}
	return err
}

// Showing is the vault the window has open.
func (o *Opened) Showing() domain.Vault { return o.API.Showing() }

// Refresh brings named notes up to date. Whatever changes a note calls it, so
// that what changed is findable before the change is reported done.
func (o *Opened) Refresh() usecase.Refresh {
	if on := o.on.Load(); on != nil {
		return on.refresh
	}
	return usecase.Refresh{
		Readers: o.cfg.VaultReaders(),
		Notes:   o.Index.NotesCutAt(o.cfg.Cutting()),
	}
}

// Recognising reads a scanned document for whoever asks. It is one job for the
// window and for an agent alike, so that what a person started through one of
// them is shown by the other. Nothing while the window has no vault.
func (o *Opened) Recognising() *container.Recognising {
	if on := o.on.Load(); on != nil {
		return on.recognising
	}
	return nil
}

// level brings named notes up to date in the index, through whatever is
// following the vault they are in. A note the window makes is level before the
// answer comes back, so it is drawn as soon as it exists.
func (o *Opened) level(ctx context.Context, v domain.Vault, paths []string) error {
	_, err := o.Refresh().Execute(ctx, v, paths)
	return err
}

// scanning reads the whole vault.
func (o *Opened) scanning(ctx context.Context, v domain.Vault) (usecase.ScanResult, error) {
	on := o.on.Load()
	if on == nil {
		return usecase.ScanResult{}, errNoVault
	}
	return on.scan.Execute(ctx, v)
}

// readable is the vault being one this window can show: the folder reads as a
// vault, and it carries the identity the list has for it.
func readable(cfg container.Config, v domain.Vault) error {
	identity := cfg.VaultIdentity()
	if err := identity.Readable(v.Path); err != nil {
		return fmt.Errorf("%w: %w", usecase.ErrUnreadable, err)
	}
	carried, found, err := identity.Of(v.Path)
	if err != nil {
		return err
	}
	if !found || carried != v.ID {
		return fmt.Errorf("%w: %s is no longer the vault %s", usecase.ErrUnreadable, v.Path, v.Name)
	}
	return nil
}

// filedUnder is the extension a note this vault holds is filed under. Empty is
// markdown.
func filedUnder(cfg container.Config) string {
	if len(cfg.Extensions) > 0 {
		return cfg.Extensions[0]
	}
	return ""
}

// settling is everything owed landing: every client writes what only it holds,
// and then the writes already taken finish. It answers with whether the vault
// settled.
//
// A client that says nothing is bounded by ctx: what it owes sits in a webview
// this process cannot reach into. A client raising a question has said
// something, and the round ends on it: the vault stays open, the door stays
// open, and the wait from there is on a person.
//
// The writes are not bounded, because the door is shut first and what is left
// is a fixed set of filesystem operations.
func settling(ctx context.Context, pages *leaving, writes *inflight) bool {
	round := pages.ask()
	select {
	case <-round.written:
	case <-round.questions:
	case <-round.over:
	case <-ctx.Done():
	}
	if round.standing() || pages.current() != round {
		return false
	}
	<-writes.seal()
	return true
}

// answering waits for the round in progress to end with every page having
// written what it owes.
func answering(ctx context.Context, pages *leaving) bool {
	round := pages.current()
	if round == nil {
		return false
	}
	select {
	case <-round.written:
		return true
	case <-round.over:
		return false
	case <-ctx.Done():
		return false
	}
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
)

// Says puts what reading the settings had to tell a person in the list of what
// is being done, which is where a person is.
func (o *Opened) Says(said []string) {
	for at, one := range said {
		o.API.say(task.Task{
			ID:     fmt.Sprintf("%s %d", readingTheSettings, at),
			Doing:  "Reading the settings",
			Failed: one,
		})
	}
}

// HandedOverIn is how long a page has to write what only it holds when the
// vault it is drawing goes. A page raising a question has answered, and the
// wait from there is on a person.
const HandedOverIn = 3 * time.Second

// settled is how long the vault has to have been still before the notes written
// into it are embedded. It is longer than the bound in ui/src/tab.ts, which
// writes an unfinished edit every five seconds while a person goes on typing.
const settled = 8 * time.Second

// collectedEvery is how often the batches left with a proofreader are asked
// after. A batch is answered in hours.
const collectedEvery = 5 * time.Minute

// nudges are the two ways work reaches the reading behind the window once the
// first pass is over: a book, which is found and cut before anything is
// embedded, and a note, which arrives already cut and owes only its vectors.
type nudges struct {
	sources chan struct{}
	notes   chan struct{}
	// read is one source with more text than its chunks account for, which is
	// what a recognition leaves behind every batch of pages.
	read chan struct{}
	// still is how long the vault has to have been quiet before a note that was
	// written is embedded.
	still time.Duration
}

func waking(still time.Duration) nudges {
	return nudges{
		sources: make(chan struct{}, 1),
		notes:   make(chan struct{}, 1),
		read:    make(chan struct{}, 1),
		still:   still,
	}
}

// pending is the sources a recognition has written more of than their chunks
// account for.
//
// One document is read at a time and a document asks many times over, so this
// holds a source once however many batches it wrote. It holds more than one
// because a nudge can be dropped while the pass is busy, and a document that
// finished while another was being asked for is a book cut to the page it
// reached.
//
// It belongs to the vault that was being read, and goes with it: a path read in
// one vault is not cut under the vault that arrives.
type pending struct {
	mu    sync.Mutex
	paths map[string]domain.Vault
}

func (p *pending) put(v domain.Vault, path string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.paths == nil {
		p.paths = map[string]domain.Vault{}
	}
	p.paths[path] = v
}

// take is everything waiting, and leaves nothing behind.
func (p *pending) take() map[string]domain.Vault {
	p.mu.Lock()
	defer p.mu.Unlock()
	held := p.paths
	p.paths = nil
	return held
}

// raise leaves one nudge waiting. What owes work is asked of the index, so
// several changes at once are one pass.
func raise(nudge chan struct{}) {
	select {
	case nudge <- struct{}{}:
	default:
	}
}

// begin starts the watch and the first scan together, and answers with what
// waits for both to stop.
//
// The watch is acted on alongside the scan: a change reaches the window through
// it and through nothing else.
//
// A vault that cannot be watched is still a vault: the application keeps
// working, and says that changes will not appear by themselves.
func begin(
	ctx context.Context,
	v domain.Vault,
	cfg container.Config,
	db *container.Index,
	api *API,
	scan usecase.Scan,
	follow usecase.Follow,
	held *holding,
	readers port.VaultReaders,
	embedder port.Embedder,
	wake nudges,
	owed *pending,
	out io.Writer,
) func() {
	trouble := func(err error) {
		if err == nil {
			api.Failed.Store("")
			return
		}
		api.Failed.Store(err.Error())
	}
	follow.Trouble = trouble
	follow.Changed = func(m usecase.Moved) {
		api.Listeners.tell(changed{paths: m.Paths, reload: m.Reload})
		if m.Sources {
			// A book dropped into an open vault is read without anybody asking.
			raise(wake.sources)
		}
		if len(m.Paths) > 0 {
			// A note written is a note cut again, and its chunks owe their
			// vectors. Which ones is not carried: the debt is in the index.
			raise(wake.notes)
		}
	}

	// Said before the goroutine starts, so a window that opens on a fresh vault
	// is shown the walk from its first moment.
	api.say(task.Task{ID: walkingNotes, Doing: "Reading the vault"})

	var running sync.WaitGroup

	watch, err := follow.Begin(ctx, v)
	if err != nil {
		fmt.Fprintf(out, "not watching %s: %v\n", v.Name, err)
		api.Unwatched.Store(err.Error())
	}
	if watch != nil {
		running.Add(1)
		go func() {
			defer running.Done()

			watch.Run(ctx)
			// Nothing reaches the window once the watch stops, so from here on
			// the vault is one that is not being followed.
			if ctx.Err() == nil {
				api.Unwatched.Store("the watch stopped")
			}
		}()
	}

	// first is the vault's first reading: the scan, and the notes written while
	// it ran read once more. It answers whether the vault was read.
	first := func() bool {
		defer api.finished(walkingNotes)

		// The walk a person watches is this one. A later one is the index being
		// brought level with a vault that moved under it.
		walk := scan
		walk.OnProgress = func(res usecase.ScanResult) {
			api.say(task.Task{ID: walkingNotes, Doing: "Reading the vault", Done: int64(res.Indexed)})
		}
		result, err := walk.Execute(ctx, v)

		// The scan writes in groups from what it read, so its copy of a note
		// lands last however early the note was read. Every note brought up to
		// date underneath it is read once more, and the newest copy of each
		// lands last.
		under := held.taken()

		switch {
		case err == nil:
		case errors.Is(err, context.Canceled):
			// Asked to stop. What it stored is correct as far as it got, and
			// there is nothing to report.
			return false
		default:
			api.Failed.Store(err.Error())
			return false
		}

		if len(under) > 0 {
			if _, err := follow.Refresh.Execute(ctx, v, under); err != nil {
				trouble(err)
			}
		}

		fmt.Fprintf(out, "%s: %d notes\n", v.Name, result.Seen)
		api.Ready.Store(true)
		return true
	}

	running.Add(1)
	go func() {
		defer running.Done()

		// Reading the sources comes after the notes: a vault is useful the
		// moment its notes answer, and a library takes minutes to cut and hours
		// to embed. Neither stops the window, and neither has to finish: an
		// index is a cache.
		if first() {
			readSources(ctx, cfg, db, api, v, readers, embedder, out)
		}

		// What arrives while the window is open is read where the first reading
		// was: one at a time, and never while another is running. A vault whose
		// first reading failed is one somebody goes on writing in, so the
		// waiting stands whatever that reading did.
		var quiet <-chan time.Time
		for {
			select {
			case <-ctx.Done():
				return
			case <-wake.sources:
				readSources(ctx, cfg, db, api, v, readers, embedder, out)
			case <-wake.read:
				// A batch of pages is on disk. What has been read of the
				// document is cut and embedded while the rest of it is still
				// being read.
				if held := owed.take(); len(held) > 0 {
					for path, of := range held {
						cutSource(ctx, cfg, db, api, embedder, of, path)
					}
					embedSources(ctx, cfg, db, api, v, readers, embedder)
				}
			case <-wake.notes:
				// Every write puts the pass off again: what was typed is
				// embedded once the vault has been still.
				quiet = time.After(wake.still)
			case <-quiet:
				quiet = nil
				embedSources(ctx, cfg, db, api, v, readers, embedder)
			}
		}
	}()

	return running.Wait
}

// holding is the index, keeping the paths of the notes written through it while
// the first scan is still reading the vault.
type holding struct {
	port.NoteRepository

	mu    sync.Mutex
	paths []string
	kept  map[string]bool
	over  bool
}

func (h *holding) Save(ctx context.Context, vaultID string, notes []domain.Note) error {
	for _, n := range notes {
		h.hold(n.Ref.Path)
	}
	return h.NoteRepository.Save(ctx, vaultID, notes)
}

func (h *holding) Remove(ctx context.Context, vaultID string, paths []string) error {
	for _, path := range paths {
		h.hold(path)
	}
	return h.NoteRepository.Remove(ctx, vaultID, paths)
}

// hold takes the path before the write it belongs to, so a note whose write
// lands while the scan is still running is one of the paths taken after it.
func (h *holding) hold(path string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.over || h.kept[path] {
		return
	}
	if h.kept == nil {
		h.kept = map[string]bool{}
	}
	h.kept[path] = true
	h.paths = append(h.paths, path)
}

// taken is every path held, and the end of the holding: the scan is over, so a
// write that lands from now on is already the last one.
func (h *holding) taken() []string {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.over = true
	paths := h.paths
	h.paths, h.kept = nil, nil
	return paths
}

// readSources takes the text out of every book in the vault and then embeds what
// was cut, reporting what it is reading as it goes.
//
// Every way is allowed to fail without the window minding. A book that will
// not parse is one book; an embedder that is not configured is the ordinary case,
// and search answers on words alone until one is.
func readSources(
	ctx context.Context,
	cfg container.Config,
	db *container.Index,
	api *API,
	v domain.Vault,
	readers port.VaultReaders,
	embedder port.Embedder,
	out io.Writer,
) {
	making, err := cfg.Searchable(ctx, db, embedder, v)
	if err != nil {
		api.say(task.Task{ID: readingBooks, Doing: "Reading books", Failed: err.Error()})
		return
	}
	making.Books.OnProgress = func(res source.ExtractResult) {
		api.say(task.Task{
			ID: readingBooks, Doing: "Reading books", About: res.Reading,
			// Every book the walk found leaves this pass one of four ways, and
			// all four count as done.
			Done:  int64(res.Extracted + res.Unchanged + res.Unreadable + res.Vanished),
			Total: int64(res.Seen),
		})
	}

	api.say(task.Task{ID: readingBooks, Doing: "Reading books"})
	res, read := making.ReadBooks(ctx, v)
	switch {
	case read == nil:
		if res.Extracted > 0 {
			fmt.Fprintf(out, "%s: %d books, %d chunks\n", v.Name, res.Extracted, res.Chunks)
		}
		api.finished(readingBooks)
	case errors.Is(read, context.Canceled):
		// Asked to stop. What it cut is correct as far as it got.
		api.finished(readingBooks)
	default:
		// A failed pass stays in the list until whoever is shown it takes it
		// out.
		api.say(task.Task{ID: readingBooks, Doing: "Reading books", Failed: read.Error()})
	}

	embedSources(ctx, cfg, db, api, v, readers, embedder)
}

// cutSource cuts one source again from whatever its text now says.
//
// A recognition writes a batch of pages and asks for this, so a book being read
// answers questions about the pages that have been read. Failing is one source:
// the next batch asks again.
func cutSource(
	ctx context.Context,
	cfg container.Config,
	db *container.Index,
	api *API,
	embedder port.Embedder,
	v domain.Vault,
	path string,
) {
	cut := func(err error) {
		api.say(task.Task{ID: readingBooks, Doing: "Reading books", About: path, Failed: err.Error()})
	}

	making, err := cfg.Searchable(ctx, db, embedder, v)
	if err != nil {
		cut(err)
		return
	}
	switch err := making.CutOne(ctx, v, path); {
	case err == nil:
		// The pages that were read are cut, and a cut that failed before this
		// one is over.
		api.finished(readingBooks)
	case !errors.Is(err, context.Canceled):
		cut(err)
	}
}

// embedSources gives the chunks of the vault the vectors they owe, and reads no
// file the index does not already hold a chunk of.
//
// It is the whole of what a note that was written owes: the chunks are cut
// where the note is stored, and what has no vector is a question for the index.
func embedSources(
	ctx context.Context,
	cfg container.Config,
	db *container.Index,
	api *API,
	v domain.Vault,
	readers port.VaultReaders,
	embedder port.Embedder,
) {
	if embedder == nil {
		return
	}

	// What this pass owes, asked once before it starts: the chunks that can
	// carry a vector and do not. The pass finds them a few hundred at a time,
	// and a total that grows as it goes is a count that never settles.
	owing := int64(0)
	if api.Progress != nil {
		if held, embedded, err := api.Progress.Progress(ctx, v.ID, text(&api.Recipe)); err == nil {
			owing = max(0, held-embedded)
		}
	}

	indexing := func(err error) {
		api.say(task.Task{ID: makingVectors, Doing: "Indexing", Failed: err.Error()})
	}

	making, err := cfg.Searchable(ctx, db, embedder, v)
	if err != nil {
		indexing(err)
		return
	}
	making.Vectors.OnProgress = func(res source.EmbedResult) {
		// A person who edited one note is waiting on that note, so this is the
		// work in hand and not the size of the vault.
		api.say(task.Task{
			ID: makingVectors, Doing: "Indexing",
			Done: int64(res.Embedded), Total: owing,
		})
	}

	// This pass says what it owes and what it has made. The source a vector is
	// made from is named by the reading of that source.
	api.say(task.Task{ID: makingVectors, Doing: "Indexing", Total: owing})
	switch _, err := making.MakeVectors(ctx, v); {
	case err == nil, errors.Is(err, context.Canceled):
		api.finished(makingVectors)
	default:
		// A vault short of the vectors it owes is searched by its words alone,
		// and the reason for it stands in the list.
		indexing(err)
	}
}
