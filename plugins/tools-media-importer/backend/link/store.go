package link

import (
	"context"
	"encoding/json"
	"errors"
	"path"
	"strings"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/captions"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/download"
)

// What is kept for a link, in the plugin's folder of the vault's store, under the key of the link:
// the content hash of its file.
//
//   - `<key>.txt` is the prose of an article, and `<key>.vtt` the captions of a video.
//   - `<key>.json` says what fetched the text and what it is called.
//   - `<key>.none` is an address that publishes no text, or a link whose text a person deleted. The
//     host's asking for text leaves it be; a person asking fetches again.
//   - `<key>.part` is a copy arriving, and `<key><ext>` the copy whole, `.mp4` or what the site
//     publishes.
const (
	proseExt   = ".txt"
	captionExt = ".vtt"
	recordExt  = ".json"
	noneExt    = ".none"
	partExt    = ".part"
)

// copyExts are the containers a copy is kept in.
var copyExts = []string{".mp4", ".m4a", ".webm", ".mp3", ".ogg", ".wav", ".flac"}

// texts are what is kept for the links of one vault.
type texts struct {
	store host.Store
}

// record is what fetched the text of a link.
type record struct {
	Address    string `json:"address"`
	Title      string `json:"title,omitempty"`
	Producer   string `json:"producer"`
	Downloader string `json:"downloader"`
}

// readSegments is the text kept under the key as the segments of a text version, and its kind:
// the prose whole, or one segment to a caption. A key with no text gets host.ErrNotExist.
func (t texts) readSegments(ctx context.Context, key string) ([]domain.TextSegment, domain.TextKind, error) {
	raw, err := t.store.Read(ctx, key+captionExt)
	if err == nil {
		return getCaptionSegments(captions.Parse(raw)), kindCaptions, nil
	}
	if !errors.Is(err, host.ErrNotExist) {
		return nil, "", err
	}
	raw, err = t.store.Read(ctx, key+proseExt)
	if err != nil {
		return nil, "", err
	}
	return []domain.TextSegment{{Text: string(raw)}}, kindArticle, nil
}

// The kinds of the text versions of links.
const (
	kindArticle  domain.TextKind = "article"
	kindCaptions domain.TextKind = "transcript"
)

// versionExt ends the file of a link's text in the common format of text versions.
const versionExt = ".jsonl"

// writeText keeps what was fetched under the key, the captions or the prose, with its record, and
// takes away what an earlier fetch kept of the other kind.
func (t texts) writeText(ctx context.Context, key string, fetched download.Text, rec record) error {
	name, other, content := key+proseExt, key+captionExt, []byte(fetched.Prose)
	if len(fetched.Cues) > 0 {
		name, other, content = key+captionExt, key+proseExt, captions.Marshal(fetched.Cues)
	}
	raw, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	return errors.Join(
		t.store.Write(ctx, name, content),
		t.store.Write(ctx, key+recordExt, raw),
		t.store.Delete(ctx, other),
		t.store.Delete(ctx, key+noneExt),
	)
}

// readRecord is what fetched the text kept under the key, and the zero record where none is kept.
func (t texts) readRecord(ctx context.Context, key string) (record, error) {
	raw, err := t.store.Read(ctx, key+recordExt)
	if errors.Is(err, host.ErrNotExist) {
		return record{}, nil
	}
	if err != nil {
		return record{}, err
	}
	var rec record
	return rec, json.Unmarshal(raw, &rec)
}

// markNone marks the key as an address that publishes no text.
func (t texts) markNone(ctx context.Context, key string) error {
	return t.store.Write(ctx, key+noneExt, nil)
}

// isNone reports whether the key is marked as one the host's asking does not fetch.
func (t texts) isNone(ctx context.Context, key string) (bool, error) {
	_, err := t.store.Read(ctx, key+noneExt)
	if errors.Is(err, host.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

// findCopy is the name and size of the copy kept under the key, and false where none is.
func (t texts) findCopy(ctx context.Context, key string) (host.StoreFile, bool, error) {
	files, err := t.store.List(ctx)
	if err != nil {
		return host.StoreFile{}, false, err
	}
	for _, one := range files {
		if isCopyOf(one.Name, key) {
			return one, true, nil
		}
	}
	return host.StoreFile{}, false, nil
}

// isCopyOf reports whether a file of the folder is the whole copy kept under the key.
func isCopyOf(name, key string) bool {
	ext := strings.ToLower(path.Ext(name))
	return strings.TrimSuffix(name, path.Ext(name)) == key && containsExt(ext)
}

func containsExt(ext string) bool {
	for _, one := range copyExts {
		if one == ext {
			return true
		}
	}
	return false
}

// deleteCopy takes away the copy kept under the key, whole or arriving.
func (t texts) deleteCopy(ctx context.Context, key string) error {
	files, err := t.store.List(ctx)
	if err != nil {
		return err
	}
	var errs []error
	for _, one := range files {
		if isCopyOf(one.Name, key) || one.Name == key+partExt {
			errs = append(errs, t.store.Delete(ctx, one.Name))
		}
	}
	return errors.Join(errs...)
}

// getCaptionSegments is the captions as the segments of a text version, one line to a caption,
// each with its span in seconds as its selector.
func getCaptionSegments(cues []download.Cue) []domain.TextSegment {
	segments := make([]domain.TextSegment, len(cues))
	for i, cue := range cues {
		segments[i] = domain.TextSegment{
			Text:     cue.Text + "\n",
			Selector: host.Selector{T: []float64{float64(cue.From) / 1000, float64(cue.To) / 1000}}.Encode(),
		}
	}
	return segments
}
