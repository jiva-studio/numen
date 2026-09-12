package mcp_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/mcp"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	derived "github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/internal/transcript"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

// A site answering what a test puts in it. Nothing here reaches a network.
type site struct {
	title string
	cues  []transcript.Cue
	bytes []byte
}

func (s *site) Downloading(domain.URL) port.DownloadModel {
	return port.DownloadModel{Tool: "a test", Version: "1", Producer: derived.Captions}
}

func (s *site) Metadata(context.Context, domain.URL) (port.Metadata, error) {
	return port.Metadata{Title: s.title, Length: 83_500, Captions: []string{"en"}}, nil
}

func (s *site) Text(
	context.Context, domain.URL, port.PreferredCaptions,
) (port.Text, error) {
	if len(s.cues) == 0 {
		return port.Text{}, port.ErrNothingDownloaded
	}
	return port.Text{
		Producer: derived.Captions, Cues: s.cues, Title: s.title, Length: 83_500,
	}, nil
}

func (s *site) Download(_ context.Context, _ domain.URL, into io.Writer) (port.Copy, error) {
	if len(s.bytes) == 0 {
		return port.Copy{}, port.ErrNothingDownloaded
	}
	_, err := into.Write(s.bytes)
	return port.Copy{MediaType: derived.CopyType, Extension: derived.CopyExtension}, err
}

// importing is the core with one address it can reach.
func importing(t *testing.T, from *site) (domain.Vault, mcp.Core) {
	t.Helper()
	v, core := served(t)
	core.Sources.Import = &source.ImportURL{
		Readers: filesystem.VaultReaders{},
		Derived: filesystem.DerivedStores{
			Area: derived.Transcript, Areas: []string{derived.Article, derived.Copies},
		},
		By: from,
	}
	core.Sources.URLs = &source.CreateURL{Writers: filesystem.VaultWriters{}}
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

	out := call[mcp.ImportOutcome](t, session, "url_import", map[string]any{"url": aVideo})

	if out.Refused != "" {
		t.Fatalf("the address was refused: %s", out.Refused)
	}
	if out.Producer != derived.Captions {
		t.Errorf("the words were fetched by %q", out.Producer)
	}
	if out.Bytes == 0 {
		t.Error("nothing came back from the address")
	}
	if !strings.HasSuffix(out.Path, ".url") {
		t.Errorf("it stands at %q", out.Path)
	}
}

// A copy is asked for and not assumed: it is the video itself on somebody's
// disk, and an agent that was not asked for one does not fetch one.
func TestAnAgentAsksForACopy(t *testing.T) {
	from := &site{title: "Entropy explained", bytes: []byte("the bytes of a video")}
	_, core := importing(t, from)
	session := sessionOf(t, mcp.New(core))

	without := call[mcp.ImportOutcome](t, session, "url_import", map[string]any{"url": aVideo})
	if without.CopiedBytes != 0 {
		t.Errorf("a copy of %d bytes was fetched unasked", without.CopiedBytes)
	}

	with := call[mcp.ImportOutcome](t, session, "url_import", map[string]any{
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
		if one.Name == "url_import" {
			t.Error("url_import is served by a build that reaches no address")
		}
	}
}

// A note is prose the person wrote; what was fetched for the address it points
// at stands beside it and is asked for by kind. An agent handed only the prose
// is handed a note about a video with the video missing.
func TestAnAgentReadsWhatWasFetchedForANote(t *testing.T) {
	from := &site{title: "Entropy explained", cues: []transcript.Cue{
		{Text: "what was said", From: 1500, To: 4200},
	}}
	v, core := importing(t, from)
	core.Sources.Derived = filesystem.DerivedStores{
		Area: derived.Transcript, Areas: []string{derived.Article, derived.Copies},
	}
	said := &sourced{said: map[string]port.SourceText{}}
	core.Sources.Queries = said
	session := sessionOf(t, mcp.New(core))
	_ = v

	made := call[mcp.ImportOutcome](t, session, "url_import", map[string]any{"url": aVideo})
	if made.Refused != "" {
		t.Fatalf("the address was refused: %s", made.Refused)
	}
	// The index is what says which producer stands for a path, and a scan is
	// what writes it. This is that row.
	said.said[made.Path] = port.SourceText{
		Fingerprint: domain.Fingerprint{Path: made.Path},
		Producer:    derived.Captions,
		Hash:        derived.Fingerprint([]byte(aVideo)),
	}

	listed := call[struct {
		Artifacts []mcp.Artifact `json:"artifacts"`
	}](t, session, "artifact_list", map[string]any{"path": made.Path})
	if len(listed.Artifacts) == 0 {
		t.Fatal("the note carries nothing an agent can reach")
	}
	// Words with the times they were said at are a transcript, whoever made
	// them: a site publishing them is another producer and not another kind.
	if listed.Artifacts[0].Kind != "transcript" {
		t.Errorf("it carries %q", listed.Artifacts[0].Kind)
	}

	read := call[struct {
		Words string `json:"words"`
	}](t, session, "artifact_read", map[string]any{"path": made.Path, "kind": "transcript"})
	if !strings.Contains(read.Words, "what was said") {
		t.Errorf("the words read %q", read.Words)
	}
}

// sourced is what the index says was made from each path.
type sourced struct {
	port.SourceQueries
	said map[string]port.SourceText
}

func (s *sourced) Reading(
	_ context.Context, _ domain.VaultID, path string,
) (port.SourceText, bool, error) {
	one, held := s.said[path]
	return one, held, nil
}
