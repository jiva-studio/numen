package neighborhood

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/note"
)

func TestNeighborhoodInternalsAndErrors(t *testing.T) {
	ctx := t.Context()
	v := domain.Vault{ID: "v1"}
	errBoom := errors.New("boom")

	if MostRead != 30 {
		t.Fatalf("expected MostRead == 30, got %d", MostRead)
	}

	// Test ShowNeighborhood.Execute error branches: linked error, GetNotes error, GetTypes error,
	// boundary len(out.Notes) == MostRead vs MostRead+1.
	snLinkErr := NewShow(
		note.NewShowLinks(stubLinkQueries{linksErr: errBoom}),
		stubNoteQueries{},
		note.NewRead(stubVaultReaders{}),
	)
	if _, err := snLinkErr.Execute(ctx, v, "decks/A.md"); !errors.Is(err, errBoom) {
		t.Fatalf("expected linked error, got %v", err)
	}

	linksOne := stubLinkQueries{
		links: map[string][]domain.ResolvedLink{
			"decks/A.md": {
				{Link: domain.Link{Target: domain.Address{Scheme: domain.SchemeNote, Value: "Note1.md"}}, To: "Note1.md", ToVault: v.ID},
			},
		},
	}
	snNotesErr := NewShow(
		note.NewShowLinks(linksOne),
		stubNoteQueries{notesErr: errBoom},
		note.NewRead(stubVaultReaders{}),
	)
	if _, err := snNotesErr.Execute(ctx, v, "decks/A.md"); !errors.Is(err, errBoom) {
		t.Fatalf("expected GetNotes error, got %v", err)
	}

	snTypesErr := NewShow(
		note.NewShowLinks(linksOne),
		stubNoteQueries{typesErr: errBoom},
		note.NewRead(stubVaultReaders{}),
	)
	if _, err := snTypesErr.Execute(ctx, v, "decks/A.md"); !errors.Is(err, errBoom) {
		t.Fatalf("expected GetTypes error, got %v", err)
	}

	// Test exactly MostRead (30) notes -> Unread == 0, and readBodies skipping empty Path before reading later note.
	var thirtyLinks []domain.ResolvedLink
	notesMap := make(map[string]domain.NoteRef, 30)
	typesMap := make(map[string]domain.NoteType, 30)
	thirtyLinks = append(thirtyLinks, domain.ResolvedLink{
		Link: domain.Link{Target: domain.Address{Scheme: domain.SchemeName, Value: "Dangling"}},
		To:   "",
	})
	for i := 1; i < 30; i++ {
		p := "Note" + string(rune('A'+i)) + ".md"
		thirtyLinks = append(thirtyLinks, domain.ResolvedLink{
			Link:    domain.Link{Target: domain.Address{Scheme: domain.SchemeNote, Value: p}},
			To:      p,
			ToVault: v.ID,
		})
		notesMap[p] = domain.NoteRef{Path: p, Title: p}
	}
	readerOk := stubVaultReader{
		statFn: func(_ context.Context, path string) (domain.FileStat, error) {
			return domain.FileStat{Path: path, Kind: domain.KindNote, Size: 5}, nil
		},
		readFn: func(_ context.Context, _ string) ([]byte, error) {
			return []byte("prose"), nil
		},
	}
	snThirty := NewShow(
		note.NewShowLinks(stubLinkQueries{links: map[string][]domain.ResolvedLink{"decks/A.md": thirtyLinks}}),
		stubNoteQueries{notes: notesMap, types: typesMap},
		note.NewRead(stubVaultReaders{reader: readerOk}),
	)
	gotThirty, err := snThirty.Execute(ctx, v, "decks/A.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(gotThirty.Notes) != 30 || gotThirty.Unread != 0 {
		t.Fatalf("expected 30 notes and Unread=0, got len=%d Unread=%d", len(gotThirty.Notes), gotThirty.Unread)
	}
	if gotThirty.Notes[1].Body != "prose" {
		t.Fatalf("expected second neighbor body read after dangling first neighbor, got %q", gotThirty.Notes[1].Body)
	}

	// Test getNeighbors duplicate backlinks & seen link, scheme Note vs Name vs http, and getNeighborTitle missing note.
	neighbors := getNeighbors(v, "decks/A.md", note.Links{
		Links: []domain.ResolvedLink{
			{Link: domain.Link{Target: domain.Address{Scheme: "https", Value: "example.com"}}},
			{Link: domain.Link{Target: domain.Address{Scheme: domain.SchemeNote, Value: "N1.md"}}, To: "N1.md", ToVault: v.ID},
			{Link: domain.Link{Target: domain.Address{Scheme: domain.SchemeNote, Value: "N1.md"}}, To: "N1.md", ToVault: v.ID},
		},
		Backlinks: []domain.ResolvedLink{
			{From: "N1.md"},
			{Link: domain.Link{Label: "first"}, From: "N2.md"},
			{Link: domain.Link{Label: "duplicate"}, From: "N2.md"},
			{Link: domain.Link{Label: "after"}, From: "N3.md"},
		},
	})
	if len(neighbors) != 3 || neighbors[0].Path != "N1.md" || neighbors[1].Path != "N2.md" || neighbors[1].Label != "first" || neighbors[2].Path != "N3.md" {
		t.Fatalf("unexpected getNeighbors deduplication: %+v", neighbors)
	}
	if _, ok := getNeighborTitle(Neighbor{Path: "Gone.md"}, map[string]domain.NoteRef{}, map[string]domain.NoteType{}); ok {
		t.Fatal("expected missing note in index to not be included")
	}
}
