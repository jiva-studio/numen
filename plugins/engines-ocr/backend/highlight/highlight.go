// Package highlight answers OcrService: where runs of a file's live text stand on its pages, read
// from the live version through the host whichever plugin made it.
package highlight

import (
	"context"
	"errors"
	"fmt"
	"unicode/utf8"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	numenv1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

// longestRun is the most text one question about a place may cover.
const longestRun = 100_000

// Service answers OcrService over the text versions of the shown vault.
type Service struct {
	numenv1connect.UnimplementedOcrServiceHandler

	Texts host.Texts
}

// ReadOcr answers, for each run of the file's live text asked about, what it says and the boxes
// covering it, in the order the runs were asked about. A file with no live version, or one whose
// version stands nowhere on a page, answers each run lit nowhere.
func (s Service) ReadOcr(
	ctx context.Context, req *connect.Request[numenv1.ReadOcrRequest],
) (*connect.Response[numenv1.ReadOcrResponse], error) {
	runs, err := parseSpans(req.Msg.GetSpans())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	text, err := s.readLiveText(ctx, req.Msg.GetPath())
	if err != nil {
		return nil, err
	}
	out := &numenv1.ReadOcrResponse{Runs: make([]*numenv1.Run, 0, len(runs))}
	for _, run := range runs {
		out.Runs = append(out.Runs, &numenv1.Run{Text: getRunText(text.Text, run), Boxes: newBoxes(text.GetBoxes(run))})
	}
	return connect.NewResponse(out), nil
}

// readLiveText is the file's live version, and no text where none is live.
func (s Service) readLiveText(ctx context.Context, path string) (domain.VersionText, error) {
	listing, err := s.Texts.ListVersions(ctx, path)
	if errors.Is(err, host.ErrNotExist) {
		return domain.VersionText{}, connect.NewError(connect.CodeNotFound, err)
	}
	if err != nil {
		return domain.VersionText{}, connect.NewError(connect.CodeUnavailable, err)
	}
	for _, version := range listing.Versions {
		if version.GetID() != listing.Live {
			continue
		}
		text, err := host.ReadWholeText(ctx, s.Texts, version)
		if err != nil {
			return domain.VersionText{}, connect.NewError(connect.CodeUnavailable, err)
		}
		return text, nil
	}
	return domain.VersionText{}, nil
}

// parseSpans is the runs of the text asked about, in bytes, in the order they were asked about.
func parseSpans(spans []*numenv1.Span) ([]domain.ByteSpan, error) {
	if len(spans) == 0 || len(spans) > domain.MostHighlights {
		return nil, fmt.Errorf("ask about between one and %d places, not %d", domain.MostHighlights, len(spans))
	}
	runs := make([]domain.ByteSpan, 0, len(spans))
	for _, one := range spans {
		run := domain.ByteSpan{From: int(one.GetFrom()), To: int(one.GetTo())}
		if run.From < 0 {
			return nil, fmt.Errorf("from: %d is not a place in the text", run.From)
		}
		if run.Len() < 1 || run.Len() > longestRun {
			return nil, fmt.Errorf("to: %d is not the end of a run of the text", run.To)
		}
		runs = append(runs, run)
	}
	return runs, nil
}

// getRunText is what a run says, widened to whole characters and cut at the end of the text.
func getRunText(text string, run domain.ByteSpan) string {
	from, to := min(run.From, len(text)), min(run.To, len(text))
	for from > 0 && !utf8.RuneStart(text[from]) {
		from--
	}
	for to < len(text) && !utf8.RuneStart(text[to]) {
		to++
	}
	return text[from:to]
}

func newBoxes(boxes []domain.HighlightBox) []*numenv1.Box {
	out := make([]*numenv1.Box, 0, len(boxes))
	for _, one := range boxes {
		out = append(out, &numenv1.Box{
			Span: &numenv1.Span{From: int32(one.From), To: int32(one.To)},
			Page: int32(one.Page),
			Rect: &numenv1.Rect{MinX: one.MinX, MinY: one.MinY, MaxX: one.MaxX, MaxY: one.MaxY},
		})
	}
	return out
}
