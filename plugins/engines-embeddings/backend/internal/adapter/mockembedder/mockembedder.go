// Package mockembedder answers with vectors computed from the words of a text, with no model. Automated
// tests select it.
package mockembedder

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"math"
	"math/rand/v2"
	"os"
	"slices"
	"strings"
	"unicode"

	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
)

// mockTermDimensions is how many dimensions of the word part one term takes,
// each with a sign of its own.
const mockTermDimensions = 4

// mockShare is the length of the shared part against the length of the word
// part. The cosine of two mock vectors is (s² + c) / (s² + 1), where c is the
// cosine of their word counts.
const mockShare = 0.9

// Embedder answers with vectors computed from the words of a text, with no
// model. A word listed in getConcept counts as its concept, so a text is near a
// question that names the concept by another word.
type Embedder struct {
	LogPath string
}

// MockHoldWord is the word that makes a mock call run until its context ends.
// End-to-end tests write it into a note to hold a vector pass open.
const MockHoldWord = "numenmockhold"

// Embed returns one unit vector per text. A batch with a text that contains
// MockHoldWord returns only when ctx ends, with its error.
func (m Embedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	m.record(texts)
	if slices.ContainsFunc(texts, isHeld) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	out := make([][]float32, len(texts))
	for i, text := range texts {
		out[i] = embedMock(stripMockPrefix(text))
	}
	return out, nil
}

func stripMockPrefix(text string) string {
	if rest, ok := strings.CutPrefix(text, model.MockQueryPrefix); ok {
		return rest
	}
	return strings.TrimPrefix(text, model.MockDocumentPrefix)
}

func (m Embedder) record(texts []string) {
	if m.LogPath == "" {
		return
	}
	if err := appendLine(m.LogPath, texts); err != nil {
		fmt.Fprintf(os.Stderr, "write the log of the mock model: %v\n", err)
	}
}

func appendLine(path string, texts []string) (err error) {
	line, err := json.Marshal(map[string][]string{"texts": texts})
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	_, err = f.Write(append(line, '\n'))
	return err
}

func isHeld(text string) bool {
	return strings.Contains(strings.ToLower(text), MockHoldWord)
}

// Concurrency is how many batches the mock runs in parallel.
func (Embedder) Concurrency() int { return 1 }

// Close releases nothing.
func (Embedder) Close() error { return nil }

func embedMock(text string) []float32 {
	counts := map[string]int{}
	for _, word := range strings.FieldsFunc(strings.ToLower(text), isNotWordRune) {
		counts[getConcept(word)]++
	}
	// gomutants:disable-next-line INTEGER_INCREMENT reason="an empty term has count zero and adds zero"
	terms := make([]string, 0, len(counts))
	for term := range counts {
		terms = append(terms, term)
	}
	// gomutants:disable-next-line STATEMENT_REMOVE reason="sorting terms fixes float addition order across map iterations"
	slices.Sort(terms)

	v := make([]float32, model.MockSharedDimensions+model.MockWordDimensions)
	var squares float64
	for _, term := range terms {
		n := counts[term]
		addTerm(v[model.MockSharedDimensions:], term, float64(n))
		squares += float64(n * n)
	}
	shared := float32(mockShare * math.Sqrt(squares) / math.Sqrt(model.MockSharedDimensions))
	for i := range model.MockSharedDimensions {
		v[i] = shared
	}
	return model.Normalise(v)
}

func addTerm(v []float32, term string, weight float64) {
	h := fnv.New64a()
	_, _ = h.Write([]byte(term)) // a hash.Hash never returns an error
	rng := rand.New(rand.NewPCG(h.Sum64(), 0))
	step := float32(weight / math.Sqrt(mockTermDimensions))
	for range mockTermDimensions {
		dim := rng.IntN(len(v))
		if rng.Uint64()&1 == 1 {
			v[dim] += step
		} else {
			v[dim] -= step
		}
	}
}

func isNotWordRune(r rune) bool {
	return !unicode.IsLetter(r) && !unicode.IsDigit(r)
}

func getConcept(word string) string {
	switch word {
	case "boat", "ferry", "ship", "vessel":
		return "boat"
	case "car", "automobile", "vehicle":
		return "car"
	case "cat", "feline", "kitten":
		return "cat"
	}
	return word
}
