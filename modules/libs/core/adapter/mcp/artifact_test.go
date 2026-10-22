package mcp_test

import (
	"context"
	"io"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/mcp"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	derived "github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// closeTrackingStores wraps a store so a test can tell whether what it opened
// was closed.
type closeTrackingStores struct {
	inner  port.DerivedStores
	closed map[domain.ArtifactName]bool
}

func (s closeTrackingStores) Open(v domain.Vault) (port.DerivedStore, error) {
	one, err := s.inner.Open(v)
	if err != nil {
		return nil, err
	}
	return closeTrackingStore{DerivedStore: one, closed: s.closed}, nil
}

// closeTrackingStore embeds the store it wraps, so every method but Open
// forwards unchanged.
type closeTrackingStore struct {
	port.DerivedStore
	closed map[domain.ArtifactName]bool
}

func (s closeTrackingStore) OpenArtifact(ctx context.Context, name domain.ArtifactName) (io.ReadSeekCloser, int64, error) {
	reader, size, err := s.DerivedStore.OpenArtifact(ctx, name)
	if err != nil {
		return reader, size, err
	}
	s.closed[name] = false
	return closeTrackingReader{ReadSeekCloser: reader, name: name, closed: s.closed}, size, nil
}

// closeTrackingReader embeds the reader it wraps, so Read and Seek forward
// unchanged and only Close is recorded.
type closeTrackingReader struct {
	io.ReadSeekCloser
	name   domain.ArtifactName
	closed map[domain.ArtifactName]bool
}

func (r closeTrackingReader) Close() error {
	r.closed[r.name] = true
	return r.ReadSeekCloser.Close()
}

// A copy artifact is opened only to ask its size, and the file that answers
// that must not stay open: artifact_list is asked for every file a vault
// carries, over and over, as an agent works.
func TestListingClosesTheFileItOpensForACopy(t *testing.T) {
	from := &site{title: "Entropy explained", bytes: []byte("the bytes of a video")}
	_, core := newCoreWithImport(t, from)

	closed := map[domain.ArtifactName]bool{}
	stores := closeTrackingStores{
		inner: filesystem.DerivedStores{
			Area: filesystem.TranscriptDir, Areas: []string{domain.Article, filesystem.CopyDir},
		},
		closed: closed,
	}
	queries := &sourced{indexed: map[string]port.SourceText{}}
	core.Sources.GetArtifactList = source.NewGetArtifactList(queries, stores)
	core.Sources.GetArtifactText = source.NewGetArtifactText(queries, stores)
	session := sessionOf(t, mcp.New(core))

	made := call[mcp.ImportOutcome](t, session, "url_import", map[string]any{
		"url": aVideo, "copy": true,
	})
	if made.Error != "" || made.CopyError != "" {
		t.Fatalf("the address or its copy was rejected: %q %q", made.Error, made.CopyError)
	}
	queries.indexed[made.Path] = port.SourceText{
		Fingerprint: domain.Fingerprint{Path: made.Path},
		Producer:    domain.Captions,
		Hash:        derived.Fingerprint([]byte(aVideo)),
	}

	items := call[struct {
		Artifacts []mcp.Artifact `json:"artifacts"`
	}](t, session, "artifact_list", map[string]any{"path": made.Path})

	hasCopy := false
	for _, one := range items.Artifacts {
		if one.Kind == "copy" {
			hasCopy = true
		}
	}
	if !hasCopy {
		t.Fatalf("no copy artifact was listed: %+v", items.Artifacts)
	}
	if len(closed) == 0 {
		t.Fatal("artifact_list opened no file")
	}
	for name, ok := range closed {
		if !ok {
			t.Errorf("%s was opened and never closed", name)
		}
	}
}
