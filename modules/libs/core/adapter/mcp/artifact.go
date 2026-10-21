package mcp

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

// Artifact is one thing made from a file of the vault, as an agent is told
// about it.
//
// It is addressed by the file it was made from and the kind it is: what made it
// is named by the store and is nobody's to spell, and where it lies is the
// application's own folder, which nothing outside reaches.
type Artifact struct {
	Path     string `json:"path" jsonschema:"the file it was made from, by the path the vault files it under"`
	Kind     string `json:"kind" jsonschema:"what it is: transcript for words with times, article for a page.s prose, ocr for the text off a scan, copy for the bytes of what is at an address"`
	Producer string `json:"producer" jsonschema:"what made it: ocr, asr, captions, article"`
	Bytes    int64  `json:"bytes" jsonschema:"how large it is"`
	Format   string `json:"format" jsonschema:"what artifact_read gives back, as a media type: text/vtt for a transcript, application/json for a recognition off a scan, text/plain for a page.s prose. A copy is not read, and says what it is played as"`
}

func newArtifact(one source.Artifact) Artifact {
	return Artifact{Path: one.Path, Kind: one.Kind, Producer: one.Producer, Bytes: one.Bytes, Format: one.Format}
}

func newArtifacts(held []source.Artifact) []Artifact {
	if held == nil {
		return nil
	}
	out := make([]Artifact, 0, len(held))
	for _, one := range held {
		out = append(out, newArtifact(one))
	}
	return out
}

func addArtifactTools(server *sdk.Server, core Core) {
	if core.Sources.GetArtifactList == nil || core.Sources.GetArtifactText == nil {
		return
	}

	addTool(server, &sdk.Tool{
		Name:  "artifact_list",
		Title: "List artifacts",
		Description: "What this application made from one file of the vault and keeps " +
			"beside it: the text read out of a scan, a recording transcribed, " +
			"the words a site published with a video, the prose a page is written " +
			"around, and a copy of a video on this disk. None of it is in the file, and " +
			"none of it comes back from `note_read` — a note pointing at a video is " +
			"prose the person wrote, and the words said in the video stand here. Read " +
			"one with `artifact_read`.\n\nA kind says what a thing is and a producer " +
			"says what made it: words with the times they were said at are a " +
			"`transcript` whether a model wrote them down or a site published them, and the " +
			"prose of a page is an `article` where the text of a scan is an `ocr`: " +
			"an ocr carries where on which page each word stands.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Path string `json:"path" jsonschema:"the file to ask about, by the path the vault files it under"`
	}) (*sdk.CallToolResult, struct {
		Artifacts []Artifact `json:"artifacts"`
	}, error) {
		type out = struct {
			Artifacts []Artifact `json:"artifacts"`
		}
		target, err := core.Sources.GetArtifactList.Execute(ctx, core.getCurrentVault().Vault, in.Path)
		if err != nil {
			return nil, out{}, err
		}
		return nil, out{Artifacts: newArtifacts(target)}, nil
	})

	addTool(server, &sdk.Tool{
		Name:  "artifact_read",
		Title: "Read an artifact",
		Description: "A range of one thing made from a file, as it stands on disk, by the " +
			"kind `artifact_list` names it: `transcript` for words with the times they " +
			"were said at, written as WebVTT; `article` for the prose a page is written " +
			"around; `ocr` for the text read off a scan. A copy of a video is bytes a " +
			"person plays and is not read here.\n\nAn hour of speech is longer than one answer carries, " +
			"so ask for the range beginning at the previous range's start plus its length " +
			"to read on. The answer says how long the whole of it is.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Path   string `json:"path" jsonschema:"the file it was made from"`
		Kind   string `json:"kind" jsonschema:"which of them to read: transcript, article or ocr"`
		Start  int    `json:"start,omitempty" jsonschema:"where the range begins in the text, in bytes; the beginning by default"`
		Length int    `json:"length,omitempty" jsonschema:"how much to read, in bytes; as much as one call carries by default"`
	}) (*sdk.CallToolResult, struct {
		Artifact
		Words  string `json:"words" jsonschema:"the range of the text, as it now stands"`
		Start  int    `json:"start" jsonschema:"where the range begins, which is what was asked for held within the text"`
		Length int    `json:"length" jsonschema:"how long the range is"`
	}, error) {
		type out = struct {
			Artifact
			Words  string `json:"words" jsonschema:"the range of the text, as it now stands"`
			Start  int    `json:"start" jsonschema:"where the range begins, which is what was asked for held within the text"`
			Length int    `json:"length" jsonschema:"how long the range is"`
		}
		read, err := core.Sources.GetArtifactText.Execute(
			ctx, core.getCurrentVault().Vault, in.Path, in.Kind, in.Start, in.Length)
		if err != nil {
			return nil, out{}, err
		}
		return nil, out{Artifact: newArtifact(read.Artifact), Words: read.Words, Start: read.Start, Length: read.Length}, nil
	})
}
