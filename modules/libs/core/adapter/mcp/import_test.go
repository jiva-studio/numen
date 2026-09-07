package mcp_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/mcp"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	derived "github.com/jiva-studio/numen/modules/libs/core/text"
	"github.com/jiva-studio/numen/modules/libs/core/transcript"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

// A site answering what a test puts in it. Nothing here reaches a network.
type site struct {
	title string
	cues  []transcript.Cue
	bytes []byte
}

func (s *site) Fetching(domain.WebAddress) port.FetchModel {
	return port.FetchModel{Tool: "a test", Version: "1"}
}

func (s *site) Metadata(context.Context, domain.WebAddress) (port.Metadata, error) {
	return port.Metadata{Title: s.title, Length: 83_500, Captions: []string{"en"}}, nil
}

func (s *site) Subtitles(
	context.Context, domain.WebAddress, string,
) ([]transcript.Cue, error) {
	if len(s.cues) == 0 {
		return nil, port.ErrNothingFetched
	}
	return s.cues, nil
}

func (s *site) Audio(context.Context, domain.WebAddress, io.Writer) error {
	return port.ErrNothingFetched
}

func (s *site) Download(_ context.Context, _ domain.WebAddress, into io.Writer) (port.Download, error) {
	if len(s.bytes) == 0 {
		return port.Download{}, port.ErrNothingFetched
	}
	_, err := into.Write(s.bytes)
	return port.Download{MediaType: derived.CopyType, Extension: derived.CopyExtension}, err
}

func (s *site) Article(context.Context, domain.WebAddress) (port.Article, error) {
	return port.Article{}, port.ErrNothingFetched
}

// importing is the core with one address it can reach.
func importing(t *testing.T, from *site) (domain.Vault, mcp.Core) {
	t.Helper()
	v, core := served(t)
	core.Notes.Import = &source.ImportURL{
		Readers: filesystem.VaultReaders{},
		Derived: filesystem.DerivedStores{
			Area: derived.Captions, Areas: []string{derived.Article, derived.ASR},
		},
		By: from,
	}
	return v, core
}

const aVideo = "https://www.youtube.com/watch?v=dQw4w9WgXcQ"

// An agent hands over an address and gets a note with what is at it: the words
// published with the video stand under the note's own prose, and a search over
// the vault answers about them from then on.
func TestAnAgentImportsAnAddress(t *testing.T) {
	from := &site{title: "Entropy explained", cues: []transcript.Cue{
		{Text: "what was said", From: 1500, To: 4200},
	}}
	_, core := importing(t, from)
	session := sessionOf(t, mcp.New(core))

	out := call[mcp.ImportOutcome](t, session, "note_import", map[string]any{"url": aVideo})

	if out.Refused != "" {
		t.Fatalf("the address was refused: %s", out.Refused)
	}
	if out.Producer != derived.Captions {
		t.Errorf("the words were fetched by %q", out.Producer)
	}
	if out.Words == 0 {
		t.Error("nothing came back from the address")
	}
	// The note is called what is at the address, and nobody had to name it.
	if out.Title != "Entropy explained" {
		t.Errorf("the note is called %q", out.Title)
	}
	if !strings.HasSuffix(out.Path, ".md") {
		t.Errorf("the note stands at %q", out.Path)
	}
}

// A copy is asked for and not assumed: it is the video itself on somebody's
// disk, and an agent that was not asked for one does not fetch one.
func TestAnAgentAsksForACopy(t *testing.T) {
	from := &site{title: "Entropy explained", bytes: []byte("the bytes of a video")}
	_, core := importing(t, from)
	session := sessionOf(t, mcp.New(core))

	without := call[mcp.ImportOutcome](t, session, "note_import", map[string]any{"url": aVideo})
	if without.CopiedBytes != 0 {
		t.Errorf("a copy of %d bytes was fetched unasked", without.CopiedBytes)
	}

	with := call[mcp.ImportOutcome](t, session, "note_import", map[string]any{
		"url": "https://youtu.be/oHg5SJYRHA0", "copy": true,
	})
	if with.CopyRefused != "" {
		t.Fatalf("the copy was refused: %s", with.CopyRefused)
	}
	if with.CopiedBytes != int64(len(from.bytes)) {
		t.Errorf("the copy is %d bytes", with.CopiedBytes)
	}
}

// A build that cannot reach an address serves no tool that would: an agent is
// told what it can do by what it is offered.
func TestABuildThatReachesNoAddressServesNoImport(t *testing.T) {
	_, core := served(t)
	session := sessionOf(t, mcp.New(core))

	found, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range found.Tools {
		if one.Name == "note_import" {
			t.Error("note_import is served by a build that reaches no address")
		}
	}
}
