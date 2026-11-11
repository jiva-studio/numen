package document

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/pdf"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/textlayer"
)

// ErrText is the failure to read the live text version of a file.
var ErrText = errors.New("read the text of")

// mostSpans is the most ranges one request for highlights may hold.
const mostSpans = 64

// LiveTexts reads the live text version of a file.
type LiveTexts interface {
	ListVersions(ctx context.Context, path string) (host.TextListing, error)
	ReadText(ctx context.Context, path string, id domain.TextVersionID, cursor string) (host.TextPage, error)
}

// RangeError is a request for ranges that are too many or are not ranges of a text.
type RangeError struct {
	Message string
}

func (e *RangeError) Error() string { return e.Message }

// GetHighlights places each range of the file's live text version on the pages and returns the
// places of each range, in the order of the ranges. A file with no live version, or a version
// without places, has no places.
func (r *Reader) GetHighlights(ctx context.Context, vault domain.VaultID, path string, ranges []domain.ByteRange) ([][]Place, error) {
	if _, err := r.files(vault).StatFile(ctx, path); errors.Is(err, host.ErrOutside) || errors.Is(err, host.ErrNoVault) {
		return nil, err
	}
	end, err := checkRanges(ranges)
	if err != nil {
		return nil, err
	}
	text, err := readLiveText(ctx, r.texts(vault), path, end)
	if err != nil {
		return nil, fmt.Errorf("%w %s: %w", ErrText, path, err)
	}
	out := make([][]Place, 0, len(ranges))
	for _, run := range ranges {
		out = append(out, GetPlaces(textlayer.GetBoxes(text, pdf.ByteRange{Start: run.Start, End: run.End})))
	}
	return out, nil
}

func checkRanges(ranges []domain.ByteRange) (end int, err error) {
	if len(ranges) == 0 || len(ranges) > mostSpans {
		return 0, &RangeError{Message: fmt.Sprintf("expected between one and %d ranges, got %d", mostSpans, len(ranges))}
	}
	for _, run := range ranges {
		if run.Start < 0 || run.End <= run.Start {
			return 0, &RangeError{Message: fmt.Sprintf("%d..%d is not a range of the text", run.Start, run.End)}
		}
		end = max(end, run.End)
	}
	return end, nil
}

func readLiveText(ctx context.Context, texts LiveTexts, path string, end int) (domain.VersionText, error) {
	listing, err := texts.ListVersions(ctx, path)
	if errors.Is(err, host.ErrNotExist) {
		return domain.VersionText{Text: ""}, nil
	}
	if err != nil {
		return domain.VersionText{}, err
	}
	i := slices.IndexFunc(listing.Versions, func(v domain.TextVersion) bool { return v.GetID() == listing.Live })
	if listing.Live == "" || i < 0 {
		return domain.VersionText{Text: ""}, nil
	}
	live := listing.Versions[i]
	var parts []domain.TextSegment
	size, cursor := 0, ""
	for {
		page, err := texts.ReadText(ctx, path, live.GetID(), cursor)
		if err != nil {
			return domain.VersionText{}, err
		}
		parts = append(parts, page.Segments...)
		for _, part := range page.Segments {
			size += len(part.Text)
		}
		if page.NextCursor == "" || page.NextCursor == cursor || size >= end {
			return domain.NewVersionText(live, parts), nil
		}
		cursor = page.NextCursor
	}
}
