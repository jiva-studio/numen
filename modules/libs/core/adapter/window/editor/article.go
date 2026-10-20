package editor

import (
	"context"
	"fmt"
	"unicode/utf8"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/internal/transcript"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// articleHandler answers with the prose a page is written around.
type articleHandler struct {
	files         *artifactFiles
	getTranscript source.GetTranscript
}

// ReadArticle answers with the prose a page is written around. A file nothing
// has been fetched for holds none, which is an answer.
func (h *articleHandler) ReadArticle(
	ctx context.Context,
	r *connect.Request[v1.ReadArticleRequest],
) (*connect.Response[v1.ReadArticleResponse], error) {
	showing, ref, err := h.files.getArtifactFile(
		ctx, r.Msg.GetPath(), v1.ArtifactKind_ARTIFACT_KIND_ARTICLE)
	if err != nil {
		return nil, err
	}
	standing, err := h.getTranscript.Execute(ctx, showing, ref.Path)
	if err != nil {
		return nil, connect.NewError(getReachCode(err), err)
	}
	whole, _ := transcript.Parse(standing.Words)
	prose, err := readSpan(whole, r.Msg.GetSpan())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&v1.ReadArticleResponse{Text: prose}), nil
}

// readSpan is the run of the prose a request named, and the whole of it where a
// request named none. A run is held within the prose and stands on whole
// characters.
func readSpan(prose string, at *v1.Span) (string, error) {
	if at == nil {
		return prose, nil
	}
	span := domain.ByteSpan{From: int(at.GetFrom()), To: int(at.GetTo())}
	if span.From < 0 {
		return "", fmt.Errorf("from: %d is not a place in the prose", span.From)
	}
	if span.Empty() {
		return "", fmt.Errorf("to: %d is not the end of a run of the prose", span.To)
	}
	start, end := min(span.From, len(prose)), min(span.To, len(prose))
	for start > 0 && !utf8.RuneStart(prose[start]) {
		start--
	}
	for end < len(prose) && !utf8.RuneStart(prose[end]) {
		end++
	}
	return prose[start:end], nil
}
