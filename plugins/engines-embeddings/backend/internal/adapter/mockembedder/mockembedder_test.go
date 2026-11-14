package mockembedder

import (
	"context"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
)

// floor is the cosine a passage must reach to answer a search by meaning.
const floor = 0.5

func cosine(a, b []float32) float64 {
	var sum float64
	for i := range a {
		sum += float64(a[i]) * float64(b[i])
	}
	return sum
}

func TestMockEmbedderBlocksABatchWithTheHoldWordUntilCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	ended := make(chan error, 1)
	go func() {
		_, err := Embedder{}.Embed(ctx, []string{"a note", "a Note with NumenMockHold in it"})
		ended <- err
	}()

	select {
	case err := <-ended:
		t.Fatalf("the held batch returned before its context ended: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	cancel()
	if err := <-ended; !errors.Is(err, context.Canceled) {
		t.Errorf("the held batch ended with %v, want context.Canceled", err)
	}

	vectors, err := Embedder{}.Embed(t.Context(), []string{"a note"})
	if err != nil || len(vectors) != 1 {
		t.Errorf("a batch without the hold word answered %d vectors and %v", len(vectors), err)
	}
}

func TestEmbedMock_ParaphraseQuestionNearPassage(t *testing.T) {
	question := embedMock("boat")
	passage := embedMock("The ferry leaves the harbor at dawn with the morning post.")
	if got := cosine(question, passage); got < floor {
		t.Errorf("a passage about a ferry stands %.3f from a question about a boat, want at least %.2f", got, floor)
	}
}

func TestAMockQuestionIsFarFromAPassageSharingNoConcept(t *testing.T) {
	question := embedMock("boat")
	passage := embedMock("A measure of uncertainty. To Jaynes it is a property of the description, not of the system.")
	if got := cosine(question, passage); got >= floor {
		t.Errorf("a passage sharing no word with the question stands %.3f from it, want under %.2f", got, floor)
	}
}

func TestMockQuestionIsNearPassageWithItsWord(t *testing.T) {
	question := embedMock("Entropy")
	passage := embedMock("Entropy\n\nA measure of uncertainty. To Jaynes it is a property of the description, not of the system.")
	if got := cosine(question, passage); got < floor {
		t.Errorf("a passage holding the word stands %.3f from the question, want at least %.2f", got, floor)
	}
}

func TestAMockVectorIsUnitLengthAndTheSameEveryTime(t *testing.T) {
	text := "The ferry leaves the harbor at dawn."
	first, second := embedMock(text), embedMock(text)
	if !slices.Equal(first, second) {
		t.Error("one text gave two vectors")
	}
	if got := math.Sqrt(cosine(first, first)); math.Abs(got-1) > 1e-5 {
		t.Errorf("length %.6f, want 1", got)
	}
	if len(first) != model.GetMockModel().Dimensions {
		t.Errorf("got %d dimensions, want %d", len(first), model.GetMockModel().Dimensions)
	}
}

func TestGetMockModelAndEmbedderDetails(t *testing.T) {
	info := model.GetMockModel()
	if info.Dimensions != 2112 || info.MaxTokens != 512 || info.Pooling != model.PoolMean || info.From != "mock" {
		t.Errorf("model.GetMockModel() = %+v", info)
	}
	if info.Concurrency != 1 || info.FloorHint != 0 || info.MinChunkLength != 0 {
		t.Errorf("model.GetMockModel() limits = %+v", info)
	}
	if info.Handling.QueryPrefix != model.MockQueryPrefix || info.Handling.DocumentPrefix != model.MockDocumentPrefix || !info.Handling.Normalize {
		t.Errorf("model.GetMockModel() handling = %+v", info.Handling)
	}

	logFile := filepath.Join(t.TempDir(), "mock.jsonl")
	m := Embedder{LogPath: logFile}
	vectors, err := m.Embed(t.Context(), []string{model.MockQueryPrefix + "boat", model.MockDocumentPrefix + "ferry"})
	if err != nil || len(vectors) != 2 {
		t.Fatalf("Embed() = %d vectors, %v", len(vectors), err)
	}
	if _, err := m.Embed(t.Context(), []string{"second"}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(vectors[0], vectors[1]) || !slices.Equal(vectors[1], embedMock("boat")) {
		t.Error("query and document prefixes were not stripped to the same concept vector")
	}
	logged, err := os.ReadFile(logFile)
	if err != nil || !strings.Contains(string(logged), "passage: ferry") || !strings.Contains(string(logged), "second") {
		t.Errorf("log file = %q, %v", logged, err)
	}
	if info, err := os.Stat(logFile); err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("log file mode = %v, %v, want 0600", info, err)
	}

	origStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	(Embedder{}).record([]string{"quiet"})
	m.record([]string{"valid"})
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stderr = origStderr
	stderrBytes, err := io.ReadAll(r)
	if closeErr := r.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	if err != nil || len(stderrBytes) != 0 {
		t.Errorf("empty or valid LogPath wrote to stderr: %q, %v", stderrBytes, err)
	}

	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = wErr
	(Embedder{LogPath: t.TempDir()}).record([]string{"fail"})
	if err := wErr.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stderr = origStderr
	errBytes, err := io.ReadAll(rErr)
	if closeErr := rErr.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	if err != nil || !strings.Contains(string(errBytes), "write the log of the mock model:") {
		t.Errorf("failing LogPath stderr = %q, %v", errBytes, err)
	}
}

func TestMockEmbedderWaitsForHoldFileAndPollInterval(t *testing.T) {
	holdFile := filepath.Join(t.TempDir(), "hold")
	if err := os.WriteFile(holdFile, []byte("1"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(holdFileVariable, holdFile)

	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := (Embedder{}).Embed(canceled, []string{"boat"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Embed with hold file and canceled ctx = %v, want Canceled", err)
	}

	synctest.Test(t, func(t *testing.T) {
		// In the bubble the fake clock moves only when every goroutine waits, so the two markers
		// bracket the second poll, 50ms after the first.
		var isBeforePoll, isAfterPoll atomic.Bool
		markers := make(chan struct{})
		go func() {
			defer close(markers)
			time.Sleep(49500 * time.Microsecond)
			isBeforePoll.Store(true)
			time.Sleep(time.Millisecond)
			isAfterPoll.Store(true)
		}()
		go func() {
			time.Sleep(10 * time.Millisecond)
			if err := os.Remove(holdFile); err != nil {
				t.Error(err)
			}
		}()
		vectors, err := (Embedder{}).Embed(t.Context(), []string{"boat"})
		if err != nil || len(vectors) != 1 {
			t.Fatalf("Embed after hold file removed = %d, %v", len(vectors), err)
		}
		if !isBeforePoll.Load() || isAfterPoll.Load() {
			t.Errorf("Embed returned with markers (49.5ms %v, 50.5ms %v), want only the 49.5ms one", isBeforePoll.Load(), isAfterPoll.Load())
		}
		<-markers
	})
}

func TestEmbedMockDimensionsTermsAndConcepts(t *testing.T) {
	if got := getConcept("ferry"); got != "boat" {
		t.Errorf("getConcept(ferry) = %q, want boat", got)
	}
	if got := getConcept("automobile"); got != "car" {
		t.Errorf("getConcept(automobile) = %q, want car", got)
	}
	if got := getConcept("kitten"); got != "cat" {
		t.Errorf("getConcept(kitten) = %q, want cat", got)
	}
	if got := getConcept("entropy"); got != "entropy" {
		t.Errorf("getConcept(entropy) = %q, want entropy", got)
	}

	digits := embedMock("42")
	if got := math.Sqrt(cosine(digits, digits)); math.Abs(got-1) > 1e-5 {
		t.Errorf("digit token length = %.6f, want 1", got)
	}

	boat := embedMock("boat")
	if boat[0] <= 0 || boat[63] != boat[0] || boat[64] == boat[0] {
		t.Errorf("shared dimensions boundary mismatch: boat[0]=%f, boat[63]=%f, boat[64]=%f", boat[0], boat[63], boat[64])
	}
	nonZero, posCount, negCount, firstIdx := 0, 0, 0, -1
	for i, val := range boat[64:] {
		if val != 0 {
			nonZero++
			if firstIdx == -1 {
				firstIdx = i
			}
			if val > 0 {
				posCount++
			} else {
				negCount++
			}
		}
	}
	if nonZero != 4 || posCount != 2 || negCount != 2 || firstIdx != 439 || boat[64+firstIdx] >= 0 {
		t.Errorf("word dimensions: nonZero=%d, pos=%d, neg=%d, firstIdx=%d, val=%f", nonZero, posCount, negCount, firstIdx, boat[64+max(firstIdx, 0)])
	}

	accum := []float32{10}
	addTerm(accum, "boat", 2)
	if accum[0] < 5 {
		t.Errorf("addTerm overwrote instead of accumulating: %f", accum[0])
	}
}
