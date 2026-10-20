package source

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"slices"

	"github.com/jiva-studio/numen/modules/libs/core/internal/chunking"
	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/internal/urlfile"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// sourcesPerQuery bounds one answer about what owes its text, so that a library
// of any size is worked through in pieces.
const sourcesPerQuery = 100

// Extract turns the books a vault holds into chunks.
//
// It runs in two passes. The first records what the vault has, opening nothing.
// The second takes the text out of every source that owes it and cuts it into
// chunks, one write per source.
type Extract struct {
	readers port.VaultReaders
	sources port.SourceRepository
	queries port.SourceQueries

	// Derived is optional. It holds what a recogniser wrote; without one, a
	// source is read from its own bytes and a recognition is not looked for.
	Derived port.DerivedStore
	// Documents is optional. It reads a format that needs a library; without
	// one, a source in that format is unreadable.
	Documents port.TextExtractor
	// Area is the producer a recognition is kept under. Empty means the default.
	Area string

	// Kinds are the sorts of source this cuts. Empty means the books and the
	// recordings a vault holds.
	Kinds []domain.SourceKind

	// Sizes are how the text is cut. They are named in the recipe, so a source
	// cut at other sizes owes its text again.
	Sizes chunking.Sizes

	// Legibility is what a chunk has to read like to be kept. It is not in the
	// recipe: a chunk dropped as unreadable is not a different cut of the text.
	Legibility chunking.Legibility

	// ShouldRebuildIndex reads every file and puts it in the index again, whatever the
	// index remembers about it.
	//
	// What it remembers is a path, a size and a modification time, so a file whose
	// content changed while those did not is skipped — an archive restored by
	// `unzip`, a tree brought over by `rsync -tc`. Rebuilding is the way out of
	// that, and the only one.
	ShouldRebuildIndex bool

	// OnProgress, if set, is called as each source is opened and as each one is
	// written. A library takes minutes, and something has to be able to say how
	// far it has got.
	OnProgress func(ExtractResult)
}

// NewExtract is what a vault's sources are cut into chunks through: the vault
// they are read out of, where a source and its chunks are written, and what
// says which sources the index already holds.
func NewExtract(
	readers port.VaultReaders, sources port.SourceRepository, queries port.SourceQueries,
) Extract {
	return Extract{readers: readers, sources: sources, queries: queries}
}

// ExtractResult reports what extraction did.
//
// `Seen` counts the sources of the kinds this cuts and nothing else. `Recorded`
// and `Extracted` are separate numbers because they are separate passes: a book
// is recorded when the vault is walked and extracted when its text is read.
type ExtractResult struct {
	Seen          int    // sources found in the vault
	Recorded      int    // new or changed, so owing their text
	Unchanged     int    // skipped on size and modification time alone
	Extracted     int    // read, cut and written
	Chunks        int    // chunks written, of both sizes
	Unreadable    int    // read, and not a book
	Removed       int    // held by the index, no longer in the vault
	Forgotten     int    // read by a model once, and that recognition is gone
	Vanished      int    // named by the index, gone by the time it was read
	CurrentSource string // the source open now
}

// Execute brings the chunks of one vault's books up to date with what is on
// disk.
func (u Extract) Execute(ctx context.Context, v domain.Vault) (ExtractResult, error) {
	var res ExtractResult

	reader, err := u.readers.Open(v)
	if err != nil {
		return res, err
	}
	var swept []port.SourceText
	if err := u.discover(ctx, v, reader, &res, &swept); err != nil {
		return res, err
	}
	if err := u.dropMissingText(ctx, v, reader, &res); err != nil {
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
	known, err := u.readKnown(ctx, v)
	if err != nil {
		return err
	}
	found, err := u.walkAndRecord(ctx, v, reader, known, res)
	if err != nil {
		return err
	}
	return u.removeGone(ctx, v, known, found, res, swept)
}

// readKnown reads the fingerprints the index holds, by kind and path.
func (u Extract) readKnown(ctx context.Context, v domain.Vault) (map[domain.SourceKind]map[string]domain.Fingerprint, error) {
	known := make(map[domain.SourceKind]map[string]domain.Fingerprint, len(u.kinds()))
	for _, kind := range u.kinds() {
		held, err := u.queries.GetFingerprints(ctx, v.ID, kind)
		if err != nil {
			return nil, fmt.Errorf("read index: %w", err)
		}
		known[kind] = held
	}
	return known, nil
}

// walkAndRecord walks the vault, records each file the index does not already hold
// as it is, and returns the paths it found.
func (u Extract) walkAndRecord(
	ctx context.Context,
	v domain.Vault,
	reader port.VaultReader,
	known map[domain.SourceKind]map[string]domain.Fingerprint,
	res *ExtractResult,
) (map[string]bool, error) {
	found := make(map[string]bool)
	err := reader.WalkFiles(ctx, func(ref domain.Fingerprint) error {
		held, ours := known[ref.Kind]
		if !ours {
			return nil
		}
		res.Seen++
		found[ref.Path] = true
		if previous, ok := held[ref.Path]; ok && !u.ShouldRebuildIndex && previous.IsUnchanged(ref) {
			res.Unchanged++
			return nil
		}
		if err := u.sources.SaveSource(ctx, v.ID, domain.Source{Fingerprint: ref}); err != nil {
			return fmt.Errorf("record %s: %w", ref.Path, err)
		}
		res.Recorded++
		return nil
	})
	return found, err
}

// removeGone takes out of the index what it holds and the walk did not find.
// The text of a passage is read from the file, so a source that is not there
// answers a search with nothing.
func (u Extract) removeGone(
	ctx context.Context,
	v domain.Vault,
	known map[domain.SourceKind]map[string]domain.Fingerprint,
	found map[string]bool,
	res *ExtractResult,
	swept *[]port.SourceText,
) error {
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
		// The recognitions those paths stood on are read before the rows saying so
		// are taken out; which of them nothing stands on any more is settled once
		// every source has been cut.
		went, err := u.readSourceTexts(ctx, v, kind, gone)
		if err != nil {
			return err
		}
		*swept = append(*swept, went...)
		if err := u.sources.RemoveSources(ctx, v.ID, kind, gone); err != nil {
			return fmt.Errorf("remove: %w", err)
		}
		res.Removed += len(gone)
	}
	return nil
}

// readSourceTexts is the recognition each of these paths stood on, for the ones
// that stood on any.
func (u Extract) readSourceTexts(
	ctx context.Context,
	v domain.Vault,
	kind domain.SourceKind,
	paths []string,
) ([]port.SourceText, error) {
	if u.Derived == nil {
		return nil, nil
	}
	held, err := u.queries.GetRecognisedSources(ctx, v.ID, kind)
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

// sweep takes out the files of a recognition no source stands on any more.
//
// A recognition is named by the hash of the bytes it was made from, so a document
// renamed is the same recognition at another path and two copies of one document
// are one recognition. What is asked is whether any source still names it, and the
// question is asked once every source has been cut and says what it stands on.
//
// A walk that failed returns before any of this, so a folder that could not be
// read takes nothing with it.
func (u Extract) sweep(ctx context.Context, v domain.Vault, went []port.SourceText) error {
	if u.Derived == nil || len(went) == 0 {
		return nil
	}
	stood, named, err := u.readReferencedRecognitions(ctx, v)
	if err != nil {
		return err
	}
	for _, r := range went {
		if stood[port.SourceText{Producer: r.Producer, Hash: r.Hash}] {
			continue
		}
		gone := domain.GetArtifactNames(r.Producer, r.Hash)
		// A copy of a video is named by the address alone, so it goes when
		// nothing names that address any more, whoever downloaded the words.
		if !named[r.Hash] {
			gone = append(gone, domain.NewCopyName(r.Hash))
		}
		if err := u.removeNames(ctx, gone); err != nil {
			return err
		}
	}
	return nil
}

// readReferencedRecognitions returns the recognitions some source stands on and the hashes some
// source names.
func (u Extract) readReferencedRecognitions(ctx context.Context, v domain.Vault) (map[port.SourceText]bool, map[string]bool, error) {
	stood := make(map[port.SourceText]bool)
	named := make(map[string]bool)
	for _, kind := range u.kinds() {
		held, err := u.queries.GetRecognisedSources(ctx, v.ID, kind)
		if err != nil {
			return nil, nil, fmt.Errorf("read index: %w", err)
		}
		for _, r := range held {
			stood[port.SourceText{Producer: r.Producer, Hash: r.Hash}] = true
			named[r.Hash] = true
		}
	}
	return stood, named, nil
}

// removeNames removes each named file from the store; a file already absent
// counts as removed.
func (u Extract) removeNames(ctx context.Context, names []domain.ArtifactName) error {
	for _, name := range names {
		if err := u.Derived.RemoveArtifact(ctx, name); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("remove %s: %w", name, err)
		}
	}
	return nil
}

// dropMissingText finds the sources whose producer's files are no longer there.
//
// The store is a folder on the person's disk and they may empty it. A source
// whose text went with it answers a search with nothing and would go on doing
// so, because its recipe is still the one in use: nothing else asks after it.
// Recording it afresh with no recipe clears which producer made its text, so the
// next pass cuts it from the document again.
func (u Extract) dropMissingText(ctx context.Context, v domain.Vault, reader port.VaultReader, res *ExtractResult) error {
	if u.Derived == nil {
		return nil
	}
	for _, kind := range u.kinds() {
		recognised, err := u.queries.GetRecognisedSources(ctx, v.ID, kind)
		if err != nil {
			return fmt.Errorf("read index: %w", err)
		}
		for _, r := range recognised {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := u.recordIfTextMissing(ctx, v, reader, r, res); err != nil {
				return err
			}
		}
	}
	return nil
}

// recordIfTextMissing records the source afresh when the store holds no text for its
// recognition.
func (u Extract) recordIfTextMissing(
	ctx context.Context,
	v domain.Vault,
	reader port.VaultReader,
	r port.SourceText,
	res *ExtractResult,
) error {
	if u.hasAny(ctx,
		domain.NewArtifactName(domain.ArtifactText, r.Producer, r.Hash),
		domain.NewArtifactName(domain.ArtifactPartial, r.Producer, r.Hash),
	) {
		return nil
	}
	ref, err := reader.GetFingerprint(ctx, r.Path)
	if port.IsNoNote(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("stat %s: %w", r.Path, err)
	}
	if err := u.sources.SaveSource(ctx, v.ID, domain.Source{Fingerprint: ref}); err != nil {
		return fmt.Errorf("record %s: %w", r.Path, err)
	}
	res.Forgotten++
	return nil
}

// hasAny says the store has something under one of these names. A recognition
// still being written is the text of the pages read so far.
func (u Extract) hasAny(ctx context.Context, names ...domain.ArtifactName) bool {
	for _, name := range names {
		if _, err := u.Derived.ReadArtifact(ctx, name); !errors.Is(err, fs.ErrNotExist) {
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
				return u.queries.GetUnchunkedSources(ctx, v.ID, kind, sourcesPerQuery)
			},
			func(ctx context.Context) ([]string, error) {
				return u.queries.GetSourcesWithOtherRecipe(ctx, v.ID, kind, known, sourcesPerQuery)
			},
		)
	}
	for _, ask := range questions {
		for {
			paths, err := ask(ctx)
			if err != nil {
				return err
			}
			fresh, err := u.chunkNewPaths(ctx, v, reader, paths, tried, sizes, res)
			if err != nil {
				return err
			}
			if fresh == 0 {
				break
			}
		}
	}
	return nil
}

// chunkNewPaths cuts each path not yet in tried, marks it tried, and returns how
// many it cut.
func (u Extract) chunkNewPaths(
	ctx context.Context,
	v domain.Vault,
	reader port.VaultReader,
	paths []string,
	tried map[string]bool,
	sizes chunking.Sizes,
	res *ExtractResult,
) (int, error) {
	fresh := 0
	for _, path := range paths {
		if tried[path] {
			continue
		}
		tried[path] = true
		fresh++
		if err := ctx.Err(); err != nil {
			return fresh, err
		}
		if err := u.source(ctx, v, reader, path, sizes, res); err != nil {
			return fresh, err
		}
	}
	return fresh, nil
}

// ExtractOne brings a single source's chunks up to date with its text.
//
// It is what a recognition calls as it writes: the pages already read are cut
// and can be embedded while the rest of the document is still being read.
func (u Extract) ExtractOne(ctx context.Context, v domain.Vault, path string) (ExtractResult, error) {
	var res ExtractResult
	reader, err := u.readers.Open(v)
	if err != nil {
		return res, err
	}
	return res, u.source(ctx, v, reader, path, u.sizes(), &res)
}

// source takes the text out of one book and writes the chunks it was cut into.
//
// A file that will not parse is counted and the run goes on: no structure is an
// ordinary outcome, and one bad book must not stop a library. A vault out of
// reach, or an index that will not take the chunks, stops it.
//
// The file is stated before it is read, so a file that changes while it is being
// read keeps the older fingerprint and is asked for again.
func (u Extract) source(
	ctx context.Context,
	v domain.Vault,
	reader port.VaultReader,
	path string,
	sizes chunking.Sizes,
	res *ExtractResult,
) error {
	res.CurrentSource = path
	u.progress(*res)

	ref, err := reader.GetFingerprint(ctx, path)
	if port.IsNoNote(err) {
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
	hash, err := u.fingerprint(ref, raw)
	if errors.Is(err, text.ErrUnreadable) {
		res.Unreadable++
		return nil
	}
	if err != nil {
		return err
	}

	// A document read by a recogniser has a text of its own, and the chunks are
	// places in that. It is found by the hash of the bytes it was read from, so a
	// file whose modification time moved without its content is claimed again
	// rather than read from scratch.
	doc, from, err := u.text(ctx, ref, raw, hash)
	if errors.Is(err, text.ErrUnreadable) {
		res.Unreadable++
		return nil
	}
	if err != nil {
		return err
	}

	chunks := chunksOf(doc, sizes, u.Legibility)
	extraction := domain.SourceChunks{
		Source: domain.Source{
			Fingerprint: ref,
			Hash:        hash,
			Recipe:      recipe(name, sizes),
			Producer:    from,
		},
		Chunks: chunks,
	}
	if err := u.sources.SaveExtraction(ctx, v.ID, extraction); err != nil {
		return fmt.Errorf("write the chunks of %s: %w", path, err)
	}
	res.Extracted++
	res.Chunks += countChunks(chunks)
	u.progress(*res)
	return nil
}

// chunksOf cuts one source's text: the large chunks a result shows, each
// holding the small chunks that carry a vector.
func chunksOf(doc *text.Document, sizes chunking.Sizes, reads chunking.Legibility) []domain.Chunk {
	return chunking.Chunks(doc.Text, doc.Parts, sizes, reads, func(c chunking.Chunk, isLarge bool) domain.Chunk {
		out := chunkAt(doc, c)
		// The name of a section is kept on the chunk that begins it, and on that
		// one only.
		if isLarge {
			out.Opens = doc.GetNamesAt(c.Start)
		}
		return out
	})
}

// chunkAt is one chunk with what it holds and where the source says it is. The
// text goes with it to be indexed for its words and is not kept.
func chunkAt(doc *text.Document, c chunking.Chunk) domain.Chunk {
	return domain.Chunk{
		Start:    c.Start,
		Length:   c.Length,
		Location: doc.Locate(c.Start),
		Text:     c.ExtractText(doc.Text),
	}
}

// countChunks is how many chunks of both sizes a cut produced.
func countChunks(chunks []domain.Chunk) int {
	n := len(chunks)
	for _, c := range chunks {
		n += len(c.Small)
	}
	return n
}

// recipe names what produced a source's text: the reader that took it out, and
// the sizes it was cut into. Both are asked as one question — a source whose
// recipe is not this one owes its text again.
func recipe(reader string, s chunking.Sizes) string {
	return fmt.Sprintf("%s/large=%d+%d/small=%d+%d/limit=%d",
		reader, s.Large, s.LargeOverlap, s.Small, s.SmallOverlap, s.Limit)
}

// recipes are what every reader would produce at these sizes. A source carrying
// none of them owes its text: its own reader has changed, or the sizes have.
func recipes(s chunking.Sizes) []string {
	named := text.Readers()
	out := make([]string, 0, len(named))
	for _, reader := range named {
		out = append(out, recipe(reader, s))
	}
	return out
}

// sizes are the sizes the cut works to, which is what the recipe has to name:
// a recipe describing anything else describes bytes that were never written.
func (u Extract) sizes() chunking.Sizes { return u.Sizes.Resolve() }

func (u Extract) progress(res ExtractResult) {
	if u.OnProgress != nil {
		u.OnProgress(res)
	}
}

// text is what a source says, and which producer made it.
//
// A recognition of these bytes stands in for the file's own text layer: it is
// what a person asked for, and a document whose layer is unusable is why they
// asked. A recognition still being written is the text of the pages read so far.
// Where there is none, the file speaks for itself and no producer is named.
func (u Extract) text(ctx context.Context, ref domain.Fingerprint, raw []byte, hash string) (*text.Document, string, error) {
	// A url is its address and nothing else, so whatever was fetched for that
	// address is its text. Which producer fetched it is what the store says.
	if ref.Kind == domain.KindURL {
		if u.Derived == nil {
			return &text.Document{}, "", nil
		}
		words, from, err := text.ReadDownloaded(ctx, u.Derived, hash)
		if err != nil {
			return nil, "", err
		}
		return &text.Document{Text: words}, from, nil
	}
	if u.Derived != nil {
		from := u.producer(ref)
		for _, name := range []domain.ArtifactName{
			domain.NewArtifactName(domain.ArtifactText, from, hash),
			domain.NewArtifactName(domain.ArtifactPartial, from, hash),
		} {
			switch found, err := u.Derived.ReadArtifact(ctx, name); {
			case err == nil:
				// The parts of a recognition bound the chunks it is cut into, the
				// way an outline bounds a book's.
				doc, err := text.ReadComposed(ctx, u.Derived, from, hash, found)
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

// producer is where this file's text comes from: a recording is listened to,
// and everything else is read.
func (u Extract) producer(ref domain.Fingerprint) string {
	if ref.Kind == domain.KindRecording {
		return domain.ASR
	}
	return u.area()
}

// kinds are the sorts of source this run works through.
func (u Extract) kinds() []domain.SourceKind {
	if len(u.Kinds) == 0 {
		return []domain.SourceKind{domain.KindBook, domain.KindRecording, domain.KindURL}
	}
	return u.Kinds
}

// fingerprint hashes the source's content, reading a URL file's target address
// where the source is a link.
func (u Extract) fingerprint(ref domain.Fingerprint, raw []byte) (string, error) {
	if ref.Kind != domain.KindURL {
		return text.Fingerprint(raw), nil
	}
	at, err := urlfile.Read(raw)
	if errors.Is(err, urlfile.ErrNoAddress) || errors.Is(err, domain.ErrNotAURL) {
		return "", text.ErrUnreadable
	}
	if err != nil {
		return "", err
	}
	return text.Fingerprint([]byte(string(at))), nil
}
