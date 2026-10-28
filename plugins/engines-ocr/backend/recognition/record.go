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

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
)

// record is the progress of a recognition that has not ended, kept in the plugin's folder of the
// vault's store between batches: the content it reads, what reads it, and the text of each page read
// so far. A recognition stopped part way takes it up again where it stopped. The record is deleted
// once the version is put, so the folder holds only the version file of a recognised path.
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

// getRecordName returns the store file name of the progress of a path's recognition.
func getRecordName(path string) string {
	sum := sha256.Sum256([]byte(path))
	return hex.EncodeToString(sum[:]) + ".progress.json"
}

// getVersionName returns the store file name of the text version of a path's recognition.
func getVersionName(path string) string {
	sum := sha256.Sum256([]byte(path))
	return hex.EncodeToString(sum[:]) + ".jsonl"
}

// textKind is the kind of the text versions the plugin makes.
const textKind domain.TextKind = "recognition"

// readRecord returns the progress of the path's recognition, or nil when there is none.
func readRecord(ctx context.Context, store host.Store, path string) (*record, error) {
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

func writeRecord(ctx context.Context, store host.Store, rec *record) error {
	raw, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	if err := store.Write(ctx, getRecordName(rec.Path), raw); err != nil {
		return fmt.Errorf("keep the progress of the recognition of %s: %w", rec.Path, err)
	}
	return nil
}

// isFor reports whether the record was made from the content hash with the recipe.
func (r *record) isFor(inputHash, recipe string) bool {
	return r != nil && r.InputHash == inputHash && r.Recipe == recipe
}

// getTextSegments is the segments of one page as the version file holds them: the page counted
// from 1 and the rectangles of each segment in its selector.
func (r *record) getTextSegments(index int) []domain.TextSegment {
	out := make([]domain.TextSegment, 0, len(r.Pages[index]))
	for _, one := range r.Pages[index] {
		selector := host.Selector{Page: index + 1, Boxes: one.Rects}
		out = append(out, domain.TextSegment{Text: one.Text, Heading: one.Section, Selector: selector.Encode()})
	}
	return out
}

// getText is the whole text of the record, as the host joins its parts.
func (r *record) getText() domain.VersionText {
	var segments []domain.TextSegment
	for index := range r.Pages {
		segments = append(segments, r.getTextSegments(index)...)
	}
	return domain.NewVersionText(domain.TextVersion{Kind: textKind}, segments)
}

// blockGap ends every block of a page.
const blockGap = "\n\n"

// newPage returns the parts of one recognised page. Each block is split at the start of each box, so
// every part has the rectangle of its line; a block without boxes is one part without rectangles. A
// heading block starts a section named by its text.
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

// newBlockParts returns the parts of one non-empty block, split at the start of each box.
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
			out = append(out, part{Text: block.Text[from:to], Rects: [][4]float32{getFractions(bounds, box.Rect)}})
		}
	}
	if len(out) == 0 {
		return []part{{Text: block.Text}}
	}
	return out
}

func clampSpan(offset, length int) int {
	return min(max(offset, 0), length)
}

// getFractions is a rectangle of the page in fractions of it.
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
