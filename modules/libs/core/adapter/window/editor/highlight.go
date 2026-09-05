package editor

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/highlight"
)

// ListHighlights answers where runs of a source's text sit: the pages each
// falls on and, on each, the rectangles covering it.
func (a *API) ListHighlights(
	ctx context.Context,
	r *connect.Request[v1.ListHighlightsRequest],
) (*connect.Response[v1.ListHighlightsResponse], error) {
	showing := a.Showing()
	if showing.ID == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errNoVault)
	}
	runs, err := places(r.Msg.GetAt())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// A reading is read off the disk, and a layer is asked of the document,
	// which waits for one of the pool. A caller is answered or told to ask
	// again; it is not held while a recognition has every worker.
	ctx, cancel := context.WithTimeout(ctx, patience)
	defer cancel()

	found, err := a.Highlight.Execute(ctx, showing, r.Msg.GetPath(), runs)
	if err != nil {
		return nil, connect.NewError(refusedDrawing(err), err)
	}

	out := &v1.ListHighlightsResponse{Runs: make([]*v1.Highlight, 0, len(found))}
	for _, pages := range found {
		one := &v1.Highlight{Pages: make([]*v1.Page, 0, len(pages))}
		for _, page := range pages {
			on := &v1.Page{Index: int32(page.Index), Rects: make([]*v1.Rect, 0, len(page.Rects))}
			for _, box := range page.Rects {
				on.Rects = append(on.Rects, &v1.Rect{
					MinX: box.MinX, MinY: box.MinY, MaxX: box.MaxX, MaxY: box.MaxY,
				})
			}
			one.Pages = append(one.Pages, on)
		}
		out.Runs = append(out.Runs, one)
	}
	return connect.NewResponse(out), nil
}

// longestRun is the most text one question about a place may cover. A passage
// is a few hundred characters; a run of a million asks where the whole book is,
// one page at a time.
const longestRun = 100_000

// places is which parts of the source's text a caller is asking about, in the
// order they were asked about.
func places(at []*v1.Stretch) ([]highlight.Stretch, error) {
	if len(at) == 0 || len(at) > domain.MostHighlights {
		return nil, fmt.Errorf("ask about between one and %d places, not %d", domain.MostHighlights, len(at))
	}
	runs := make([]highlight.Stretch, 0, len(at))
	for _, one := range at {
		start, length := int(one.GetStart()), int(one.GetLength())
		if start < 0 {
			return nil, fmt.Errorf("start: %d is not a place in the text", start)
		}
		if length < 1 || length > longestRun {
			return nil, fmt.Errorf("length: %d is not a run of the text", length)
		}
		runs = append(runs, highlight.Stretch{Start: start, Length: length})
	}
	return runs, nil
}
