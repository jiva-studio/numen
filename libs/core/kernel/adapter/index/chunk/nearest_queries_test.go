package chunk_test

import (
	"bytes"
	"slices"
	"testing"

	sourcedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func TestTheCoarsePassAnswersTheNearestBitsFirstAndNoMoreThanTheLimit(t *testing.T) {
	queries := queriesOf(meaningFixture(t))

	ids, err := queries.FindCoarseCandidates(t.Context(), vaultA, sourcedomain.CoarseQuery{Recipe: recipe, Vector: seedVector(0xF0), Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	found, err := queries.GetEnclosingPassages(t.Context(), vaultA, ids)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"lib/near.epub", "lib/near.epub"}; !slices.Equal(getSources(found), want) {
		t.Errorf("the two nearest are in %v, want %v", getSources(found), want)
	}
}

func TestTheCoarsePassStaysInItsPathPrefixAndVault(t *testing.T) {
	queries := queriesOf(meaningFixture(t))

	ids, err := queries.FindCoarseCandidates(t.Context(), vaultA, sourcedomain.CoarseQuery{Recipe: recipe, Vector: seedVector(0xF0), Limit: 10, PathPrefix: "notes/"})
	if err != nil {
		t.Fatal(err)
	}
	found, err := queries.GetEnclosingPassages(t.Context(), vaultA, ids)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"notes/far.md", "notes/far.md"}; !slices.Equal(getSources(found), want) {
		t.Errorf("under notes/ the candidates are in %v, want %v", getSources(found), want)
	}

	whole, err := queries.FindCoarseCandidates(t.Context(), vaultB, sourcedomain.CoarseQuery{Recipe: recipe, Vector: seedVector(0xF0), Limit: 10})
	if err != nil || len(whole) != 2 {
		t.Errorf("the other vault answered %d candidates, %v, want its 2", len(whole), err)
	}
}

func TestTheCoarsePassHasNothingToSayWithoutAQueryOrAVault(t *testing.T) {
	queries := queriesOf(meaningFixture(t))

	for name, c := range map[string]struct {
		vault domain.VaultID
		query []float32
	}{
		"no vector":     {vaultA, nil},
		"unknown vault": {nobody, seedVector(0xF0)},
	} {
		ids, err := queries.FindCoarseCandidates(t.Context(), c.vault, sourcedomain.CoarseQuery{Recipe: recipe, Vector: c.query, Limit: 5})
		if err != nil || ids != nil {
			t.Errorf("%s: answered %v, %v", name, ids, err)
		}
	}
}

func TestStoredVectorsAreThoseOfTheRecipeAndKindsAsked(t *testing.T) {
	queries := queriesOf(meaningFixture(t))
	ids, err := queries.FindCoarseCandidates(t.Context(), vaultA, sourcedomain.CoarseQuery{Recipe: recipe, Vector: seedVector(0xF0), Limit: 20})
	if err != nil {
		t.Fatal(err)
	}

	all, err := queries.ListStoredVectors(t.Context(), sourcedomain.StoredVectorQuery{Recipe: recipe, Chunks: ids})
	if err != nil || len(all) != len(ids) {
		t.Fatalf("got %d vectors, %v, want one for each of the %d chunks", len(all), err, len(ids))
	}
	near, err := queries.ListStoredVectors(t.Context(), sourcedomain.StoredVectorQuery{Recipe: recipe, Chunks: ids[:1]})
	if err != nil || len(near) != 1 || !bytes.Equal(near[0].Vector, seedBytes(0xF0)) || near[0].Chunk != ids[0] {
		t.Errorf("the nearest chunk holds %+v, %v, want the bytes it was saved with", near, err)
	}
	notes, err := queries.ListStoredVectors(t.Context(), sourcedomain.StoredVectorQuery{Recipe: recipe, Kinds: []domain.SourceKind{domain.KindNote}, Chunks: ids})
	if err != nil || len(notes) != 2 {
		t.Errorf("the notes hold %d vectors, %v, want 2", len(notes), err)
	}
	other, err := queries.ListStoredVectors(t.Context(), sourcedomain.StoredVectorQuery{Recipe: "another model", Chunks: ids})
	if err != nil || len(other) != 0 {
		t.Errorf("a recipe nothing was made under holds %d vectors, %v", len(other), err)
	}
	none, err := queries.ListStoredVectors(t.Context(), sourcedomain.StoredVectorQuery{Recipe: recipe})
	if err != nil || none != nil {
		t.Errorf("no chunks hold %v, %v", none, err)
	}
	if _, err := queries.ListStoredVectors(t.Context(), sourcedomain.StoredVectorQuery{Recipe: recipe, Chunks: []domain.ChunkID{"not a row"}}); err == nil {
		t.Error("a chunk identifier of no index was accepted")
	}
}

func TestAnEnclosingPassageIsFoundOnlyInItsOwnVault(t *testing.T) {
	queries := queriesOf(meaningFixture(t))
	ids, err := queries.FindCoarseCandidates(t.Context(), vaultB, sourcedomain.CoarseQuery{Recipe: recipe, Vector: seedVector(0xF0), Limit: 5})
	if err != nil {
		t.Fatal(err)
	}

	inA, err := queries.GetEnclosingPassages(t.Context(), vaultA, ids)
	if err != nil || len(inA) != 0 {
		t.Errorf("another vault enclosed %v, %v", getSources(inA), err)
	}
	inNobody, err := queries.GetEnclosingPassages(t.Context(), nobody, ids)
	if err != nil || inNobody != nil {
		t.Errorf("an unknown vault enclosed %v, %v", inNobody, err)
	}
	if _, err := queries.GetEnclosingPassages(t.Context(), vaultB, []domain.ChunkID{"not a row"}); err == nil {
		t.Error("a chunk identifier of no index was accepted")
	}
}
