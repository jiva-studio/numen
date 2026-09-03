package source

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/lit"
	"github.com/jiva-studio/numen/modules/libs/core/ocr"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/text"
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
	Readers port.VaultReaders
	Sources port.SourceRepository
	Derived port.DerivedStores
	By      port.Recogniser

	// Documents draws the pages a model is given.
	Documents port.Documents

	// Area is the store the artifact is kept in. Empty means the default.
	Area string

	// Batch is how many pages are read before what has been read is written
	// down. A document is an hour's work, and a run that is stopped keeps what
	// it had. Zero takes the default.
	Batch int

	// Cut makes a source's chunks. It is called as pages are written down, so
	// what has been read is searchable before the rest of it is.
	Cut func(ctx context.Context, v domain.Vault, path string) error

	OnProgress func(Recognised)
}

// Recognised reports what recognition did.
type Recognised struct {
	Path    string // the document being read
	Pages   int    // how many it has
	Read    int    // how many have been read, this run and before it
	Resumed int    // how many a run before this one had already read
	Empty   bool   // it says nothing, and nothing was written
	Busy    bool   // somebody else is reading these bytes, and nothing was done
}

// DefaultBatch is how many pages are read before they are written down.
const DefaultBatch = 16

// Execute reads one document.
func (u Recognise) Execute(ctx context.Context, v domain.Vault, path string) (Recognised, error) {
	res := Recognised{Path: path}
	if u.By == nil {
		return res, errors.New("no recogniser: none is configured")
	}

	reader, err := u.Readers.Open(v)
	if err != nil {
		return res, err
	}
	ref, err := reader.Stat(ctx, path)
	if err != nil {
		return res, fmt.Errorf("stat %s: %w", path, err)
	}
	raw, err := reader.Read(ctx, path)
	if err != nil {
		return res, fmt.Errorf("read %s: %w", path, err)
	}
	store, err := u.Derived.Open(v)
	if err != nil {
		return res, err
	}

	hash := text.Fingerprint(raw)
	area := u.area()
	final, partial := text.Artifact(area, hash), text.Partial(area, hash)
	boxes, parts := text.Boxes(area, hash), text.Parts(area, hash)

	// One run to a document. The name is the hash of its bytes, and it is held for
	// as long as the reading takes.
	release, err := store.Claim(ctx, partial)
	if errors.Is(err, port.ErrClaimed) {
		res.Busy = true
		return res, nil
	}
	if err != nil {
		return res, err
	}
	defer release()

	// A document already read is not read again. The name is the hash of what
	// was read, so this holds however the file was renamed or moved since.
	if _, err := store.Read(ctx, final); err == nil {
		res.Read, res.Pages = 1, 1
		return res, u.stand(ctx, v, ref, hash, area)
	}

	if u.Documents == nil {
		return res, fmt.Errorf("%s: nothing to draw a page with", path)
	}
	scan, err := u.Documents.Draw(ctx, raw)
	if err != nil {
		return res, fmt.Errorf("%s: %w", path, err)
	}
	defer scan.Close()
	res.Pages = scan.Pages()

	done, prose, err := resumed(ctx, store, partial)
	if err != nil {
		return res, err
	}
	// The files are written one after the other, so a run that died between them
	// left coordinates and parts the count does not claim.
	if err := trimmed(ctx, store, boxes, done); err != nil {
		return res, err
	}
	if err := shortened(ctx, store, parts, prose); err != nil {
		return res, err
	}
	res.Resumed, res.Read = done, done
	u.progress(res)

	// write puts a run of pages down and cuts the source from everything the
	// document has said so far.
	//
	// The coordinates and the parts go first, so a run that dies among the
	// writes leaves them ahead of the count and the next run trims them back to
	// it.
	//
	// Nothing here says which text the source stands on. Cutting writes that
	// and the chunks cut from it in one statement, and they are one fact: a row
	// naming a text its chunks are not offsets into answers with the wrong
	// words.
	write := func(pages []ocr.Page) error {
		written, found, named := ocr.Write(pages)
		for i := range found {
			found[i].Start += prose
		}
		for i := range named {
			named[i].Start += prose
		}
		if err := store.Append(ctx, boxes, lit.Pack(found)); err != nil {
			return err
		}
		if len(named) > 0 {
			if err := store.Append(ctx, parts, ocr.Pack(named)); err != nil {
				return err
			}
		}
		if err := store.Append(ctx, partial, marked(written, res.Read)); err != nil {
			return err
		}
		said, _ := ocr.Read(written)
		prose += len(said)
		return u.cut(ctx, v, path)
	}

	pages := make([]ocr.Page, 0, u.batch())
	for index := done; index < scan.Pages(); index++ {
		if err := ctx.Err(); err != nil {
			// What has been read is on disk already. Stopping is a thing a
			// person did, and the next run begins where this one stopped.
			return res, err
		}
		drawn, err := scan.Image(index, u.By.Recognition().DPI)
		if err != nil {
			return res, fmt.Errorf("draw page %d of %s: %w", index+1, path, err)
		}
		blocks, err := u.By.Recognise(ctx, drawn)
		if err != nil {
			return res, fmt.Errorf("read page %d of %s: %w", index+1, path, err)
		}
		pages = append(pages, ocr.Page{
			At:     index,
			Size:   drawn.Bounds().Size(),
			Blocks: blocks,
		})

		res.Read = index + 1
		if len(pages) >= u.batch() {
			if err := write(pages); err != nil {
				return res, err
			}
			pages = pages[:0]
		}
		u.progress(res)
	}
	if len(pages) > 0 {
		if err := write(pages); err != nil {
			return res, err
		}
	}

	whole, err := store.Read(ctx, partial)
	if errors.Is(err, fs.ErrNotExist) {
		res.Empty = true
		return res, nil
	}
	if err != nil {
		return res, err
	}
	if said, _ := ocr.Read(whole); strings.TrimSpace(said) == "" {
		// A document that says nothing writes nothing. An empty artifact would
		// stand in for a text layer that worked, and there is no falling back
		// from one.
		res.Empty = true
		return res, u.forget(ctx, v, ref, hash, store, partial, boxes, parts)
	}

	if err := store.Write(ctx, final, whole); err != nil {
		return res, err
	}
	if err := u.record(ctx, store, area, hash); err != nil {
		return res, err
	}
	// The source stands on the artifact before the partial goes.
	if err := u.stand(ctx, v, ref, hash, area); err != nil {
		return res, err
	}
	return res, store.Remove(ctx, partial)
}

// stand puts the source on the text a producer made.
//
// Cutting writes which text a source stands on together with the chunks cut
// from it, in one statement, and the two are one fact. Where nothing cuts here
// the source is recorded as owing its text, and the scan that cuts it writes
// both.
func (u Recognise) stand(ctx context.Context, v domain.Vault, ref domain.FileRef, hash, from string) error {
	if u.Cut != nil {
		return u.Cut(ctx, v, ref.Path)
	}
	return u.claim(ctx, v, ref, hash, from)
}

// forget puts a document that said nothing back on its own text layer, and takes
// away what reading it produced.
func (u Recognise) forget(
	ctx context.Context,
	v domain.Vault,
	ref domain.FileRef,
	hash string,
	store port.DerivedStore,
	names ...string,
) error {
	for _, name := range names {
		if err := store.Remove(ctx, name); err != nil {
			return err
		}
	}
	return u.stand(ctx, v, ref, hash, "")
}

// cut makes this source's chunks from what has been read so far.
func (u Recognise) cut(ctx context.Context, v domain.Vault, path string) error {
	if u.Cut == nil {
		return nil
	}
	return u.Cut(ctx, v, path)
}

// trimmed drops the coordinates of pages no count claims.
func trimmed(ctx context.Context, store port.DerivedStore, name string, done int) error {
	raw, err := store.Read(ctx, name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	held := lit.Unpack(raw)
	kept := make([]lit.Box, 0, len(held))
	for _, box := range held {
		if box.Page < done {
			kept = append(kept, box)
		}
	}
	// Written back whatever was dropped. An append that did not land whole
	// leaves bytes that are not a record, and every record appended after them
	// is read at a shifted offset.
	return store.Write(ctx, name, lit.Pack(kept))
}

// shortened drops the parts no count claims: those opening past the prose the
// pages before the count came to.
func shortened(ctx context.Context, store port.DerivedStore, name string, prose int) error {
	raw, err := store.Read(ctx, name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	held := ocr.Unpack(raw)
	kept := make([]ocr.Part, 0, len(held))
	for _, part := range held {
		if part.Start+part.Length <= prose {
			kept = append(kept, part)
		}
	}
	return store.Write(ctx, name, ocr.Pack(kept))
}

// claim records which producer made this source's text.
//
// No recipe is written, so the source owes its text: what cuts it into chunks
// is extraction, which knows the sizes and is the one place that does.
func (u Recognise) claim(ctx context.Context, v domain.Vault, ref domain.FileRef, hash, from string) error {
	return u.Sources.SaveSource(ctx, v.ID, port.Source{Ref: ref, Hash: hash, TextFrom: from})
}

// record keeps what read the document beside what it read. Nothing on any path
// that answers a question reads it: it is there so that a person can ask what
// produced a text, and so that everything a recogniser now known to be bad
// produced can be found again.
func (u Recognise) record(ctx context.Context, store port.DerivedStore, area, hash string) error {
	named := u.By.Recognition()
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
	return store.Write(ctx, text.Beside(area, hash), append(raw, '\n'))
}

// marked is a run of pages and, after them, how many of the document have been
// read, so that a run stopped part way can be taken up again.
//
// The count stands last and is what makes the batch before it count. A batch
// that did not land whole is one no count claims, and the next run reads those
// pages again.
//
// The count is a comment: it is a line the artifact's own reader passes over,
// because a page mark is what it looks for.
func marked(raw []byte, read int) []byte {
	return append(raw, fmt.Sprintf("%s%d\n", resumeMark, read)...)
}

// resumeMark begins the line that says how much of a document has been read.
const resumeMark = ocr.Note + "pages "

// resumed is how many pages of a document a run before this one read and how
// many bytes of prose those pages came to, with anything past the last count cut
// away.
//
// A count stands after the pages it claims, so what follows the last one is a
// batch that did not land whole. The two numbers come from the same stretch of
// the file, which is what makes a coordinate written next land where its words
// are.
func resumed(ctx context.Context, store port.DerivedStore, partial string) (int, int, error) {
	raw, err := store.Read(ctx, partial)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, err
	}
	read, end := lastCount(raw)
	if end < len(raw) {
		if err := store.Write(ctx, partial, raw[:end]); err != nil {
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

func (u Recognise) area() string {
	if u.Area == "" {
		return "ocr"
	}
	return u.Area
}

func (u Recognise) batch() int {
	if u.Batch <= 0 {
		return DefaultBatch
	}
	return u.Batch
}

func (u Recognise) progress(res Recognised) {
	if u.OnProgress != nil {
		u.OnProgress(res)
	}
}
