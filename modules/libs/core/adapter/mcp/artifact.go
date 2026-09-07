package mcp

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/text"
)

// The kinds an artifact is asked for by.
//
// A kind says what a thing is, and a producer says what made it. Words with the
// times they were said at are a transcript whether a model heard them or a site
// published them. A reading carries where on which page each word stands, and
// the prose a page is written around carries no such thing, so they are two
// kinds and not one.
const (
	kindReading    = "reading"
	kindTranscript = "transcript"
	kindArticle    = "article"
	kindCopy       = "copy"
)

// Artifact is one thing made from a file of the vault, as an agent is told
// about it.
//
// It is addressed by the file it was made from and the kind it is: what made it
// is named by the store and is nobody's to spell, and where it lies is the
// application's own folder, which nothing outside reaches.
type Artifact struct {
	Path     string `json:"path" jsonschema:"the file it was made from, by the path the vault files it under"`
	Kind     string `json:"kind" jsonschema:"what it is: transcript for words with times, article for a page.s prose, reading for the text off a scan, copy for the bytes of a video"`
	Producer string `json:"producer" jsonschema:"what made it: ocr, asr, captions, article"`
	Bytes    int64  `json:"bytes" jsonschema:"how large it is"`
	Text     bool   `json:"text" jsonschema:"whether artifact_read gives it back as text; a copy is bytes and is played, not read"`
}

// kindOf is which of them a producer.s text is.
func kindOf(producer string) string {
	switch producer {
	case text.ASR, text.Captions:
		return kindTranscript
	case text.Article:
		return kindArticle
	default:
		return kindReading
	}
}

func addArtifactTools(server *sdk.Server, core Core) {
	if core.Sources.Queries == nil || core.Sources.Derived == nil {
		return
	}

	sdk.AddTool(server, &sdk.Tool{
		Name:  "artifact_list",
		Title: "What was made from a file",
		Description: "What this application made from one file of the vault and keeps " +
			"beside it: the text read out of a scan, the words heard in a recording, " +
			"the words a site published with a video, the prose a page is written " +
			"around, and a copy of a video on this disk. None of it is in the file, and " +
			"none of it comes back from `note_read` — a note pointing at a video is " +
			"prose the person wrote, and the words said in the video stand here. Read " +
			"one with `artifact_read`.\n\nA kind says what a thing is and a producer " +
			"says what made it: words with the times they were said at are a " +
			"`transcript` whether a model heard them or a site published them, and the " +
			"prose of a page is an `article` where the text of a scan is a `reading`: " +
			"a reading carries where on which page each word stands.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Path string `json:"path" jsonschema:"the file to ask about, by the path the vault files it under"`
	}) (*sdk.CallToolResult, struct {
		Artifacts []Artifact `json:"artifacts"`
	}, error) {
		type out = struct {
			Artifacts []Artifact `json:"artifacts"`
		}
		held, err := standing(ctx, core, in.Path)
		if err != nil {
			return nil, out{}, err
		}
		return nil, out{Artifacts: held}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:  "artifact_read",
		Title: "Read what was made from a file",
		Description: "The text of one thing made from a file, by the kind " +
			"`artifact_list` names it: `transcript` for words with the times they were " +
			"said at, `article` for the prose a page is written around, `reading` for " +
			"the text read off a scan. A copy of a video is bytes a person plays and " +
			"is not read here.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Path string `json:"path" jsonschema:"the file it was made from"`
		Kind string `json:"kind" jsonschema:"which of them to read: transcript, article or reading"`
	}) (*sdk.CallToolResult, struct {
		Artifact
		Words string `json:"words" jsonschema:"the text, as it now stands"`
	}, error) {
		type out = struct {
			Artifact
			Words string `json:"words" jsonschema:"the text, as it now stands"`
		}
		held, err := standing(ctx, core, in.Path)
		if err != nil {
			return nil, out{}, err
		}
		for _, one := range held {
			if one.Kind != in.Kind || !one.Text {
				continue
			}
			words, err := reading(ctx, core, in.Path)
			if err != nil {
				return nil, out{}, err
			}
			return nil, out{Artifact: one, Words: words}, nil
		}
		return nil, out{}, fmt.Errorf("nothing of the kind %q was made from %s", in.Kind, in.Path)
	})
}

// standing is what the store holds for one file, as the index says what made it.
func standing(ctx context.Context, core Core, path string) ([]Artifact, error) {
	said, held, err := core.Sources.Queries.Reading(ctx, core.shown().Vault.ID, path)
	if err != nil {
		return nil, err
	}
	if !held || said.Producer == "" {
		return nil, nil
	}
	store, err := core.Sources.Derived.Open(core.shown().Vault)
	if err != nil {
		return nil, err
	}
	out := make([]Artifact, 0, 2)
	if words, producer, err := text.Fetched(ctx, store, said.Hash); err == nil && producer != "" {
		out = append(out, Artifact{
			Path:     path,
			Kind:     kindOf(producer),
			Producer: producer,
			Bytes:    int64(len(words)),
			Text:     true,
		})
	}
	// A copy of a video is bytes and no producer's text, so it is asked for by
	// name rather than read out of what the index says.
	if _, size, err := store.Open(ctx, text.Copy(said.Hash)); err == nil {
		out = append(out, Artifact{
			Path:     path,
			Kind:     kindCopy,
			Producer: said.Producer,
			Bytes:    size,
		})
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	return out, nil
}

// reading is the text of what was made from one file, as it now stands.
func reading(ctx context.Context, core Core, path string) (string, error) {
	said, held, err := core.Sources.Queries.Reading(ctx, core.shown().Vault.ID, path)
	if err != nil || !held {
		return "", err
	}
	store, err := core.Sources.Derived.Open(core.shown().Vault)
	if err != nil {
		return "", err
	}
	words, _, err := text.Fetched(ctx, store, said.Hash)
	return words, err
}
