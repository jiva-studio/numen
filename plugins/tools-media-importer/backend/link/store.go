package link

import (
	"context"
	"encoding/json"
	"errors"
	"path"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/download"
)

// What is kept for a link in the plugin's folder of the vault's store. The text is kept under the
// key of the link's address, and the copy under the content hash of the note.
//
//   - `<key>.jsonl` is the text, in the common format of text versions. The host deletes it with
//     the version.
//   - `<key>.json` records what fetched the text, what kind it is and which file holds the copy. A
//     record whose text is gone marks a version the user deleted.
//   - `<key>.none` is an address that publishes no text.
//   - `<hash>.part` is a copy arriving, and `<hash><ext>` the copy whole.
const (
	textExt   = ".jsonl"
	recordExt = ".json"
	noneExt   = ".none"
	partExt   = ".part"
)

// The kinds of the text versions of links.
const (
	kindArticle  domain.TextKind = "article"
	kindCaptions domain.TextKind = "captions"
)

// folder is what is kept for the links of one vault.
type folder struct {
	store host.Store
}

// record is what fetched the text of a link, and the file its copy is kept in.
type record struct {
	Address    string          `json:"address"`
	Title      string          `json:"title,omitempty"`
	Producer   string          `json:"producer,omitempty"`
	Downloader string          `json:"downloader,omitempty"`
	Kind       domain.TextKind `json:"kind,omitempty"`
	Copy       string          `json:"copy,omitempty"`
}

// getKind returns the text kind a fetch produced: captions, or the prose of an article.
func getKind(fetched download.Text) domain.TextKind {
	if len(fetched.Cues) > 0 {
		return kindCaptions
	}
	return kindArticle
}

// getSegments is what was fetched as the segments of a text version: one segment to a caption, with
// its span in seconds as its selector, or the prose whole.
func getSegments(fetched download.Text) ([]domain.TextSegment, error) {
	if len(fetched.Cues) == 0 {
		return []domain.TextSegment{{Text: fetched.Prose, Heading: "", Selector: ""}}, nil
	}
	segments := make([]domain.TextSegment, len(fetched.Cues))
	for i, cue := range fetched.Cues {
		selector, err := timeSelector{T: []float64{float64(cue.From) / 1000, float64(cue.To) / 1000}}.encode()
		// gomutants:disable-next-line BRANCH_IF reason="finite float64 slice derived from int cue bounds always marshals to JSON"
		if err != nil {
			return nil, err
		}
		segments[i] = domain.TextSegment{Text: cue.Text + "\n", Heading: "", Selector: selector}
	}
	return segments, nil
}

// writeText keeps the text under the key with its record, and clears the mark of an address that
// publishes none.
func (f folder) writeText(ctx context.Context, key string, segments []domain.TextSegment, rec record) error {
	if err := host.WriteText(ctx, f.store, key+textExt, segments); err != nil {
		return err
	}
	return errors.Join(f.writeRecord(ctx, key, rec), f.store.Delete(ctx, key+noneExt))
}

// hasText reports whether the text of the key is kept.
func (f folder) hasText(ctx context.Context, key string) (bool, error) {
	return f.isKept(ctx, key+textExt)
}

func (f folder) writeRecord(ctx context.Context, key string, rec record) error {
	raw, err := json.Marshal(rec)
	// gomutants:disable-next-line BRANCH_IF reason="json.Marshal of string-only record struct cannot fail"
	if err != nil {
		return err
	}
	return f.store.Write(ctx, key+recordExt, raw)
}

// findRecord is the record kept under the key, and false where none is.
func (f folder) findRecord(ctx context.Context, key string) (record, bool, error) {
	raw, err := f.store.Read(ctx, key+recordExt)
	if errors.Is(err, host.ErrNotExist) {
		// gomutants:disable-next-line RETURN_ZERO reason="explicit empty fields equal zero record"
		return record{Address: "", Title: "", Producer: "", Downloader: "", Kind: "", Copy: ""}, false, nil
	}
	if err != nil {
		return record{}, false, err
	}
	rec := record{Address: "", Title: "", Producer: "", Downloader: "", Kind: "", Copy: ""}
	return rec, true, json.Unmarshal(raw, &rec)
}

// readRecord is the record kept under the key, and the zero record where none is kept.
func (f folder) readRecord(ctx context.Context, key string) (record, error) {
	rec, _, err := f.findRecord(ctx, key)
	return rec, err
}

// markNoText marks the key as an address that publishes no text, and clears the title, producer and
// kind an earlier fetch recorded. The record keeps the address and the copy.
func (f folder) markNoText(ctx context.Context, key string) error {
	rec, hasRecord, err := f.findRecord(ctx, key)
	if err != nil {
		return err
	}
	if hasRecord {
		rec.Title, rec.Producer, rec.Kind = "", "", ""
		if err := f.writeRecord(ctx, key, rec); err != nil {
			return err
		}
	}
	return f.store.Write(ctx, key+noneExt, nil)
}

// isNone reports whether the key is marked as an address that publishes no text.
func (f folder) isNone(ctx context.Context, key string) (bool, error) {
	return f.isKept(ctx, key+noneExt)
}

func (f folder) isKept(ctx context.Context, name string) (bool, error) {
	_, err := f.store.Read(ctx, name)
	if errors.Is(err, host.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

// findCopy is the name and size of the copy kept under the hash, and false where none is.
func (f folder) findCopy(ctx context.Context, hash string) (host.StoreFile, bool, error) {
	files, err := f.store.List(ctx)
	if err != nil {
		return host.StoreFile{}, false, err
	}
	for _, one := range files {
		if isCopyOf(one.Name, hash) {
			return one, true, nil
		}
	}
	// gomutants:disable-next-line RETURN_ZERO reason="explicit empty fields equal zero StoreFile"
	return host.StoreFile{Name: "", Size: 0}, false, nil
}

// isCopyOf reports whether a file of the folder is the whole copy kept under the hash.
func isCopyOf(name, hash string) bool {
	ext := path.Ext(name)
	return strings.TrimSuffix(name, ext) == hash && isCopyExt(strings.ToLower(ext))
}

// isCopyExt reports whether a copy is kept in the container of the extension.
func isCopyExt(ext string) bool {
	switch ext {
	case ".mp4", ".m4a", ".webm", ".mp3", ".ogg", ".wav", ".flac":
		return true
	}
	return false
}

// deleteCopy takes away the copy kept under the hash, whole or arriving.
func (f folder) deleteCopy(ctx context.Context, hash string) error {
	files, err := f.store.List(ctx)
	if err != nil {
		return err
	}
	var errs []error
	for _, one := range files {
		if isCopyOf(one.Name, hash) || one.Name == hash+partExt {
			errs = append(errs, f.store.Delete(ctx, one.Name))
		}
	}
	return errors.Join(errs...)
}
