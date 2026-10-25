package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf16"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	numenv1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/transcript"
)

// errNoLiveText is a recording no live version gives words to.
var errNoLiveText = errors.New("the recording has no transcript to edit")

// transcriptServer answers TranscriptService for the recordings of the vault the host shows: the
// cues of the live version, whichever plugin made it, and a person's edit as a version of this
// plugin made from the live one.
type transcriptServer struct {
	stores host.Store
	texts  host.Texts
}

var _ numenv1connect.TranscriptServiceHandler = transcriptServer{}

// liveCue is one cue of the live version and where its words begin in the version's text, in
// UTF-16 code units.
type liveCue struct {
	cue    *numenv1.Cue
	offset int
	length int
}

func (s transcriptServer) ReadTranscript(
	ctx context.Context, req *connect.Request[numenv1.ReadTranscriptRequest],
) (*connect.Response[numenv1.ReadTranscriptResponse], error) {
	live, _, err := s.readLive(ctx, req.Msg.GetPath())
	if err != nil {
		return nil, err
	}
	out := make([]*numenv1.Cue, 0, len(live))
	span := req.Msg.GetSpan()
	for _, one := range live {
		if span == nil || (int(span.GetFrom()) < one.offset+one.length && int(span.GetTo()) > one.offset) {
			out = append(out, one.cue)
		}
	}
	return connect.NewResponse(&numenv1.ReadTranscriptResponse{Cues: out}), nil
}

func (s transcriptServer) WriteTranscript(
	ctx context.Context, req *connect.Request[numenv1.WriteTranscriptRequest],
) (*connect.Response[numenv1.WriteTranscriptResponse], error) {
	path := req.Msg.GetPath()
	_, version, err := s.readLive(ctx, path)
	if err != nil {
		return nil, err
	}
	if version == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("%s: %w", path, errNoLiveText))
	}
	cues := readWireCues(req.Msg.GetCues())
	put := host.TextPut{Path: path, Format: domain.TextFormatTimed, Hash: hashCues(cues)}
	if version.PluginID == PluginID && isEditKey(version.Key) {
		put.Key, put.Input, put.InputHash = version.Key, version.Input, version.InputHash
	} else {
		put.Key, put.Input, put.InputHash = getEditKey(version.GetID()), version.GetID(), version.Hash
	}
	if err := (transcripts{store: s.stores}).write(ctx, put.Key, cues); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if _, err := s.texts.PutVersion(ctx, put); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&numenv1.WriteTranscriptResponse{Cues: writeWireCues(cues)}), nil
}

// readLive is the cues of the live version of a file and its record, and no cues and no record
// where no version is live.
func (s transcriptServer) readLive(ctx context.Context, path string) ([]liveCue, *domain.TextVersion, error) {
	listing, err := s.texts.ListVersions(ctx, path)
	if err != nil {
		return nil, nil, connect.NewError(getHostCode(err), err)
	}
	var version *domain.TextVersion
	for i := range listing.Versions {
		if listing.Versions[i].GetID() == listing.Live {
			version = &listing.Versions[i]
		}
	}
	if version == nil || version.Format != domain.TextFormatTimed {
		return nil, nil, nil
	}
	var live []liveCue
	offset := 0
	for cursor := ""; ; {
		page, err := s.texts.ReadText(ctx, path, version.GetID(), cursor)
		if err != nil {
			return nil, nil, connect.NewError(getHostCode(err), err)
		}
		for _, part := range page.Parts {
			length := len(utf16.Encode([]rune(part.Text)))
			live = append(live, liveCue{offset: offset, length: length, cue: &numenv1.Cue{
				Text: strings.TrimRight(part.Text, "\n"),
				From: int32(durationToMS(part.Place.Start)),
				To:   int32(durationToMS(part.Place.End)),
			}})
			offset += length
		}
		if page.NextCursor == "" || page.NextCursor == cursor {
			return live, version, nil
		}
		cursor = page.NextCursor
	}
}

// readWireCues is the cues a person left, with the words of each trimmed. A cue whose words trim
// away is dropped when the cues are written.
func readWireCues(wire []*numenv1.Cue) []transcript.Cue {
	cues := make([]transcript.Cue, 0, len(wire))
	for _, one := range wire {
		text := strings.TrimSpace(one.GetText())
		if text == "" {
			continue
		}
		cues = append(cues, transcript.Cue{Text: text, From: int(one.GetFrom()), To: int(one.GetTo())})
	}
	return cues
}

func writeWireCues(cues []transcript.Cue) []*numenv1.Cue {
	out := make([]*numenv1.Cue, len(cues))
	for i, cue := range cues {
		out[i] = &numenv1.Cue{Text: cue.Text, From: int32(cue.From), To: int32(cue.To)}
	}
	return out
}

// getHostCode is the status a refusal of the host is answered with.
func getHostCode(err error) connect.Code {
	switch {
	case errors.Is(err, host.ErrNotExist):
		return connect.CodeNotFound
	case errors.Is(err, host.ErrNoVault):
		return connect.CodeFailedPrecondition
	}
	return connect.CodeInternal
}
