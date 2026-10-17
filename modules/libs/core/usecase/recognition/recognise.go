package recognition

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/highlight"
	"github.com/jiva-studio/numen/modules/libs/core/internal/ocr"
	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
)

// Recognise reads a scanned document with a model and writes down what it saw.
//
// It is started by a person and never by the application. Whether a document's
// own text layer is any good cannot be told from the text: a page of one
// language inside a book of another is ordinary, and a page of one language read
// as another is a defect, and nothing in the words says which. So the layer is
// used until somebody says otherwise, and this is how they say it.
//
// What comes out is an artifact: a model made it and no machine here can make it
// again, so it is written into the vault and the source is cut from it
// afterwards.
type Recognise struct {
	readers port.VaultReaders
	sources port.SourceRepository
	derived port.DerivedStores
	by      port.Recogniser

	// documents draws the pages a model is given.
	documents port.PageRenderer

	// Area is the store the artifact is kept in. Empty means the default.
	Area string

	// Batch is how many pages are read before what has been read is written
	// down. A document is an hour's work, and a run that is stopped keeps what
	// it had. Zero takes the default.
	Batch int

	// Cut is optional. It makes a source's chunks, and is called as pages are
	// written down, so what has been read is searchable before the rest of it
	// is. Where nothing cuts, the source is recorded as owing its text and the
	// next scan cuts it.
	Cut func(ctx context.Context, v domain.Vault, path string) error

	OnProgress func(RecogniseResult)
}

// NewRecognise is what a scan is read through: the vault it is read out of,
// where what the vault holds is recorded, the store the recognition is written
// into, what draws a page for the model, and the model itself.
func NewRecognise(
	readers port.VaultReaders,
	sources port.SourceRepository,
	derived port.DerivedStores,
	documents port.PageRenderer,
	by port.Recogniser,
) Recognise {
	return Recognise{
		readers: readers, sources: sources, derived: derived, documents: documents, by: by,
	}
}

// By returns the recogniser.
func (u Recognise) By() port.Recogniser { return u.by }

// RecogniseResult reports what recognition did.
type RecogniseResult struct {
	Path    string // the document being read
	Pages   int    // how many it has
	Read    int    // how many have been read, this run and before it
	Resumed int    // how many a run before this one had already read
	IsEmpty bool   // it says nothing, and nothing was written
	IsBusy  bool   // somebody else is recognising these bytes, and nothing was done
}

// DefaultBatch is how many pages are read before they are written down.
const DefaultBatch = 16

// Execute reads one document.
func (u Recognise) Execute(ctx context.Context, v domain.Vault, path string) (RecogniseResult, error) {
	res := RecogniseResult{Path: path}
	reader, err := u.readers.Open(v)
	if err != nil {
		return res, err
	}
	ref, err := reader.GetFingerprint(ctx, path)
	if err != nil {
		return res, fmt.Errorf("stat %s: %w", path, err)
	}
	store, err := u.derived.Open(v)
	if err != nil {
		return res, err
	}
	// The scan owns the file once OpenDocument is called, and every return before that
	// closes it.
	file, hash, err := openDocument(ctx, reader, path)
	if err != nil {
		return res, err
	}
	area := run.GetArea(u.Area, "ocr")
	final := domain.NewArtifactName(domain.ArtifactText, area, hash)
	partial := domain.NewArtifactName(domain.ArtifactPartial, area, hash)
	boxes := domain.NewArtifactName(domain.ArtifactBoxes, area, hash)
	parts := domain.NewArtifactName(domain.ArtifactParts, area, hash)

	// One run to a document. The name is the hash of its bytes, and it is held for
	// as long as the recognition takes.
	release, err := store.ClaimArtifact(ctx, partial)
	if err != nil {
		file.Close()
		if errors.Is(err, port.ErrClaimed) {
			res.IsBusy = true
			return res, nil
		}
		return res, err
	}
	defer release()

	// A document already read is not read again. The name is the hash of what
	// was read, so this holds however the file was renamed or moved since.
	if _, err := store.ReadArtifact(ctx, final); err == nil {
		file.Close()
		res.Read, res.Pages = 1, 1
		return res, run.Stand(ctx, u.Cut, u.sources, v, ref, hash, area)
	}

	scan, err := u.documents.OpenDocument(ctx, file)
	if err != nil {
		return res, fmt.Errorf("%s: %w", path, err)
	}
	defer scan.Close()
	res.Pages = scan.GetPageCount()

	out := pageWriter{u: u, v: v, store: store, path: path, partial: partial, boxes: boxes, parts: parts}
	if err := u.readPages(ctx, scan, &out, &res); err != nil {
		return res, err
	}
	err = u.saveRecognition(ctx, v, store, ref, hash, area, out, &res)
	return res, err
}

// openDocument opens a document and returns it rewound, with the fingerprint of
// its bytes. The file is closed when it returns an error.
func openDocument(ctx context.Context, reader port.VaultReader, path string) (io.ReadSeekCloser, string, error) {
	file, err := reader.OpenShared(ctx, path)
	if err != nil {
		return nil, "", fmt.Errorf("read %s: %w", path, err)
	}
	hash, err := text.GetFingerprint(file)
	if err == nil {
		_, err = file.Seek(0, io.SeekStart)
	}
	if err != nil {
		file.Close()
		return nil, "", fmt.Errorf("read %s: %w", path, err)
	}
	return file, hash, nil
}

// pageWriter writes runs of pages into the partial and its coordinate and part
// files, and cuts the source from everything the document has said so far.
//
// The coordinates and the parts go first, so a run that dies among the writes
// leaves them ahead of the count and the next run trims them back to it. Cutting
// writes which text the source stands on together with its chunks, in one
// statement, and the two are one fact.
type pageWriter struct {
	u       Recognise
	v       domain.Vault
	store   port.DerivedStore
	path    string
	partial domain.ArtifactName
	boxes   domain.ArtifactName
	parts   domain.ArtifactName
	prose   int
}

// write appends pages and, after them, the count of pages read.
func (w *pageWriter) write(ctx context.Context, pages []ocr.Page, read int) error {
	written, found, named := ocr.Write(pages)
	for i := range found {
		found[i].From += w.prose
		found[i].To += w.prose
	}
	for i := range named {
		named[i].Start += w.prose
	}
	if err := w.store.AppendArtifact(ctx, w.boxes, highlight.Pack(found)); err != nil {
		return err
	}
	if len(named) > 0 {
		if err := w.store.AppendArtifact(ctx, w.parts, ocr.Pack(named)); err != nil {
			return err
		}
	}
	if err := w.store.AppendArtifact(ctx, w.partial, appendCount(written, read)); err != nil {
		return err
	}
	said, _ := ocr.Read(written)
	w.prose += len(said)
	return run.Cut(ctx, w.u.Cut, w.v, w.path)
}

// readPages reads the pages the partial does not yet claim, writing a batch at a
// time.
func (u Recognise) readPages(
	ctx context.Context,
	scan port.OpenDocument,
	out *pageWriter,
	res *RecogniseResult,
) error {
	done, prose, err := readResumePoint(ctx, out.store, out.partial)
	if err != nil {
		return err
	}
	out.prose = prose
	// The files are written one after the other, so a run that died between them
	// left coordinates and parts the count does not claim.
	if err := dropBoxes(ctx, out.store, out.boxes, done); err != nil {
		return err
	}
	if err := dropParts(ctx, out.store, out.parts, prose); err != nil {
		return err
	}
	res.Resumed, res.Read = done, done
	run.Report(u.OnProgress, *res)

	batch := run.GetSize(u.Batch, DefaultBatch)
	pages := make([]ocr.Page, 0, batch)
	for index := done; index < scan.GetPageCount(); index++ {
		if err := ctx.Err(); err != nil {
			// What has been read is on disk already, and the next run begins
			// where this one stopped.
			return err
		}
		page, err := u.recognisePage(ctx, scan, out.path, index)
		if err != nil {
			return err
		}
		pages = append(pages, page)

		res.Read = index + 1
		if len(pages) >= batch {
			if err := out.write(ctx, pages, res.Read); err != nil {
				return err
			}
			pages = pages[:0]
		}
		run.Report(u.OnProgress, *res)
	}
	if len(pages) == 0 {
		return nil
	}
	return out.write(ctx, pages, res.Read)
}

// recognisePage draws one page and reads its blocks.
func (u Recognise) recognisePage(ctx context.Context, scan port.OpenDocument, path string, index int) (ocr.Page, error) {
	drawn, err := scan.RenderPage(index, u.by.GetRecognitionModel().DPI)
	if err != nil {
		return ocr.Page{}, fmt.Errorf("draw page %d of %s: %w", index+1, path, err)
	}
	blocks, err := u.by.Recognise(ctx, drawn)
	if err != nil {
		return ocr.Page{}, fmt.Errorf("read page %d of %s: %w", index+1, path, err)
	}
	return ocr.Page{Index: index, Size: drawn.Bounds().Size(), Blocks: blocks}, nil
}

// saveRecognition stands the source on the artifact made from the partial, or forgets a
// document that said nothing.
func (u Recognise) saveRecognition(
	ctx context.Context,
	v domain.Vault,
	store port.DerivedStore,
	ref domain.Fingerprint,
	hash, area string,
	out pageWriter,
	res *RecogniseResult,
) error {
	partial := out.partial
	final := domain.NewArtifactName(domain.ArtifactText, area, hash)
	whole, err := store.ReadArtifact(ctx, partial)
	if errors.Is(err, fs.ErrNotExist) {
		res.IsEmpty = true
		return nil
	}
	if err != nil {
		return err
	}
	if said, _ := ocr.Read(whole); strings.TrimSpace(said) == "" {
		// A document that says nothing writes nothing: an empty artifact would
		// stand in for a text layer that worked.
		res.IsEmpty = true
		return u.forget(ctx, v, ref, hash, store, partial, out.boxes, out.parts)
	}

	if err := store.WriteArtifact(ctx, final, whole); err != nil {
		return err
	}
	if err := u.record(ctx, store, area, hash); err != nil {
		return err
	}
	// The source stands on the artifact before the partial goes.
	if err := run.Stand(ctx, u.Cut, u.sources, v, ref, hash, area); err != nil {
		return err
	}
	return store.RemoveArtifact(ctx, partial)
}

// forget puts a document that said nothing back on its own text layer, and takes
// away what recognising it produced.
func (u Recognise) forget(
	ctx context.Context,
	v domain.Vault,
	ref domain.Fingerprint,
	hash string,
	store port.DerivedStore,
	names ...domain.ArtifactName,
) error {
	for _, name := range names {
		if err := store.RemoveArtifact(ctx, name); err != nil {
			return err
		}
	}
	return run.Stand(ctx, u.Cut, u.sources, v, ref, hash, "")
}

// dropBoxes drops the coordinates of pages no count claims.
func dropBoxes(ctx context.Context, store port.DerivedStore, name domain.ArtifactName, done int) error {
	raw, err := store.ReadArtifact(ctx, name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	held := highlight.Unpack(raw)
	kept := make([]domain.HighlightBox, 0, len(held))
	for _, box := range held {
		if box.Page < done {
			kept = append(kept, box)
		}
	}
	// Written back whatever was dropped. An append that did not land whole
	// leaves bytes that are not a record, and every record appended after them
	// is read at a shifted offset.
	return store.WriteArtifact(ctx, name, highlight.Pack(kept))
}

// dropParts drops the parts no count claims: those opening past the prose the
// pages before the count came to.
func dropParts(ctx context.Context, store port.DerivedStore, name domain.ArtifactName, prose int) error {
	raw, err := store.ReadArtifact(ctx, name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	held := ocr.Unpack(raw)
	kept := make([]domain.OCRPart, 0, len(held))
	for _, part := range held {
		if part.Start+part.Length <= prose {
			kept = append(kept, part)
		}
	}
	return store.WriteArtifact(ctx, name, ocr.Pack(kept))
}

// record keeps what read the document beside what it read. Nothing on any path
// that answers a question reads it: it is there so that a person can ask what
// produced a text, and so that everything a recogniser now known to be bad
// produced can be found again.
func (u Recognise) record(ctx context.Context, store port.DerivedStore, area, hash string) error {
	named := u.by.GetRecognitionModel()
	raw, err := json.MarshalIndent(struct {
		Layout     string `json:"layout"`
		Recogniser string `json:"recogniser"`
		DPI        int    `json:"dpi"`
		From       string `json:"from"`
		Recipe     string `json:"recipe"`
	}{named.Layout, named.Recogniser, named.DPI, named.From, named.Recipe()}, "", "  ")
	if err != nil {
		return err
	}
	return store.WriteArtifact(ctx, domain.NewArtifactName(domain.ArtifactProducer, area, hash), append(raw, '\n'))
}

// appendCount is a run of pages and, after them, how many of the document have
// been read, so that a run stopped part way can be taken up again.
//
// The count stands last and is what makes the batch before it count. A batch
// that did not land whole is one no count claims, and the next run reads those
// pages again.
//
// The count is a comment: it is a line the artifact's own reader passes over,
// because a page mark is what it looks for.
func appendCount(raw []byte, read int) []byte {
	return append(raw, fmt.Sprintf("%s%d\n", resumeMark, read)...)
}

// resumeMark begins the line that says how much of a document has been read.
const resumeMark = ocr.Note + "pages "

// readResumePoint is how many pages of a document a run before this one read
// and how many bytes of prose those pages came to, with anything past the last
// count cut away.
//
// A count stands after the pages it claims, so what follows the last one is a
// batch that did not land whole. The two numbers come from the same stretch of
// the file, which is what makes a coordinate written next land where its words
// are.
func readResumePoint(ctx context.Context, store port.DerivedStore, partial domain.ArtifactName) (int, int, error) {
	raw, err := store.ReadArtifact(ctx, partial)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, err
	}
	read, end := lastCount(raw)
	if end < len(raw) {
		if err := store.WriteArtifact(ctx, partial, raw[:end]); err != nil {
			return 0, 0, err
		}
	}
	said, _ := ocr.Read(raw[:end])
	return read, len(said), nil
}

// lastCount is the last count a partial carries and where the line holding it
// ends. A file carrying none is a document nothing has read.
func lastCount(raw []byte) (read, end int) {
	at := strings.LastIndex(string(raw), resumeMark)
	if at < 0 {
		return 0, 0
	}
	line := string(raw[at+len(resumeMark):])
	stop := strings.IndexByte(line, '\n')
	if stop < 0 {
		return lastCount(raw[:at])
	}
	read, err := strconv.Atoi(strings.TrimSpace(line[:stop]))
	if err != nil {
		return lastCount(raw[:at])
	}
	return read, at + len(resumeMark) + stop + 1
}
