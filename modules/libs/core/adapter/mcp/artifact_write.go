package mcp

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/internal/transcript"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// errNotCorrectable is a kind whose corrections are not the text itself. A
// reading's are records keyed to the printed line each one puts right, and the
// coordinates on the page hang off those numbers.
var errNotCorrectable = errors.New(
	"only a transcript is written here; a reading is put right by a proofreading run")

func addArtifactWriteTool(server *sdk.Server, core Core) {
	if core.Sources.Queries == nil || core.Sources.Derived == nil {
		return
	}

	sdk.AddTool(server, &sdk.Tool{
		Name:  "artifact_write",
		Title: "Write an artifact",
		Description: "Write a stretch of a transcript as it should read. What the site " +
			"or the model produced is never rewritten: the corrections go beside it, " +
			"and taking them away brings back what was transcribed.\n\nThe cues you send " +
			"stand in place of every cue that begins inside the stretch they cover, so a " +
			"transcript is corrected a stretch at a time the way `artifact_read` reads " +
			"one. Name `start` and `length`, in milliseconds, only to replace a stretch " +
			"wider than what you send — to take words out, say. Send WebVTT, times and all: a transcript " +
			"without its times is not one, and the times are what a person's click " +
			"in the tab lands on. Only `transcript` is written here — a reading off a " +
			"scan is corrected by a proofreading run, whose corrections are keyed to " +
			"the printed line.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Path   string `json:"path" jsonschema:"the file it was made from"`
		Kind   string `json:"kind" jsonschema:"which of them to write: transcript"`
		Words  string `json:"words" jsonschema:"the stretch as it should read, as WebVTT with the times each cue was said between"`
		Start  int    `json:"start,omitempty" jsonschema:"where the stretch begins, in milliseconds; the first cue sent by default"`
		Length int    `json:"length,omitempty" jsonschema:"how long the stretch is, in milliseconds; what the cues sent cover by default"`
	}) (*sdk.CallToolResult, struct {
		Cues int `json:"cues" jsonschema:"how many cues the transcript now holds"`
	}, error) {
		type out = struct {
			Cues int `json:"cues" jsonschema:"how many cues the transcript now holds"`
		}
		if in.Kind != kindTranscript {
			return nil, out{}, errNotCorrectable
		}
		_, put := transcript.Parse([]byte(in.Words))
		if len(put) == 0 {
			return nil, out{}, errors.New("no cue was read out of those words")
		}

		held, err := artifactsOf(ctx, core, in.Path)
		if err != nil {
			return nil, out{}, err
		}
		for _, one := range held {
			if one.Kind != kindTranscript {
				continue
			}
			cues, err := corrected(ctx, core, in.Path, one.Producer, put, in.Start, in.Length)
			if err != nil {
				return nil, out{}, err
			}
			return nil, out{Cues: cues}, nil
		}
		return nil, out{}, fmt.Errorf("nothing of the kind %q was made from %s", in.Kind, in.Path)
	})
}

// corrected writes the transcript with those cues standing in place of the ones
// the stretch covers, and answers with how many it now holds.
func corrected(
	ctx context.Context, core Core, path, producer string,
	put []transcript.Cue, start, length int,
) (int, error) {
	words, err := wordsOf(ctx, core, path, producer)
	if err != nil {
		return 0, err
	}
	hash, err := hashOf(ctx, core, path)
	if err != nil || hash == "" {
		return 0, err
	}
	store, err := core.Sources.Derived.Open(core.shown().Vault)
	if err != nil {
		return 0, err
	}

	// A run appends to what it is writing down, and what is being appended to is
	// not corrected underneath.
	release, err := store.Claim(ctx, text.Partial(producer, hash))
	if errors.Is(err, port.ErrClaimed) {
		return 0, errors.New("a run is writing this transcript down")
	}
	if err != nil {
		return 0, err
	}
	defer release()

	_, stood := transcript.Parse([]byte(words))
	whole := standing(stood, put, start, length)
	written := append(transcript.Marshal(whole), transcript.Hand()...)
	if err := store.Write(ctx, text.Corrections(producer, hash), written); err != nil {
		return 0, err
	}
	if core.Sources.Changed != nil {
		core.Sources.Changed(path)
	}
	return len(whole), nil
}

// standing is the cues as they now read: the ones outside the stretch as they
// were, and the ones sent in place of every cue beginning inside it.
func standing(stood, put []transcript.Cue, start, length int) []transcript.Cue {
	// The cues sent are what they cover: a call naming no stretch replaces what
	// stands between the first of them and the last, and never the whole of it.
	if length <= 0 {
		start, length = put[0].From, put[len(put)-1].To-put[0].From
	}
	end := start + length
	out := make([]transcript.Cue, 0, len(stood)+len(put))
	for _, one := range stood {
		if one.From < start {
			out = append(out, one)
		}
	}
	out = append(out, put...)
	for _, one := range stood {
		if one.From >= end {
			out = append(out, one)
		}
	}
	return out
}
