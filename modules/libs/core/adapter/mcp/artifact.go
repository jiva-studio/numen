package mcp

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/text"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/file"
)

// The kinds an artifact is asked for by, which are the kinds the store files
// them under.
//
// A kind says what a thing is, and a producer says what made it. Words with the
// times they were said at are a transcript whether a model heard them or a site
// published them. An ocr carries where on which page each word stands, and the
// prose a page is written around carries no such thing, so they are two kinds
// and not one.
const (
	kindReading    = text.Reading
	kindTranscript = text.Transcript
	kindArticle    = text.Article
	kindCopy       = text.Copies
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
	Format   string `json:"format" jsonschema:"what artifact_read gives back, as a media type: text/vtt for a transcript, application/json for a reading off a scan, text/plain for a page.s prose. A copy is not read, and says what it is played as"`
}

// Readable says whether artifact_read hands this back. A copy is bytes to play
// and not words to read.
func (a Artifact) Readable() bool { return a.Kind != kindCopy }

// formatOf is what a producer's artifact is written as, which is what a caller
// reading it parses. A transcript is a subtitle file, a reading off a scan is
// the boxes each word stood in, and prose is prose.
func formatOf(producer string) string {
	switch producer {
	case text.ASR, text.Captions:
		return "text/vtt"
	case text.Reading:
		return "application/json"
	default:
		return "text/plain"
	}
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
		Title: "List artifacts",
		Description: "What this application made from one file of the vault and keeps " +
			"beside it: the text read out of a scan, the words heard in a recording, " +
			"the words a site published with a video, the prose a page is written " +
			"around, and a copy of a video on this disk. None of it is in the file, and " +
			"none of it comes back from `note_read` — a note pointing at a video is " +
			"prose the person wrote, and the words said in the video stand here. Read " +
			"one with `artifact_read`.\n\nA kind says what a thing is and a producer " +
			"says what made it: words with the times they were said at are a " +
			"`transcript` whether a model heard them or a site published them, and the " +
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
		held, err := artifactsOf(ctx, core, in.Path)
		if err != nil {
			return nil, out{}, err
		}
		return nil, out{Artifacts: held}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
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
		held, err := artifactsOf(ctx, core, in.Path)
		if err != nil {
			return nil, out{}, err
		}
		for _, one := range held {
			if one.Kind != in.Kind || !one.Readable() {
				continue
			}
			words, err := wordsOf(ctx, core, in.Path, one.Producer)
			if err != nil {
				return nil, out{}, err
			}
			start, held := rangeOf(words, in.Start, in.Length)
			return nil, out{Artifact: one, Words: held, Start: start, Length: len(held)}, nil
		}
		return nil, out{}, fmt.Errorf("nothing of the kind %q was made from %s", in.Kind, in.Path)
	})
}

// hashOf is the hash everything made from one file stands under. A url stands
// under its address, and the index recorded that when it walked past.
func hashOf(ctx context.Context, core Core, path string) (string, error) {
	said, held, err := core.Sources.Queries.Reading(ctx, core.shown().Vault.ID, path)
	if err != nil || !held {
		return "", err
	}
	return said.Hash, nil
}

// artifactsOf is what the store holds for one file.
func artifactsOf(ctx context.Context, core Core, path string) ([]Artifact, error) {
	hash, err := hashOf(ctx, core, path)
	if err != nil || hash == "" {
		return nil, err
	}
	store, err := core.Sources.Derived.Open(core.shown().Vault)
	if err != nil {
		return nil, err
	}
	out := make([]Artifact, 0, 2)
	for _, producer := range text.Producers() {
		_, size, held, err := fileOf(ctx, store, producer, hash)
		if err != nil {
			return nil, err
		}
		if !held {
			continue
		}
		out = append(out, Artifact{
			Path:     path,
			Kind:     kindOf(producer),
			Producer: producer,
			Bytes:    size,
			Format:   formatOf(producer),
		})
	}
	// A copy of a video is bytes and no producer's text, so it is asked for by
	// name rather than found by what wrote it.
	if _, size, err := store.Open(ctx, text.Copy(hash)); err == nil {
		out = append(out, Artifact{
			Path: path, Kind: kindCopy, Bytes: size, Format: text.CopyType,
		})
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	return out, nil
}

// rangeOf is the range of a text a call asked for, held within it and cut on whole
// characters. A call naming no length takes as much as one answer carries.
func rangeOf(words string, start, length int) (int, string) {
	if length <= 0 || length > file.MostRead {
		length = file.MostRead
	}
	start = max(0, min(start, len(words)))
	length = min(length, len(words)-start)
	at, raw := file.WholeCharacters(start, []byte(words[start:start+length]), len(words))
	return at, string(raw)
}

// fileOf is the file one producer's artifact stands in, and its size.
func fileOf(
	ctx context.Context, store port.DerivedStore, producer, hash string,
) (name string, size int64, held bool, err error) {
	for _, one := range []string{
		text.Corrections(producer, hash),
		text.Artifact(producer, hash),
		text.Partial(producer, hash),
	} {
		reader, size, err := store.Open(ctx, one)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", 0, false, err
		}
		_ = reader.Close()
		return one, size, true, nil
	}
	return "", 0, false, nil
}

// wordsOf is the artifact as it stands on disk. A transcript is the WebVTT it
// is written in, times and all.
func wordsOf(ctx context.Context, core Core, path, producer string) (string, error) {
	hash, err := hashOf(ctx, core, path)
	if err != nil || hash == "" {
		return "", err
	}
	store, err := core.Sources.Derived.Open(core.shown().Vault)
	if err != nil {
		return "", err
	}
	name, _, held, err := fileOf(ctx, store, producer, hash)
	if err != nil || !held {
		return "", err
	}
	raw, err := store.Read(ctx, name)
	return string(raw), err
}
