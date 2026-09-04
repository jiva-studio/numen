package source

import (
	"bytes"
	"context"
	"encoding/hex"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// A vector is bought once and reclaimed by every vault that holds the same
// text. What the coarse pass reads is the same either way, so whether a passage
// reaches the rerank is settled by its text.
func TestAReclaimedVectorCarriesTheCoarseFormItWasBoughtWith(t *testing.T) {
	index, shelf := newStore(), newLibrary()
	shelf.hold(bookPath, domain.KindBook, bookOf(t, "A Book", words(sanskrit, 900)), 1)

	embed := Embed{
		Readers: vaults{string(first.ID): shelf, string(second.ID): shelf},
		Chunks:  index, Vectors: index,
		Embedder: &faint{dims: dimensions}, BatchCharacters: 4000,
	}

	cutBooks(t, index, shelf, first)
	if _, err := embed.Execute(t.Context(), first); err != nil {
		t.Fatal(err)
	}

	// What the index now holds, as a second vault finds it.
	bought := map[string]port.Vector{}
	index.kept = map[string][]byte{}
	for _, held := range index.vectors {
		for _, v := range held {
			hash := hex.EncodeToString(v.Hash)
			bought[hash] = v
			index.kept[v.Model.Recipe()+"/"+hash] = v.Value
		}
	}
	if len(bought) == 0 {
		t.Fatal("nothing was bought")
	}

	index.groups = nil
	cutBooks(t, index, shelf, second)
	res, err := embed.Execute(t.Context(), second)
	if err != nil {
		t.Fatal(err)
	}
	if res.Reused == 0 {
		t.Fatal("the second vault bought every vector again")
	}

	var read int
	for _, group := range index.groups {
		for _, v := range group {
			was, held := bought[hex.EncodeToString(v.Hash)]
			if !held {
				continue
			}
			read++
			if !bytes.Equal(v.Coarse, was.Coarse) {
				t.Fatalf("a reclaimed vector carries %d coarse bytes that differ from the %d it was bought with",
					differing(v.Coarse, was.Coarse), len(was.Coarse))
			}
		}
	}
	if read == 0 {
		t.Fatal("no reclaimed vector was written")
	}
}

// faint answers with a direction that leans on one dimension, so the rest sit
// below half a quantised step and are stored as zero. Which way a stored zero
// leans is what the two coarse forms have to agree about.
type faint struct{ dims int }

func (f *faint) Model() port.EmbeddingModel {
	return port.EmbeddingModel{Name: "faint", Dimensions: f.dims}
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

// differing is how many bytes of two coarse forms are not the same.
func differing(a, b []byte) int {
	var n int
	for i := range a {
		if i < len(b) && a[i] != b[i] {
			n++
		}
	}
	return n
}
