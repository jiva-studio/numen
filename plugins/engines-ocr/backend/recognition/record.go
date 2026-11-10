package recognition

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"sort"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// record is the progress of a recognition that has not ended, kept in the plugin's folder of the
// vault's store between batches: the content it reads, what reads it, and the text of each page read
// so far. A recognition stopped part way takes it up again where it stopped. The record is deleted
// once the version is put, so the folder holds only the version file of a recognized path.
type record struct {
	Path      string   `json:"path"`
	InputHash string   `json:"inputHash"`
	Recipe    string   `json:"recipe"`
	PageCount int      `json:"pageCount"`
	Pages     [][]part `json:"pages"`
}

// part is one run of a page's text and the rectangles it was read from, in fractions of the page.
type part struct {
	Text    string       `json:"text"`
	Rects   [][4]float32 `json:"rects,omitempty"`
	Section string       `json:"section,omitempty"`
}

func getRecordName(path string) string {
	sum := sha256.Sum256([]byte(path))
	return hex.EncodeToString(sum[:]) + ".progress.json"
}

func getVersionName(path string) string {
	sum := sha256.Sum256([]byte(path))
	return hex.EncodeToString(sum[:]) + ".jsonl"
}

// textKind labels the text versions the plugin creates.
const textKind domain.TextKind = "recognition"

func readRecord(ctx context.Context, store Folder, path string) (*record, error) {
	raw, err := store.Read(ctx, getRecordName(path))
	if errors.Is(err, host.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read the recognition of %s: %w", path, err)
	}
	var rec record
	if err := json.Unmarshal(raw, &rec); err != nil {
		return nil, fmt.Errorf("read the recognition of %s: %w", path, err)
	}
	return &rec, nil
}

func writeRecord(ctx context.Context, store Folder, rec *record) error {
	raw, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	if err := store.Write(ctx, getRecordName(rec.Path), raw); err != nil {
		return fmt.Errorf("keep the progress of the recognition of %s: %w", rec.Path, err)
	}
	return nil
}

func (r *record) isFor(inputHash, recipe string) bool {
	return r != nil && r.InputHash == inputHash && r.Recipe == recipe
}

func (r *record) getTextSegments(index int) ([]domain.TextSegment, error) {
	out := make([]domain.TextSegment, 0, len(r.Pages[index]))
	for _, one := range r.Pages[index] {
		selector, err := pageSelector{Page: index + 1, Boxes: one.Rects}.encode()
		if err != nil {
			return nil, err
		}
		out = append(out, domain.TextSegment{Text: one.Text, Heading: one.Section, Selector: selector})
	}
	return out, nil
}

func (r *record) getText() (domain.VersionText, error) {
	var segments []domain.TextSegment
	for index := range r.Pages {
		page, err := r.getTextSegments(index)
		if err != nil {
			return domain.VersionText{}, err
		}
		segments = append(segments, page...)
	}
	return domain.NewVersionText(domain.TextVersion{
		Path: "", PluginID: "", Key: "", Source: "", Kind: textKind, File: "",
		InputHash: "", TextHash: "", CreateTime: time.Time{},
	}, segments), nil
}

// blockGap ends every block of a page.
const blockGap = "\n\n"

func newPage(bounds image.Rectangle, blocks []Block) []part {
	var out []part
	for _, block := range blocks {
		if block.Text == "" {
			continue
		}
		parts := newBlockParts(bounds, block)
		parts[len(parts)-1].Text += blockGap
		if block.IsHeading {
			parts[0].Section = block.Text
		}
		out = append(out, parts...)
	}
	return out
}

func newBlockParts(bounds image.Rectangle, block Block) []part {
	boxes := append([]Box(nil), block.Boxes...)
	sort.SliceStable(boxes, func(i, j int) bool { return boxes[i].Range.Start < boxes[j].Range.Start })
	var out []part
	for i, box := range boxes {
		from, to := 0, len(block.Text)
		if i > 0 {
			from = clampSpan(box.Range.Start, len(block.Text))
		}
		if i+1 < len(boxes) {
			to = clampSpan(boxes[i+1].Range.Start, len(block.Text))
		}
		if to > from {
			out = append(out, part{Text: block.Text[from:to], Rects: [][4]float32{getFractions(bounds, box.Rect)}, Section: ""})
		}
	}
	if len(out) == 0 {
		return []part{{Text: block.Text, Rects: nil, Section: ""}}
	}
	return out
}

func clampSpan(offset, length int) int {
	return min(max(offset, 0), length)
}

func getFractions(page, rect image.Rectangle) [4]float32 {
	wide, high := float32(page.Dx()), float32(page.Dy())
	if wide <= 0 || high <= 0 {
		return [4]float32{}
	}
	return [4]float32{
		float32(rect.Min.X-page.Min.X) / wide, float32(rect.Min.Y-page.Min.Y) / high,
		float32(rect.Max.X-page.Min.X) / wide, float32(rect.Max.Y-page.Min.Y) / high,
	}
}
