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

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/engine"
)

// record is what the plugin keeps of the recognition of one file in its folder of the vault's
// store: the content it read, what read it, and the text of each page read so far. A record that is
// not complete is a recognition stopped part way, taken up again where it stopped.
type record struct {
	Path       string   `json:"path"`
	InputHash  string   `json:"inputHash"`
	Recipe     string   `json:"recipe"`
	PageCount  int      `json:"pageCount"`
	Pages      [][]part `json:"pages"`
	IsComplete bool     `json:"isComplete"`
}

// part is one run of a page's text and the rectangles it was read from, in fractions of the page.
type part struct {
	Text    string       `json:"text"`
	Rects   [][4]float32 `json:"rects,omitempty"`
	Section string       `json:"section,omitempty"`
}

// getRecordName is the file of the plugin's folder the recognition of a path is kept in.
func getRecordName(path string) string {
	sum := sha256.Sum256([]byte(path))
	return hex.EncodeToString(sum[:]) + ".json"
}

// readRecord is the record kept for the path, and nil where none is kept.
func readRecord(ctx context.Context, store host.Store, path string) (*record, error) {
	raw, err := store.Read(ctx, getRecordName(path))
	if errors.Is(err, host.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read the recognition of %s: %w", path, err)
	}
	var kept record
	if err := json.Unmarshal(raw, &kept); err != nil {
		return nil, fmt.Errorf("read the recognition of %s: %w", path, err)
	}
	return &kept, nil
}

func writeRecord(ctx context.Context, store host.Store, kept *record) error {
	raw, err := json.Marshal(kept)
	if err != nil {
		return err
	}
	if err := store.Write(ctx, getRecordName(kept.Path), raw); err != nil {
		return fmt.Errorf("keep the recognition of %s: %w", kept.Path, err)
	}
	return nil
}

// isFor reports whether the record was read from the content by what the recipe names. No record is
// read from nothing.
func (r *record) isFor(inputHash, recipe string) bool {
	return r != nil && r.InputHash == inputHash && r.Recipe == recipe
}

// isCompleteFor reports whether the record holds every page of the content.
func (r *record) isCompleteFor(inputHash string) bool {
	return r != nil && r.InputHash == inputHash && r.IsComplete
}

// getTextParts is the parts of one page as the host reads them.
func (r *record) getTextParts(index int) []domain.TextPart {
	out := make([]domain.TextPart, 0, len(r.Pages[index]))
	for _, one := range r.Pages[index] {
		rects := make([]domain.Rect, 0, len(one.Rects))
		for _, rect := range one.Rects {
			rects = append(rects, domain.Rect{MinX: rect[0], MinY: rect[1], MaxX: rect[2], MaxY: rect[3]})
		}
		out = append(out, domain.TextPart{Text: one.Text, Section: one.Section, Place: domain.TextPlace{Page: index, Rects: rects}})
	}
	return out
}

// getText is the whole text of the record, as the host joins its parts.
func (r *record) getText() domain.VersionText {
	var parts []domain.TextPart
	for index := range r.Pages {
		parts = append(parts, r.getTextParts(index)...)
	}
	return domain.NewVersionText(domain.TextVersion{Format: domain.TextFormatPositioned}, parts)
}

// blockGap closes every block of a page, so the blocks and the pages read apart.
const blockGap = "\n\n"

// newPage is the parts of one page the recogniser read. A block is cut at the start of each of its
// boxes, so a part stands in the rectangle of the line it holds; the text before its first box goes
// with the first, and a block with no box is one part standing nowhere. A heading block opens a
// section named by its text.
func newPage(bounds image.Rectangle, blocks []engine.Block) []part {
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

// newBlockParts is the parts of one block that says something, cut at the start of each box.
func newBlockParts(bounds image.Rectangle, block engine.Block) []part {
	boxes := append([]engine.Box(nil), block.Boxes...)
	sort.SliceStable(boxes, func(i, j int) bool { return boxes[i].Span.From < boxes[j].Span.From })
	var out []part
	for i, box := range boxes {
		from, to := 0, len(block.Text)
		if i > 0 {
			from = clampSpan(box.Span.From, len(block.Text))
		}
		if i+1 < len(boxes) {
			to = clampSpan(boxes[i+1].Span.From, len(block.Text))
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

func clampSpan(at, length int) int {
	return min(max(at, 0), length)
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
