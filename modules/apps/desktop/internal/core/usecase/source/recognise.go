package source

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/ocr"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/pdf"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/text"
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

	// Area is the store the artifact is kept in. Empty means the default.
	Area string

	// Batch is how many pages are read before what has been read is written
	// down. A document is an hour's work, and a run that is stopped keeps what
	// it had. Zero takes the default.
	Batch int

	OnProgress func(RecogniseResult)
}

// RecogniseResult reports what recognition did.
type RecogniseResult struct {
	Path    string // the document being read
	Pages   int    // how many it has
	Read    int    // how many have been read, this run and before it
	Resumed int    // how many a run before this one had already read
	Empty   bool   // it says nothing, and nothing was written
}

// DefaultBatch is how many pages are read before they are written down.
const DefaultBatch = 16

// Execute reads one document.
func (u Recognise) Execute(ctx context.Context, v domain.Vault, path string) (RecogniseResult, error) {
	res := RecogniseResult{Path: path}
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

	hash := fingerprint(raw)
	area := u.area()
	final, partial := text.Artifact(area, hash), text.Partial(area, hash)

	// A document already read is not read again. The name is the hash of what
	// was read, so this holds however the file was renamed or moved since.
	if _, err := store.Read(ctx, final); err == nil {
		res.Read, res.Pages = 1, 1
		return res, u.claim(ctx, v, ref, hash, final)
	}

	scan, err := pdf.Open(raw)
	if err != nil {
		return res, fmt.Errorf("%s: %w", path, err)
	}
	defer scan.Close()
	res.Pages = scan.Pages()

	done, err := resumed(ctx, store, partial)
	if err != nil {
		return res, err
	}
	res.Resumed, res.Read = done, done
	u.progress(res)

	pages := make([]ocr.Page, 0, u.batch())
	for index := done; index < scan.Pages(); index++ {
		if err := ctx.Err(); err != nil {
			// What has been read is on disk already. Stopping is a thing a
			// person did, and the next run begins where this one stopped.
			return res, err
		}
		image, err := scan.Image(index, u.By.Recognition().DPI)
		if err != nil {
			return res, fmt.Errorf("draw page %d of %s: %w", index+1, path, err)
		}
		blocks, err := u.By.Read(ctx, image)
		if err != nil {
			return res, fmt.Errorf("read page %d of %s: %w", index+1, path, err)
		}
		pages = append(pages, ocr.Page{Label: scan.Label(index), Blocks: blocks})

		res.Read = index + 1
		if len(pages) >= u.batch() {
			if err := store.Append(ctx, partial, written(pages, res.Read)); err != nil {
				return res, err
			}
			pages = pages[:0]
		}
		u.progress(res)
	}
	if len(pages) > 0 {
		if err := store.Append(ctx, partial, written(pages, res.Read)); err != nil {
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
	if prose, _ := ocr.Read(whole); strings.TrimSpace(prose) == "" {
		// A document that says nothing writes nothing. An empty artifact would
		// stand in for a text layer that worked, and there is no falling back
		// from one.
		res.Empty = true
		return res, store.Remove(ctx, partial)
	}

	if err := store.Write(ctx, final, whole); err != nil {
		return res, err
	}
	if err := store.Remove(ctx, partial); err != nil {
		return res, err
	}
	if err := u.record(ctx, store, area, hash); err != nil {
		return res, err
	}
	return res, u.claim(ctx, v, ref, hash, final)
}

// claim records that this source's text is now the artifact's.
//
// No recipe is written, so the source owes its text: what cuts it into windows
// is extraction, which knows the sizes and is the one place that does.
func (u Recognise) claim(ctx context.Context, v domain.Vault, ref domain.FileRef, hash, name string) error {
	return u.Sources.SaveSource(ctx, v.ID, port.Source{Ref: ref, Hash: hash, TextPath: name})
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

// written is a run of pages and how many of the document have been read,
// together, so that a run stopped part way can be taken up again.
//
// The count is a comment: it is a line the artifact's own reader passes over,
// because a page mark is what it looks for.
func written(pages []ocr.Page, read int) []byte {
	return append([]byte(fmt.Sprintf("%s%d\n", resumeMark, read)), ocr.Write(pages)...)
}

// resumeMark begins the line that says how much of a document has been read.
const resumeMark = ocr.Note + "pages "

// resumed is how many pages of a document a run before this one read.
func resumed(ctx context.Context, store port.DerivedStore, partial string) (int, error) {
	raw, err := store.Read(ctx, partial)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	// The last count written is the one that counts: every batch appends one,
	// and the run stopped after the last of them.
	at := strings.LastIndex(string(raw), resumeMark)
	if at < 0 {
		return 0, nil
	}
	line := string(raw[at+len(resumeMark):])
	if end := strings.IndexByte(line, '\n'); end >= 0 {
		line = line[:end]
	}
	read, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil {
		return 0, nil
	}
	return read, nil
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

func (u Recognise) progress(res RecogniseResult) {
	if u.OnProgress != nil {
		u.OnProgress(res)
	}
}
