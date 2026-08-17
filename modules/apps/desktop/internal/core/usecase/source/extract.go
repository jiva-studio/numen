package source

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/epub"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/window"
)

// extractor names the reader that takes text out of a book. It is part of the
// recipe, and it changes when the text or the offsets the reader produces do.
const extractor = "epub-1"

// sourcesPerQuery bounds one answer about what owes its text, so that a library
// of any size is worked through in pieces.
const sourcesPerQuery = 100

// Extract turns the books a vault holds into chunks.
//
// It runs in two passes. The first records what the vault has, opening nothing.
// The second takes the text out of every source that owes it and cuts it into
// windows, one write per source.
type Extract struct {
	Readers port.VaultReaders
	Sources port.SourceRepository
	Owing   port.SourceQueries

	// Sizes are how the text is cut. They are named in the recipe, so a source
	// cut at other sizes owes its text again.
	Sizes window.Sizes

	// OnProgress, if set, is called as each source is opened and as each one is
	// written. A library takes minutes, and something has to be able to say how
	// far it has got.
	OnProgress func(ExtractResult)
}

// ExtractResult reports what extraction did.
//
// `Seen` counts books and nothing else. `Recorded` and `Extracted` are separate
// numbers because they are separate passes: a book is recorded when the vault is
// walked and extracted when its text is read.
type ExtractResult struct {
	Seen       int    // books found in the vault
	Recorded   int    // new or changed, so owing their text
	Unchanged  int    // skipped on size and modification time alone
	Extracted  int    // read, cut and written
	Chunks     int    // windows written, of both sizes
	Unreadable int    // read, and not a book
	Removed    int    // held by the index, no longer in the vault
	Vanished   int    // named by the index, gone by the time it was read
	Reading    string // the source open now
}

// Execute brings the chunks of one vault's books up to date with what is on
// disk.
func (u Extract) Execute(ctx context.Context, v domain.Vault) (ExtractResult, error) {
	var res ExtractResult

	reader, err := u.Readers.Open(v)
	if err != nil {
		return res, err
	}
	if err := u.discover(ctx, v, reader, &res); err != nil {
		return res, err
	}
	if err := u.cut(ctx, v, reader, &res); err != nil {
		return res, err
	}
	return res, nil
}

// discover records every book the vault holds. Nothing is opened: a file whose
// size and modification time are what the index believes is left alone, and one
// that differs is recorded with no recipe, which is what owing its text means.
func (u Extract) discover(ctx context.Context, v domain.Vault, reader port.VaultReader, res *ExtractResult) error {
	known, err := u.Owing.Fingerprints(ctx, v.ID, domain.KindBook)
	if err != nil {
		return fmt.Errorf("read index: %w", err)
	}

	found := make(map[string]bool, len(known))
	if err := reader.Walk(ctx, func(ref domain.FileRef) error {
		if ref.Kind != domain.KindBook {
			return nil
		}
		res.Seen++
		found[ref.Path] = true
		if previous, ok := known[ref.Path]; ok && previous.Unchanged(ref) {
			res.Unchanged++
			return nil
		}
		if err := u.Sources.SaveSource(ctx, v.ID, port.Source{Ref: ref}); err != nil {
			return fmt.Errorf("record %s: %w", ref.Path, err)
		}
		res.Recorded++
		return nil
	}); err != nil {
		return err
	}

	// What the index holds and the walk did not find is gone from the vault. It
	// is taken out here rather than left: the text of a passage is read from the
	// file, so a source that is not there answers a search with nothing and the
	// row only wastes the coarse pass.
	gone := make([]string, 0)
	for path := range known {
		if !found[path] {
			gone = append(gone, path)
		}
	}
	if len(gone) == 0 {
		return nil
	}
	slices.Sort(gone)
	if err := u.Sources.RemoveSources(ctx, v.ID, domain.KindBook, gone); err != nil {
		return fmt.Errorf("remove: %w", err)
	}
	res.Removed = len(gone)
	return nil
}

// cut extracts and cuts every source that owes its text.
//
// Which those are is asked of the data, on both keys that answer it: a source
// with no small window has never been cut, and a source carrying another recipe
// was cut by another extractor or at other sizes. Each question is asked again
// until it names nothing that has not been tried, so a run that stopped part way
// is continued by starting another.
func (u Extract) cut(ctx context.Context, v domain.Vault, reader port.VaultReader, res *ExtractResult) error {
	sizes := u.sizes()
	recipe := recipe(sizes)
	tried := map[string]bool{}

	questions := []func(context.Context) ([]string, error){
		func(ctx context.Context) ([]string, error) {
			return u.Owing.Unchunked(ctx, v.ID, domain.KindBook, sourcesPerQuery)
		},
		func(ctx context.Context) ([]string, error) {
			return u.Owing.ByOtherRecipe(ctx, v.ID, domain.KindBook, recipe, sourcesPerQuery)
		},
	}
	for _, ask := range questions {
		for {
			paths, err := ask(ctx)
			if err != nil {
				return err
			}
			fresh := 0
			for _, path := range paths {
				if tried[path] {
					continue
				}
				tried[path] = true
				fresh++
				if err := ctx.Err(); err != nil {
					return err
				}
				if err := u.source(ctx, v, reader, path, sizes, recipe, res); err != nil {
					return err
				}
			}
			if fresh == 0 {
				break
			}
		}
	}
	return nil
}

// source takes the text out of one book and writes the windows it was cut into.
//
// A file that will not parse is counted and the run goes on. Extraction never
// refuses: no structure is an ordinary outcome, and one bad book must not stop a
// library.
//
// The file is stated before it is read, so a file that changes while it is being
// read keeps the older fingerprint and is asked for again.
func (u Extract) source(
	ctx context.Context,
	v domain.Vault,
	reader port.VaultReader,
	path string,
	sizes window.Sizes,
	recipe string,
	res *ExtractResult,
) error {
	res.Reading = path
	u.progress(*res)

	ref, err := reader.Stat(ctx, path)
	if errors.Is(err, fs.ErrNotExist) {
		res.Vanished++
		return nil
	}
	if err != nil {
		return fmt.Errorf("stat %s: %w", path, err)
	}
	raw, err := reader.Read(ctx, path)
	if errors.Is(err, fs.ErrNotExist) {
		res.Vanished++
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	book, err := epub.Read(raw)
	if err != nil {
		res.Unreadable++
		return nil
	}

	windows := windowsOf(book, sizes)
	extraction := port.Extraction{
		Source: port.Source{
			Ref:    ref,
			Hash:   fingerprint(raw),
			Recipe: recipe,
		},
		Windows: windows,
	}
	if err := u.Sources.SaveExtraction(ctx, v.ID, extraction); err != nil {
		return fmt.Errorf("write the chunks of %s: %w", path, err)
	}
	res.Extracted++
	res.Chunks += counted(windows)
	u.progress(*res)
	return nil
}

// windowsOf cuts one book's text: the large windows a result shows, each holding
// the small windows that carry a vector.
func windowsOf(book *epub.Book, sizes window.Sizes) []port.Window {
	places := make([]window.Place, 0, len(book.Places))
	for _, p := range book.Places {
		places = append(places, window.Place{Title: p.Title, Offset: p.Offset})
	}

	var out []port.Window
	for _, large := range window.Cut(book.Text, places, sizes) {
		w := windowAt(book, large)
		for _, small := range large.Small {
			w.Small = append(w.Small, windowAt(book, small))
		}
		out = append(out, w)
	}
	return out
}

// windowAt is one window with what it holds and where the book says it is. The
// text goes with it to be indexed for its words and is not kept.
func windowAt(book *epub.Book, w window.Window) port.Window {
	return port.Window{
		Start:    w.Start,
		Length:   w.Length,
		Location: located(book.Locate(w.Start)),
		Text:     w.Slice(book.Text),
	}
}

// located is where a window is, in the words the book itself uses: the part its
// own navigation names, and the page of the printed book it was made from. Empty
// where the book named neither, which is half the books read.
func located(at epub.Location) string {
	named := make([]string, 0, 2)
	if at.Place != "" {
		named = append(named, at.Place)
	}
	if at.Page != "" {
		named = append(named, at.Page)
	}
	return strings.Join(named, ", ")
}

// counted is how many windows of both sizes a cut produced.
func counted(windows []port.Window) int {
	n := len(windows)
	for _, w := range windows {
		n += len(w.Small)
	}
	return n
}

// recipe names what produced a source's text: the reader that took it out, and
// the sizes it was cut into. Both are asked as one question — a source whose
// recipe is not this one owes its text again.
func recipe(s window.Sizes) string {
	return fmt.Sprintf("%s/large=%d+%d/small=%d+%d/limit=%d",
		extractor, s.Large, s.LargeOverlap, s.Small, s.SmallOverlap, s.Limit)
}

// sizes fills in what configuration left unset with the defaults the cut applies,
// so that the recipe names the sizes the text was cut into.
func (u Extract) sizes() window.Sizes {
	s := u.Sizes
	if s.Large == 0 {
		s.Large = window.DefaultLarge
	}
	if s.Small <= 0 {
		s.Small = window.DefaultSmall
	}
	if s.LargeOverlap == 0 {
		s.LargeOverlap = window.DefaultLargeOverlap
	}
	if s.SmallOverlap == 0 {
		s.SmallOverlap = window.DefaultSmallOverlap
	}
	if s.Limit <= 0 {
		s.Limit = window.DefaultLimit
	}
	return s
}

// fingerprint addresses the content of a file the index has read.
func fingerprint(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func (u Extract) progress(res ExtractResult) {
	if u.OnProgress != nil {
		u.OnProgress(res)
	}
}
