package recognition

import (
	"context"
	"errors"
	"fmt"
	"image"
	"sort"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/domain/progress"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/domain/scan"
)

func readRecord(ctx context.Context, store Folder, path string) (*progress.Record, error) {
	raw, err := store.Read(ctx, progress.GetRecordName(path))
	if errors.Is(err, host.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read the recognition of %s: %w", path, err)
	}
	rec, err := progress.ParseRecord(raw)
	if err != nil {
		return nil, fmt.Errorf("read the recognition of %s: %w", path, err)
	}
	return rec, nil
}

func writeRecord(ctx context.Context, store Folder, rec *progress.Record) error {
	raw, err := rec.Encode()
	if err != nil {
		return err
	}
	if err := store.Write(ctx, progress.GetRecordName(rec.Path), raw); err != nil {
		return fmt.Errorf("keep the progress of the recognition of %s: %w", rec.Path, err)
	}
	return nil
}

// blockGap ends every block of a page.
const blockGap = "\n\n"

func newPage(bounds image.Rectangle, blocks []scan.Block) []progress.Part {
	var out []progress.Part
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

func newBlockParts(bounds image.Rectangle, block scan.Block) []progress.Part {
	boxes := append([]scan.Box(nil), block.Boxes...)
	sort.SliceStable(boxes, func(i, j int) bool { return boxes[i].Range.Start < boxes[j].Range.Start })
	var out []progress.Part
	for i, box := range boxes {
		from, to := 0, len(block.Text)
		if i > 0 {
			from = clampSpan(box.Range.Start, len(block.Text))
		}
		if i+1 < len(boxes) {
			to = clampSpan(boxes[i+1].Range.Start, len(block.Text))
		}
		if to > from {
			out = append(out, progress.Part{Text: block.Text[from:to], Rects: [][4]float32{getFractions(bounds, box.Rect)}, Section: ""})
		}
	}
	if len(out) == 0 {
		return []progress.Part{{Text: block.Text, Rects: nil, Section: ""}}
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
