package search_test

import (
	"context"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/usecase/search"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/textport"
)

// versionTexts is one text version every path reads as.
type versionTexts struct{ text domain.VersionText }

func (t versionTexts) Open(domain.Vault) (textport.VaultTexts, error) { return t, nil }

func (t versionTexts) FindLiveVersion(context.Context, string, string) (domain.TextVersionID, error) {
	return t.text.Version.GetID(), nil
}

func (t versionTexts) GetLiveText(context.Context, string, string) (domain.VersionText, error) {
	return t.text, nil
}

func (t versionTexts) GetVersionText(_ context.Context, _ string, id domain.TextVersionID) (domain.VersionText, error) {
	if id != t.text.Version.GetID() {
		return domain.VersionText{}, textport.ErrNoText
	}
	return t.text, nil
}

func (versionTexts) RequestText(context.Context, string, string) error { return nil }

func TestAPassageOfATextVersionCarriesTheSelectorOfItsSegment(t *testing.T) {
	ctx := t.Context()
	c := newCorpus(t)
	const path = "talks/lecture.mp3"
	segments := []domain.TextSegment{
		{Text: "Opening remarks about the river.\n", Selector: `{"t":[0,2]}`},
		{Text: "The kingfisher dives for its food.\n", Selector: `{"t":[2,5]}`},
	}
	version := domain.TextVersion{Path: path, PluginID: "test.transcriber", Key: "k", Kind: "transcript"}
	text := domain.NewVersionText(version, segments)
	second := len(segments[0].Text)
	if err := c.db.Sources().SaveExtraction(ctx, c.first.ID, domain.SourceChunks{
		Source: domain.Source{
			Fingerprint: domain.Fingerprint{Path: path, Kind: domain.KindRecording, Size: 1},
			Hash:        "h", Recipe: "r", Producer: string(version.GetID()),
		},
		Chunks: []domain.Chunk{
			{Start: 0, Length: second, Text: segments[0].Text},
			{Start: second, Length: len(segments[1].Text), Text: segments[1].Text},
		},
	}); err != nil {
		t.Fatal(err)
	}

	found, err := search.New(c.db.ChunkQueries(), vaultreader.Readers{}, versionTexts{text: text}, nil, 0, nil).
		Execute(ctx, c.first, "kingfisher", search.Parameters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].Source != path {
		t.Fatalf("found %+v, want the lecture", found)
	}
	if found[0].Selector != `{"t":[2,5]}` {
		t.Fatalf("the passage carries selector %q, want the second segment's", found[0].Selector)
	}
}
