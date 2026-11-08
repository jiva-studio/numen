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
// key of the link's address, and so are the copy and the page.
//
//   - `<key>.jsonl` is the text, in the common format of text versions. The host deletes it with
//     the version.
//   - `<key>.json` records what fetched the text and what kind it is. A record whose text is gone
//     marks a version the user deleted.
//   - `<key>.part` is a copy arriving, and `<key>.video<ext>` or `<key>.audio<ext>` a copy whole. One
//     copy of each is kept per address, shared by the link notes of that address, and deleted
//     once no link note refers to it.
//   - `<key>.html` is the page that frames the player of the video.
const (
	textExt   = ".jsonl"
	recordExt = ".json"
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

// record is what fetched the text of a link.
type record struct {
	Address    string          `json:"address"`
	Title      string          `json:"title,omitempty"`
	Producer   string          `json:"producer,omitempty"`
	Downloader string          `json:"downloader,omitempty"`
	Kind       domain.TextKind `json:"kind,omitempty"`
}

func getKind(fetched download.Text) domain.TextKind {
	if len(fetched.Cues) > 0 {
		return kindCaptions
	}
	return kindArticle
}

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

func (f folder) writeText(ctx context.Context, key string, segments []domain.TextSegment, rec record) error {
	if err := host.WriteText(ctx, f.store, key+textExt, segments); err != nil {
		return err
	}
	return f.writeRecord(ctx, key, rec)
}

func (f folder) writeRecord(ctx context.Context, key string, rec record) error {
	raw, err := json.Marshal(rec)
	// gomutants:disable-next-line BRANCH_IF reason="json.Marshal of string-only record struct cannot fail"
	if err != nil {
		return err
	}
	return f.store.Write(ctx, key+recordExt, raw)
}

func (f folder) findRecord(ctx context.Context, key string) (record, bool, error) {
	raw, err := f.store.Read(ctx, key+recordExt)
	if errors.Is(err, host.ErrNotExist) {
		// gomutants:disable-next-line RETURN_ZERO reason="explicit empty fields equal zero record"
		return record{Address: "", Title: "", Producer: "", Downloader: "", Kind: ""}, false, nil
	}
	if err != nil {
		return record{}, false, err
	}
	rec := record{Address: "", Title: "", Producer: "", Downloader: "", Kind: ""}
	return rec, true, json.Unmarshal(raw, &rec)
}

func (f folder) readRecord(ctx context.Context, key string) (record, error) {
	rec, _, err := f.findRecord(ctx, key)
	return rec, err
}

func (f folder) findCopy(ctx context.Context, key string, media download.Media) (host.StoreFile, bool, error) {
	files, err := f.store.List(ctx)
	if err != nil {
		return host.StoreFile{}, false, err
	}
	for _, one := range files {
		if isCopyOf(one.Name, key) && strings.TrimSuffix(one.Name, path.Ext(one.Name)) == key+"."+string(media) {
			return one, true, nil
		}
	}
	// gomutants:disable-next-line RETURN_ZERO reason="explicit empty fields equal zero StoreFile"
	return host.StoreFile{Name: "", Size: 0}, false, nil
}

func (f folder) listCopies(ctx context.Context, key string) ([]string, error) {
	files, err := f.store.List(ctx)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, one := range files {
		if isCopyOf(one.Name, key) {
			names = append(names, one.Name)
		}
	}
	return names, nil
}

func getCopyName(key string, media download.Media, ext string) string {
	return key + "." + string(media) + ext
}

func isCopyOf(name, key string) bool {
	ext := path.Ext(name)
	switch strings.TrimSuffix(name, ext) {
	case key + "." + string(download.MediaVideo), key + "." + string(download.MediaAudio):
		return isCopyExt(strings.ToLower(ext))
	}
	return false
}

func isCopyExt(ext string) bool {
	switch ext {
	case ".mp4", ".m4a", ".webm", ".mp3", ".ogg", ".wav", ".flac":
		return true
	}
	return false
}
