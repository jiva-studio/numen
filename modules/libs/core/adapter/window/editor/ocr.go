package editor

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/highlight"
)

// ReadOcr answers with what a model read off a document's pages: for each run
// of the text asked about, what it says and the boxes covering it.
func (a *API) ReadOcr(
	ctx context.Context,
	r *connect.Request[v1.ReadOcrRequest],
) (*connect.Response[v1.ReadOcrResponse], error) {
	showing, ref, err := a.carrying(ctx, r.Msg.GetPath(), v1.ArtifactKind_ARTIFACT_KIND_OCR)
	if err != nil {
		return nil, err
	}
	runs, err := places(r.Msg.GetSpans())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// A reading is read off the disk, and a layer is asked of the document,
	// which waits for one of the pool. A caller is answered or told to ask
	// again; it is not held while a recognition has every worker.
	ctx, cancel := context.WithTimeout(ctx, patience)
	defer cancel()

	found, err := a.Highlight.Execute(ctx, showing, ref.Path, runs)
	if err != nil {
		return nil, connect.NewError(refusedDrawing(err), err)
	}

	out := &v1.ReadOcrResponse{Runs: make([]*v1.Run, 0, len(found))}
	for _, one := range found {
		out.Runs = append(out.Runs, &v1.Run{Text: one.Text, Boxes: boxed(one.Boxes)})
	}
	return connect.NewResponse(out), nil
}

// boxed is where a run of the text was read, as a caller reads it.
func boxed(boxes []highlight.Box) []*v1.Box {
	out := make([]*v1.Box, 0, len(boxes))
	for _, one := range boxes {
		out = append(out, &v1.Box{
			Span: &v1.Span{From: int32(one.From), To: int32(one.To)},
			Page: int32(one.Page),
			Rect: &v1.Rect{
				MinX: one.MinX, MinY: one.MinY, MaxX: one.MaxX, MaxY: one.MaxY,
			},
		})
	}
	return out
}

// longestRun is the most text one question about a place may cover. A passage
// is a few hundred characters; a run of a million asks where the whole book is,
// one page at a time.
const longestRun = 100_000

// places is which parts of the source's text a caller is asking about, in the
// order they were asked about.
func places(at []*v1.Span) ([]domain.Span, error) {
	if len(at) == 0 || len(at) > domain.MostHighlights {
		return nil, fmt.Errorf("ask about between one and %d places, not %d", domain.MostHighlights, len(at))
	}
	runs := make([]domain.Span, 0, len(at))
	for _, one := range at {
		span := domain.Span{From: int(one.GetFrom()), To: int(one.GetTo())}
		if span.From < 0 {
			return nil, fmt.Errorf("from: %d is not a place in the text", span.From)
		}
		if span.Len() < 1 || span.Len() > longestRun {
			return nil, fmt.Errorf("to: %d is not the end of a run of the text", span.To)
		}
		runs = append(runs, span)
	}
	return runs, nil
}
