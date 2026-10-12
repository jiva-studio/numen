package mcp

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

type artifactWriteInput struct {
	Path   string `json:"path" jsonschema:"the file it was made from"`
	Kind   string `json:"kind" jsonschema:"which of them to write: transcript"`
	Words  string `json:"words" jsonschema:"the span as it should read, as WebVTT with the times each cue was said between"`
	Start  int    `json:"start,omitempty" jsonschema:"where the span begins, in milliseconds; the first cue sent by default"`
	Length int    `json:"length,omitempty" jsonschema:"how long the span is, in milliseconds; what the cues sent cover by default"`
}

type artifactWriteOutput struct {
	Cues int `json:"cues" jsonschema:"how many cues the transcript now holds"`
}

func addArtifactWriteTool(server *sdk.Server, core Core) {
	if core.Sources.WriteCorrections == nil {
		return
	}

	addTool(server, &sdk.Tool{
		Name:  "artifact_write",
		Title: "Write an artifact",
		Description: "Write a span of a transcript as it should read. What the site " +
			"or the model produced is never rewritten: the corrections go beside it, " +
			"and taking them away brings back what was transcribed.\n\nThe cues you send " +
			"stand in place of every cue that begins inside the span they cover, so a " +
			"transcript is corrected a span at a time the way `artifact_read` reads " +
			"one. Name `start` and `length`, in milliseconds, only to replace a span " +
			"wider than what you send — to take words out, say. Send WebVTT, times and all: a transcript " +
			"without its times is not one, and the times are what a person's click " +
			"in the tab lands on. Only `transcript` is written here — a recognition off a " +
			"scan is corrected by a proofreading run, whose corrections are keyed to " +
			"the printed line.",
	}, newHandler(func(ctx context.Context, in artifactWriteInput) (artifactWriteOutput, error) {
		cues, err := core.Sources.WriteCorrections.Execute(
			ctx, core.getCurrentVault().Vault, source.Correction{
				Path: in.Path, Kind: in.Kind, Words: in.Words, Start: in.Start, Length: in.Length,
			},
		)
		if err != nil {
			return artifactWriteOutput{}, err
		}
		return artifactWriteOutput{Cues: cues}, nil
	}))
}
