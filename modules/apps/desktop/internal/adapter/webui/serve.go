package webui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/task"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/note"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/search"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/source"
	usecase "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/vault"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/window"
)

// Opened is a vault put together and running: the questions a client may ask,
// and the pieces anything else working the same vault needs.
type Opened struct {
	API   *API
	Vault domain.Vault
	Index *container.Index
	// Refresh brings named notes up to date. Whatever changes a note calls it,
	// so that what changed is findable before the change is reported done.
	Refresh usecase.Refresh
	// Recognising reads a scanned document for whoever asks. It is one job for
	// the window and for an agent alike, so that what a person started through
	// one of them is shown by the other.
	Recognising *container.Recognising

	// Embedder turns text into vectors, for filling the index and for turning a
	// query into one. It is the same embedder for both, so a query's vector and
	// the stored vectors come from one model. Nil for an installation with none,
	// and then every search is answered by words alone.
	Embedder port.Embedder
	// Settle is everything owed landing before anything is taken away. It is
	// called while the window is still drawn, and calling it again is free. It
	// answers false where a page is holding work a person is being asked about,
	// and then nothing has been taken away and the vault is as it was.
	Settle func(ctx context.Context) bool
	// Answered is every page having written what it owes. It is what the window
	// waits on while a person answers a question, and that wait is on a person
	// and is not measured. It answers false where ctx ended or the vault was
	// asked again.
	Answered func(ctx context.Context) bool
	// Close stops the scan, waits for it, and closes the index.
	Close func() error
}

// Open puts together everything the window needs: the vault it shows, the
// questions it may ask, and a scan running behind it.
//
// The scan is started and left running. A vault of a hundred thousand notes
// takes a minute and a half, and the first note is answerable long before that.
//
// Going takes two steps. Settle is called while the window is still drawn: the
// clients write what only they hold and the writes in the air land. Closing
// then stops the scan, waits for it, and closes the database — in that order,
// because the database is what the scan writes to.
func Open(ctx context.Context, cfg container.Config, out io.Writer) (*Opened, error) {
	registry, err := cfg.Registry()
	if err != nil {
		return nil, err
	}
	vaults, err := usecase.List{Registry: registry}.Execute()
	if err != nil {
		return nil, err
	}
	if len(vaults) == 0 {
		made, err := firstVault(cfg, registry)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(out, "%s: a vault to write in, at %s\n", made.Name, made.Path)
		vaults = []domain.Vault{made}
	}

	db, err := cfg.OpenIndex(ctx)
	if err != nil {
		return nil, err
	}

	// Opened once, for as long as the window is. A local model holds a session
	// that takes seconds to build, and both filling the index and answering a
	// query need it.
	embedder, closeEmbedder, why := cfg.Embedder()
	if why != nil {
		fmt.Fprintf(out, "not embedding %s: %v\n", vaults[0].Name, why)
	}
	if closeEmbedder == nil {
		closeEmbedder = func() error { return nil }
	}

	wake := waking(settled)

	watching, stop := context.WithCancel(ctx)
	// One list of what is being done, for everything that does anything and for
	// the window that shows it. One job behind it too: what a person asked for
	// is one piece of work however they asked for it.
	tasks := task.New()
	recognising := cfg.Recognising(db.Sources(), tasks)

	api := &API{
		Vault:     vaults[0],
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
	}
	// Named before anything is read: it is what decides whether a chunk already
	// carries a vector, and what tells the window that something is going to
	// embed what was cut.
	if embedder != nil {
		model := embedder.Model()
		// The coarse index is built for one width. A model of another width
		// rebuilds it from what has been made.
		if err := db.FitVectors(ctx, model.Dimensions); err != nil {
			fmt.Fprintf(out, "not embedding %s: %v\n", vaults[0].Name, err)
			embedder = nil
		} else {
			api.Model.Store(model.String())
			api.Recipe.Store(model.Recipe())
		}
	}

	// The search the window offers is the search the application already does.
	// It is built once the embedder is settled, so a model that could not be
	// fitted leaves the words half to answer on its own.
	finds := search.New(db.Passages(), cfg.VaultReaders(), cfg.DerivedStores(), embedder, cfg.Embedding.Floor)
	api.Finds = &finds
	scan := usecase.Scan{
		Readers:      cfg.VaultReaders(),
		Vaults:       db.Vaults(),
		Notes:        db.Notes(),
		Known:        db.Queries(),
		Maintenance:  db.Maintenance(),
		RebuildIndex: cfg.RebuildIndex,
		OnProgress: func(res usecase.ScanResult) {
			api.Indexed.Store(int64(res.Indexed))
		},
	}
	api.Scan = scan.Execute

	// Following the vault is a use case; this adapter only says who hears about
	// it. Whatever a change turns out to mean is decided in one place, so a
	// second way of showing a vault does not decide it again.
	held := &holding{NoteRepository: db.Notes()}
	refresh := usecase.Refresh{Readers: cfg.VaultReaders(), Notes: held}

	// A note the window makes is level in the index before the answer comes
	// back, so it is drawn as soon as it exists.
	level := func(ctx context.Context, v domain.Vault, paths []string) error {
		_, err := refresh.Execute(ctx, v, paths)
		return err
	}
	api.Makes = &note.Create{
		Writers:   cfg.VaultWriters(),
		Names:     db.Queries(),
		Index:     level,
		Extension: filedUnder(cfg),
	}
	api.Joins = &note.Linking{
		Readers: cfg.VaultReaders(),
		Writers: cfg.VaultWriters(),
		Index:   level,
	}

	follow := usecase.Follow{
		Watcher: cfg.VaultWatcher(),
		Refresh: refresh,
		Scan:    scan,
	}

	wait := begin(watching, cfg, db, api, scan, follow, held, cfg.VaultReaders(), embedder, wake, out)

	return &Opened{
		API:         api,
		Vault:       api.Vault,
		Index:       db,
		Refresh:     refresh,
		Recognising: recognising,
		Embedder:    embedder,
		Settle:      func(ctx context.Context) bool { return settling(ctx, &api.Leaving, &api.Writing) },
		Answered:    func(ctx context.Context) bool { return answering(ctx, &api.Leaving) },
		Close: func() error {
			stop()
			wait()
			// The embedder goes after the work that uses it and before the
			// database, which is the order they depend on each other in.
			err := closeEmbedder()
			if closed := db.Close(); err == nil {
				err = closed
			}
			return err
		},
	}, nil
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

// settled is how long the vault has to have been still before the notes written
// into it are embedded. It is longer than the bound in ui/src/tab.ts, which
// writes an unfinished edit every five seconds while a person goes on typing.
const settled = 8 * time.Second

// nudges are the two ways work reaches the reading behind the window once the
// first pass is over: a book, which is found and cut before anything is
// embedded, and a note, which arrives already cut and owes only its vectors.
type nudges struct {
	sources chan struct{}
	notes   chan struct{}
	// still is how long the vault has to have been quiet before a note that was
	// written is embedded.
	still time.Duration
}

func waking(still time.Duration) nudges {
	return nudges{
		sources: make(chan struct{}, 1),
		notes:   make(chan struct{}, 1),
		still:   still,
	}
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
	cfg container.Config,
	db *container.Index,
	api *API,
	scan usecase.Scan,
	follow usecase.Follow,
	held *holding,
	readers port.VaultReaders,
	embedder port.Embedder,
	wake nudges,
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

	// Set before the goroutine starts, so that a client asking between opening
	// and the first read is told there is more to come.
	api.Busy.Store(true)

	var running sync.WaitGroup

	watch, err := follow.Begin(ctx, api.Vault)
	if err != nil {
		fmt.Fprintf(out, "not watching %s: %v\n", api.Vault.Name, err)
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
		result, err := scan.Execute(ctx, api.Vault)
		api.Indexed.Store(int64(result.Indexed))

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
			if _, err := follow.Refresh.Execute(ctx, api.Vault, under); err != nil {
				trouble(err)
			}
		}

		fmt.Fprintf(out, "%s: %d notes\n", api.Vault.Name, result.Seen)
		api.Ready.Store(true)
		return true
	}

	running.Add(1)
	go func() {
		defer running.Done()
		// Set false on every way out of the reading.
		defer api.Busy.Store(false)
		// Set false on every way out of the reading, and embedding is part of it.
		defer api.Learning.Store(false)

		// Reading the sources comes after the notes: a vault is useful the
		// moment its notes answer, and a library takes minutes to cut and hours
		// to embed. Neither stops the window, and neither has to finish: an
		// index is a cache.
		if first() {
			readSources(ctx, cfg, db, api, readers, embedder, out)
		}
		// Everything this vault owed is read.
		api.Busy.Store(false)

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
				api.Busy.Store(true)
				readSources(ctx, cfg, db, api, readers, embedder, out)
				api.Busy.Store(false)
			case <-wake.notes:
				// Every write puts the pass off again. What was typed is
				// embedded once the vault has been still, and there is work
				// to come from the moment the write lands.
				if text(&api.Model) != "" {
					api.Busy.Store(true)
					api.Learning.Store(true)
				}
				quiet = time.After(wake.still)
			case <-quiet:
				quiet = nil
				api.Busy.Store(true)
				embedSources(ctx, db, api, readers, embedder, out)
				api.Busy.Store(false)
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
// Both halves are allowed to fail without the window minding. A book that will
// not parse is one book; an embedder that is not configured is the ordinary case,
// and search answers on words alone until one is.
func readSources(
	ctx context.Context,
	cfg container.Config,
	db *container.Index,
	api *API,
	readers port.VaultReaders,
	embedder port.Embedder,
	out io.Writer,
) {
	// A window is cut under the limit of the model that will read it. Without a
	// model the default bound stands: what is cut now is what a model of any width
	// is later given.
	sizes := window.Sizes{}
	if embedder != nil {
		sizes.Limit = window.Under(embedder.Model().MaxTokens)
	}

	derived, err := cfg.DerivedStores().Open(api.Vault)
	if err != nil {
		fmt.Fprintf(out, "reading the sources of %s: %v\n", api.Vault.Name, err)
		return
	}
	extract := source.Extract{
		Readers:      readers,
		Sources:      db.Sources(),
		Owing:        db.SourcesKnown(),
		Derived:      derived,
		Sizes:        sizes,
		RebuildIndex: cfg.RebuildIndex,
		OnProgress: func(res source.ExtractResult) {
			api.Reading.Store(res.Reading)
			api.Books.Store(int64(res.Seen))
			// Every book the walk found leaves this pass one of four ways, and
			// all four count as done.
			api.BooksRead.Store(int64(res.Extracted + res.Unchanged + res.Unreadable + res.Vanished))
		},
	}
	if res, err := extract.Execute(ctx, api.Vault); err != nil {
		if !errors.Is(err, context.Canceled) {
			fmt.Fprintf(out, "reading the sources of %s: %v\n", api.Vault.Name, err)
		}
	} else if res.Extracted > 0 {
		fmt.Fprintf(out, "%s: %d books, %d chunks\n", api.Vault.Name, res.Extracted, res.Chunks)
	}
	api.Reading.Store("")
	api.Owed.Store(&Owed{})

	embedSources(ctx, db, api, readers, embedder, out)
}

// embedSources gives the chunks of the vault the vectors they owe, and reads no
// file the index does not already hold a chunk of.
//
// It is the whole of what a note that was written owes: the chunks are cut
// where the note is stored, and what has no vector is a question for the index.
func embedSources(
	ctx context.Context,
	db *container.Index,
	api *API,
	readers port.VaultReaders,
	embedder port.Embedder,
	out io.Writer,
) {
	if embedder == nil {
		return
	}

	// What this pass owes, asked once before it starts: the chunks that can
	// carry a vector and do not. The pass finds them a few hundred at a time,
	// and a total that grows as it goes is a count that never settles.
	owing := int64(0)
	if api.Progress != nil {
		if held, embedded, err := api.Progress.Progress(ctx, api.Vault.ID, text(&api.Recipe)); err == nil {
			owing = max(0, held-embedded)
		}
	}

	embed := source.Embed{
		Readers:  readers,
		Chunks:   db.VectorsOwing(),
		Vectors:  db.Vectors(),
		Embedder: embedder,
		OnProgress: func(res source.EmbedResult) {
			api.Reading.Store(res.Reading)
			// A person who edited one note is waiting on that note, so this is
			// the work in hand and not the size of the vault.
			api.Owed.Store(&Owed{Owing: owing, Made: int64(res.Embedded)})
		},
	}
	api.Owed.Store(&Owed{Owing: owing})
	api.Learning.Store(true)
	if _, err := embed.Execute(ctx, api.Vault); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintf(out, "embedding %s: %v\n", api.Vault.Name, err)
	}
	api.Learning.Store(false)
	api.Reading.Store("")
	api.Owed.Store(&Owed{})
}

// Showing is the vault the window has open.
func (a *API) Showing() domain.Vault { return a.Vault }

// firstVault is somewhere to write, made for a person who has added nothing.
//
// An application that answers "you have no vault" and stops is one that asks
// somebody to read its manual before it will do anything. It makes a folder
// where this system keeps documents, and that folder is theirs: ordinary files
// in an ordinary place, which they may move or replace with one of their own.
func firstVault(cfg container.Config, registry port.VaultRegistry) (domain.Vault, error) {
	documents, err := filesystem.Documents()
	if err != nil {
		return domain.Vault{}, err
	}
	root := filepath.Join(documents, defaultVaultName)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return domain.Vault{}, err
	}
	// A folder that is already a vault keeps the identity it has.
	if _, err := filesystem.Initialize(root, cfg.VaultOptions().ServiceDir, time.Now()); err != nil {
		return domain.Vault{}, err
	}
	return usecase.Add{
		Identity: cfg.VaultIdentity(),
		Registry: registry,
		Now:      time.Now,
	}.Execute(root, defaultVaultName)
}

// defaultVaultName is what the first vault is called, and what its folder is
// named inside the documents folder.
const defaultVaultName = "numen"
