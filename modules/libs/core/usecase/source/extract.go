package source

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"slices"

	"github.com/jiva-studio/numen/modules/libs/core/cutting"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/text"
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
// chunks, one write per source.
type Extract struct {
	Readers port.VaultReaders
	Sources port.SourceRepository
	Owing   port.SourceQueries

	// Derived holds what a recogniser wrote. Without one, a source is read from
	// its own bytes and a recognition is not looked for.
	Derived port.DerivedStore
	// Documents reads a format that needs a library. Without one, a source in
	// that format is unreadable.
	Documents port.Documents
	// Area is the producer a recognition is kept under. Empty means the default.
	Area string

	// Kinds are the sorts of source this cuts. Empty means the books and the
	// recordings a vault holds.
	Kinds []domain.SourceKind

	// Sizes are how the text is cut. They are named in the recipe, so a source
	// cut at other sizes owes its text again.
	Sizes cutting.Sizes

	// RebuildIndex reads every file and puts it in the index again, whatever the
	// index remembers about it.
	//
	// What it remembers is a path, a size and a modification time, so a file whose
	// content changed while those did not is skipped — an archive restored by
	// `unzip`, a tree brought over by `rsync -tc`. Rebuilding is the way out of
	// that, and the only one.
	RebuildIndex bool

	// OnProgress, if set, is called as each source is opened and as each one is
	// written. A library takes minutes, and something has to be able to say how
	// far it has got.
	OnProgress func(ExtractResult)
}

// ExtractResult reports what extraction did.
//
// `Seen` counts the sources of the kinds this cuts and nothing else. `Recorded`
// and `Extracted` are separate numbers because they are separate passes: a book
// is recorded when the vault is walked and extracted when its text is read.
type ExtractResult struct {
	Seen       int    // sources found in the vault
	Recorded   int    // new or changed, so owing their text
	Unchanged  int    // skipped on size and modification time alone
	Extracted  int    // read, cut and written
	Chunks     int    // chunks written, of both sizes
	Unreadable int    // read, and not a book
	Removed    int    // held by the index, no longer in the vault
	Forgotten  int    // read by a model once, and that reading is gone
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
	var swept []port.SourceText
	if err := u.discover(ctx, v, reader, &res, &swept); err != nil {
		return res, err
	}
	if err := u.forgotten(ctx, v, reader, &res); err != nil {
		return res, err
	}
	if err := u.cut(ctx, v, reader, &res); err != nil {
		return res, err
	}
	return res, u.sweep(ctx, v, swept)
}

// discover records every book the vault holds. Nothing is opened: a file whose
// size and modification time are what the index believes is left alone, and one
// that differs is recorded with no recipe, which is what owing its text means.
func (u Extract) discover(
	ctx context.Context,
	v domain.Vault,
	reader port.VaultReader,
	res *ExtractResult,
	swept *[]port.SourceText,
) error {
	known := make(map[domain.SourceKind]map[string]domain.Fingerprint, len(u.kinds()))
	for _, kind := range u.kinds() {
		held, err := u.Owing.Fingerprints(ctx, v.ID, kind)
		if err != nil {
			return fmt.Errorf("read index: %w", err)
		}
		known[kind] = held
	}

	found := make(map[string]bool)
	if err := reader.Walk(ctx, func(ref domain.Fingerprint) error {
		held, ours := known[ref.Kind]
		if !ours {
			return nil
		}
		res.Seen++
		found[ref.Path] = true
		if previous, ok := held[ref.Path]; ok && !u.RebuildIndex && previous.Unchanged(ref) {
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
	for _, kind := range u.kinds() {
		gone := make([]string, 0)
		for path := range known[kind] {
			if !found[path] {
				gone = append(gone, path)
			}
		}
		if len(gone) == 0 {
			continue
		}
		slices.Sort(gone)
		// What those paths stood on, before the rows saying so are taken out.
		// Which of those readings nothing stands on any more is a question for
		// once every source has been cut.
		went, err := u.standing(ctx, v, kind, gone)
		if err != nil {
			return err
		}
		*swept = append(*swept, went...)
		if err := u.Sources.RemoveSources(ctx, v.ID, kind, gone); err != nil {
			return fmt.Errorf("remove: %w", err)
		}
		res.Removed += len(gone)
	}
	return nil
}

// standing is the reading each of these paths stood on, for the ones that stood
// on any.
func (u Extract) standing(
	ctx context.Context,
	v domain.Vault,
	kind domain.SourceKind,
	paths []string,
) ([]port.SourceText, error) {
	if u.Derived == nil {
		return nil, nil
	}
	held, err := u.Owing.Recognised(ctx, v.ID, kind)
	if err != nil {
		return nil, fmt.Errorf("read index: %w", err)
	}
	made := make(map[string]port.SourceText, len(held))
	for _, r := range held {
		made[r.Path] = r
	}
	out := make([]port.SourceText, 0, len(paths))
	for _, path := range paths {
		if r, on := made[path]; on && r.Producer != "" {
			out = append(out, r)
		}
	}
	return out, nil
}

// sweep takes out the files of a reading no source stands on any more.
//
// A reading is named by the hash of the bytes it was made from, so a document
// renamed is the same reading at another path and two copies of one document
// are one reading. What is asked is whether any source still names it, and the
// question is asked once every source has been cut and says what it stands on.
//
// A walk that failed returns before any of this, so a folder that could not be
// read takes nothing with it.
func (u Extract) sweep(ctx context.Context, v domain.Vault, went []port.SourceText) error {
	if u.Derived == nil || len(went) == 0 {
		return nil
	}
	stood := make(map[port.SourceText]bool)
	for _, kind := range u.kinds() {
		held, err := u.Owing.Recognised(ctx, v.ID, kind)
		if err != nil {
			return fmt.Errorf("read index: %w", err)
		}
		for _, r := range held {
			stood[port.SourceText{Producer: r.Producer, Hash: r.Hash}] = true
		}
	}
	for _, r := range went {
		if stood[port.SourceText{Producer: r.Producer, Hash: r.Hash}] {
			continue
		}
		for _, name := range text.Names(r.Producer, r.Hash) {
			if err := u.Derived.Remove(ctx, name); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("remove %s: %w", name, err)
			}
		}
	}
	return nil
}

// forgotten finds the sources whose producer's files are no longer there.
//
// The store is a folder on the person's disk and they may empty it. A source
// whose text went with it answers a search with nothing and would go on doing
// so, because its recipe is still the one in use: nothing else asks after it.
// Recording it afresh with no recipe clears which producer made its text, so the
// next pass cuts it from the document again.
func (u Extract) forgotten(ctx context.Context, v domain.Vault, reader port.VaultReader, res *ExtractResult) error {
	if u.Derived == nil {
		return nil
	}
	for _, kind := range u.kinds() {
		standing, err := u.Owing.Recognised(ctx, v.ID, kind)
		if err != nil {
			return fmt.Errorf("read index: %w", err)
		}
		for _, r := range standing {
			if err := ctx.Err(); err != nil {
				return err
			}
			if u.holds(ctx, text.Artifact(r.Producer, r.Hash), text.Partial(r.Producer, r.Hash)) {
				continue
			}
			ref, err := reader.Stat(ctx, r.Path)
			if port.NoNote(err) {
				continue
			}
			if err != nil {
				return fmt.Errorf("stat %s: %w", r.Path, err)
			}
			if err := u.Sources.SaveSource(ctx, v.ID, port.Source{Ref: ref}); err != nil {
				return fmt.Errorf("record %s: %w", r.Path, err)
			}
			res.Forgotten++
		}
	}
	return nil
}

// holds says the store has something under one of these names. A recognition
// still running is the text of the pages it has read.
func (u Extract) holds(ctx context.Context, names ...string) bool {
	for _, name := range names {
		if _, err := u.Derived.Read(ctx, name); !errors.Is(err, fs.ErrNotExist) {
			return true
		}
	}
	return false
}

// cut extracts and cuts every source that owes its text.
//
// Which those are is asked of the data, on both keys that answer it: a source
// with no small chunk has never been cut, and a source carrying another recipe
// was cut by another extractor or at other sizes. Each question is asked again
// until it names nothing that has not been tried, so a run that stopped part way
// is continued by starting another.
func (u Extract) cut(ctx context.Context, v domain.Vault, reader port.VaultReader, res *ExtractResult) error {
	sizes := u.sizes()
	known := recipes(sizes)
	tried := map[string]bool{}

	var questions []func(context.Context) ([]string, error)
	for _, kind := range u.kinds() {
		questions = append(questions,
			func(ctx context.Context) ([]string, error) {
				return u.Owing.Unchunked(ctx, v.ID, kind, sourcesPerQuery)
			},
			func(ctx context.Context) ([]string, error) {
				return u.Owing.ByOtherRecipe(ctx, v.ID, kind, known, sourcesPerQuery)
			},
		)
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
				if err := u.source(ctx, v, reader, path, sizes, res); err != nil {
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

// One brings a single source's chunks up to date with its text.
//
// It is what a recognition calls as it writes: the pages already read are cut
// and can be embedded while the rest of the document is still being read.
func (u Extract) One(ctx context.Context, v domain.Vault, path string) (ExtractResult, error) {
	var res ExtractResult
	reader, err := u.Readers.Open(v)
	if err != nil {
		return res, err
	}
	return res, u.source(ctx, v, reader, path, u.sizes(), &res)
}

// source takes the text out of one book and writes the chunks it was cut into.
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
	sizes cutting.Sizes,
	res *ExtractResult,
) error {
	res.Reading = path
	u.progress(*res)

	ref, err := reader.Stat(ctx, path)
	if port.NoNote(err) {
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
		if ctx.Err() != nil {
			return err
		}
		// A file that will not open is one file. Stopping here records nothing,
		// and every later run stops at the same place, leaving every book after
		// it unread.
		res.Unreadable++
		return nil
	}

	// Which reader takes the text out is the file's to say, and it is part of
	// what the recipe records: two readers give one file two texts, and a chunk
	// keeps an offset into one of them.
	name, ok := text.ReaderName(ref)
	if !ok {
		res.Unreadable++
		return nil
	}
	hash := text.Fingerprint(raw)

	// A document read by a recogniser has a text of its own, and the chunks are
	// places in that. It is found by the hash of the bytes it was read from, so a
	// file whose modification time moved without its content is claimed again
	// rather than read from scratch.
	doc, from, err := u.text(ctx, ref, raw, hash)
	if err != nil {
		res.Unreadable++
		return nil
	}

	chunks := chunksOf(doc, sizes)
	extraction := port.Extraction{
		Source: port.Source{
			Ref:      ref,
			Hash:     hash,
			Recipe:   recipe(name, sizes),
			TextFrom: from,
		},
		Chunks: chunks,
	}
	if err := u.Sources.SaveExtraction(ctx, v.ID, extraction); err != nil {
		return fmt.Errorf("write the chunks of %s: %w", path, err)
	}
	res.Extracted++
	res.Chunks += counted(chunks)
	u.progress(*res)
	return nil
}

// chunksOf cuts one source's text: the large chunks a result shows, each
// holding the small chunks that carry a vector.
func chunksOf(doc *text.Document, sizes cutting.Sizes) []port.Chunk {
	var out []port.Chunk
	for _, large := range cutting.Cut(doc.Text, doc.Parts, sizes) {
		c := chunkAt(doc, large)
		// The name of a section is kept on the chunk that begins it, and on that
		// one only: a small chunk standing at the same offset is inside it, and
		// one section named twice is one section answering twice.
		c.Opens = doc.Opens(large.Start)
		for _, small := range large.Small {
			c.Small = append(c.Small, chunkAt(doc, small))
		}
		out = append(out, c)
	}
	return out
}

// chunkAt is one chunk with what it holds and where the source says it is. The
// text goes with it to be indexed for its words and is not kept.
func chunkAt(doc *text.Document, c cutting.Chunk) port.Chunk {
	return port.Chunk{
		Start:    c.Start,
		Length:   c.Length,
		Location: doc.Locate(c.Start),
		Text:     c.Slice(doc.Text),
	}
}

// counted is how many chunks of both sizes a cut produced.
func counted(chunks []port.Chunk) int {
	n := len(chunks)
	for _, c := range chunks {
		n += len(c.Small)
	}
	return n
}

// recipe names what produced a source's text: the reader that took it out, and
// the sizes it was cut into. Both are asked as one question — a source whose
// recipe is not this one owes its text again.
func recipe(reader string, s cutting.Sizes) string {
	return fmt.Sprintf("%s/large=%d+%d/small=%d+%d/limit=%d",
		reader, s.Large, s.LargeOverlap, s.Small, s.SmallOverlap, s.Limit)
}

// recipes are what every reader would produce at these sizes. A source carrying
// none of them owes its text: its own reader has changed, or the sizes have.
func recipes(s cutting.Sizes) []string {
	named := []string{text.ReaderEPUB, text.ReaderPDF, text.ReaderRecording}
	out := make([]string, 0, len(named))
	for _, reader := range named {
		out = append(out, recipe(reader, s))
	}
	return out
}

// sizes fills in what configuration left unset with the defaults the cut applies,
// so that the recipe names the sizes the text was cut into.
func (u Extract) sizes() cutting.Sizes {
	s := u.Sizes
	if s.Large == 0 {
		s.Large = cutting.DefaultLarge
	}
	if s.Small <= 0 {
		s.Small = cutting.DefaultSmall
	}
	if s.LargeOverlap == 0 {
		s.LargeOverlap = cutting.DefaultLargeOverlap
	}
	if s.SmallOverlap == 0 {
		s.SmallOverlap = cutting.DefaultSmallOverlap
	}
	if s.Limit <= 0 {
		s.Limit = cutting.DefaultLimit
	}
	return s
}

func (u Extract) progress(res ExtractResult) {
	if u.OnProgress != nil {
		u.OnProgress(res)
	}
}

// text is what a source says, and which producer made it.
//
// A recognition of these bytes stands in for the file's own text layer: it is
// what a person asked for, and a document whose layer is unusable is why they
// asked. A recognition still running is the text of the pages it has read.
// Where there is none, the file speaks for itself and no producer is named.
func (u Extract) text(ctx context.Context, ref domain.Fingerprint, raw []byte, hash string) (*text.Document, string, error) {
	if u.Derived != nil {
		from := u.producer(ref)
		for _, name := range []string{text.Artifact(from, hash), text.Partial(from, hash)} {
			switch found, err := u.Derived.Read(ctx, name); {
			case err == nil:
				// The parts of a reading bound the chunks it is cut into, the
				// way an outline bounds a book's.
				doc, err := text.Composed(ctx, u.Derived, from, hash, found)
				if err != nil {
					return nil, "", err
				}
				return doc, from, nil
			case !errors.Is(err, fs.ErrNotExist):
				return nil, "", err
			}
		}
	}
	doc, err := text.Read(ctx, u.Documents, ref, raw)
	return doc, "", err
}

// area is the producer a recognition is kept under, and the first part of every
// name its files carry.
func (u Extract) area() string {
	if u.Area == "" {
		return "ocr"
	}
	return u.Area
}

// producer is who would have written this file's text down: a recording is
// listened to, and everything else is read.
func (u Extract) producer(ref domain.Fingerprint) string {
	if ref.Kind == domain.KindRecording {
		return text.ASR
	}
	return u.area()
}

// kinds are the sorts of source this run works through.
func (u Extract) kinds() []domain.SourceKind {
	if len(u.Kinds) == 0 {
		return []domain.SourceKind{domain.KindBook, domain.KindRecording}
	}
	return u.Kinds
}
