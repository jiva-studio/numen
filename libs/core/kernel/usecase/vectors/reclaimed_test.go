package vectors

import (
	"bytes"
	"context"
	"encoding/hex"
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/embedding"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/indexfake"
)

// A vector is computed once and reclaimed by every vault that holds the same
// text. What the coarse pass reads is the same either way, so whether a passage
// enters the rerank is determined by its text.
func TestAReclaimedVectorCarriesTheCoarseFormItWasBoughtWith(t *testing.T) {
	index, shelf := indexfake.NewStore(), indexfake.NewLibrary()
	shelf.Put(bookPath, domain.KindFile, indexfake.CreateBook(t, "A Book", indexfake.GetWords(indexfake.GetSanskrit(), 900)), 1)

	embed := newEmbed(indexfake.Vaults{indexfake.GetFirst().ID: shelf, indexfake.GetSecond().ID: shelf}, index, index)
	embed.Embedder = &faint{dims: dimensions}
	embed.BatchCharacters = 4000

	cutBooks(t, index, shelf, indexfake.GetFirst())
	if _, err := embed.Execute(t.Context(), indexfake.GetFirst()); err != nil {
		t.Fatal(err)
	}

	// What the index now holds, as a second vault finds it.
	bought := map[string]embedding.Vector{}
	index.Kept = map[string][]byte{}
	for _, target := range index.Vectors {
		for _, v := range target {
			hash := hex.EncodeToString(v.Hash)
			bought[hash] = v
			index.Kept[v.Model.Recipe()+"/"+hash] = v.Value
		}
	}
	if len(bought) == 0 {
		t.Fatal("nothing was bought")
	}

	index.Groups = nil
	cutBooks(t, index, shelf, indexfake.GetSecond())
	res, err := embed.Execute(t.Context(), indexfake.GetSecond())
	if err != nil {
		t.Fatal(err)
	}
	if res.Reused == 0 {
		t.Fatal("the second vault reclaimed no vector")
	}

	var read int
	for _, group := range index.Groups {
		for _, v := range group {
			was, targetVal := bought[hex.EncodeToString(v.Hash)]
			if !targetVal {
				continue
			}
			read++
			if !bytes.Equal(v.Coarse, was.Coarse) {
				t.Fatalf("a reclaimed vector carries %d coarse bytes that differ from the %d it was bought with",
					countDifferingBytes(v.Coarse, was.Coarse), len(was.Coarse))
			}
		}
	}
	if read == 0 {
		t.Fatal("no reclaimed vector was written")
	}
}

// faint answers with a direction that leans on one dimension, so the rest sit
// below half a quantized step and are stored as zero. Which way a stored zero
// leans is what the two coarse forms have to agree about.
type faint struct{ dims int }

func (f *faint) GetModel() embedding.Model {
	return embedding.Model{Name: "faint", Dimensions: f.dims}
}

func (*faint) Close() error { return nil }

func (f *faint) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, 0, len(texts))
	for i := range texts {
		v := make([]float32, f.dims)
		v[i%f.dims] = 1
		for j := range v {
			if v[j] == 0 {
				v[j] = 1e-6
			}
		}
		out = append(out, v)
	}
	return out, nil
}

func countDifferingBytes(a, b []byte) int {
	var n int
	for i := range a {
		if i < len(b) && a[i] != b[i] {
			n++
		}
	}
	return n
}
