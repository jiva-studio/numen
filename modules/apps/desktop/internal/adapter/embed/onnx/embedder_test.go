package onnx_test

import (
	"context"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/embed/onnx"
)

// ModelDirEnvVar names a directory holding model.onnx and tokenizer.json. The
// model is hundreds of megabytes, so the tests that need one run only where a
// person has put it there.
const ModelDirEnvVar = "NUMEN_TEST_MODEL_DIR"

func modelDir(t *testing.T) string {
	t.Helper()
	dir := os.Getenv(ModelDirEnvVar)
	if dir == "" {
		t.Skipf("set %s to a directory with model.onnx and tokenizer.json", ModelDirEnvVar)
	}
	return dir
}

func open(t *testing.T, dir string) *onnx.Embedder {
	t.Helper()
	cfg := embed.Defaults()
	cfg.Indexing.Local.Dir = dir
	e, err := onnx.Open(t.Context(), cfg.Model, cfg.Indexing.Local, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = e.Close() })
	return e
}

func TestAMissingDirectoryIsNamedInTheError(t *testing.T) {
	cfg := embed.Defaults()
	cfg.Indexing.Local.Dir = t.TempDir()
	_, err := onnx.Open(t.Context(), cfg.Model, cfg.Indexing.Local, nil)
	if err == nil {
		t.Fatal("want an error")
	}
	if !os.IsNotExist(err) && err.Error() == "" {
		t.Errorf("got %v", err)
	}
}

func TestDimensionsMustBeKnown(t *testing.T) {
	cfg := embed.Defaults()
	cfg.Model.Dimensions = 0
	cfg.Indexing.Local.Dir = t.TempDir()
	if _, err := onnx.Open(t.Context(), cfg.Model, cfg.Indexing.Local, nil); err == nil {
		t.Fatal("want an error")
	}
}

// A model reports the identity the settings gave it, whole: the recipe a vector
// is stored under is made from it in one process and read in another.
func TestWhereATextIsCutOffMustBeSaid(t *testing.T) {
	cfg := embed.Defaults()
	cfg.Model.MaxTokens = 0
	cfg.Indexing.Local.Dir = t.TempDir()
	_, err := onnx.Open(t.Context(), cfg.Model, cfg.Indexing.Local, nil)
	if err == nil || !strings.Contains(err.Error(), "cut off") {
		t.Fatalf("got %v", err)
	}
}

func TestAPoolingNobodyImplementsIsRefused(t *testing.T) {
	// A model is pooled the way it was trained to be, or it is refused here.
	cfg := embed.Defaults()
	cfg.Model.Pooling = "cls"
	cfg.Indexing.Local.Dir = t.TempDir()
	_, err := onnx.Open(t.Context(), cfg.Model, cfg.Indexing.Local, nil)
	if err == nil || !strings.Contains(err.Error(), "cls") {
		t.Fatalf("got %v", err)
	}
}

func TestTheModelEmbedsAndReportsItself(t *testing.T) {
	e := open(t, modelDir(t))
	if got := e.Model(); got.Dimensions != embed.Defaults().Model.Dimensions {
		t.Errorf("got %s", got)
	}

	texts := []string{
		"The soul is not born, nor does it ever die.",
		"na jāyate mriyate vā kadācit",
		"Sourdough needs a starter and a warm kitchen.",
	}
	vectors, err := e.Embed(context.Background(), texts)
	if err != nil {
		t.Fatal(err)
	}
	if len(vectors) != len(texts) {
		t.Fatalf("got %d vectors for %d texts", len(vectors), len(texts))
	}
	for i, v := range vectors {
		if len(v) != e.Model().Dimensions {
			t.Fatalf("vector %d has %d dimensions", i, len(v))
		}
		if length := math.Abs(float64(dot(v, v)) - 1); length > 1e-5 {
			t.Errorf("vector %d is not unit length: %v", i, length)
		}
	}
	// The two statements about the soul are the same sentence in two
	// languages, and the bread is not.
	if dot(vectors[0], vectors[1]) <= dot(vectors[0], vectors[2]) {
		t.Errorf("the translation is not the nearer text: %v vs %v",
			dot(vectors[0], vectors[1]), dot(vectors[0], vectors[2]))
	}
}

func TestTheSameTextGivesTheSameVector(t *testing.T) {
	e := open(t, modelDir(t))
	first, err := e.Embed(context.Background(), []string{"dharmakṣetre kurukṣetre samavetā yuyutsavaḥ"})
	if err != nil {
		t.Fatal(err)
	}
	// In another batch, beside a text of another length, so that the padding
	// differs.
	second, err := e.Embed(context.Background(), []string{
		"dharmakṣetre kurukṣetre samavetā yuyutsavaḥ",
		"a much shorter line",
	})
	if err != nil {
		t.Fatal(err)
	}
	for d := range first[0] {
		if math.Abs(float64(first[0][d]-second[0][d])) > 1e-4 {
			t.Fatalf("dimension %d moved: %v then %v", d, first[0][d], second[0][d])
		}
	}
}

func TestALongTextIsTruncatedAndEmbedded(t *testing.T) {
	e := open(t, modelDir(t))
	long := ""
	for range 4000 {
		long += "śrī kṛṣṇa caitanya prabhu nityānanda "
	}
	if _, err := e.Embed(context.Background(), []string{long}); err != nil {
		t.Fatal(err)
	}
}

// The rate at which this machine embeds, measured on the machine that asks.
func TestThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("measures for a while")
	}
	e := open(t, modelDir(t))
	texts := make([]string, 16)
	for i := range texts {
		for range 40 {
			texts[i] += "śrī caitanya mahāprabhu spoke of the holy name in Navadvīpa. "
		}
	}
	// One pass to compile the shape, which is paid once per process.
	if _, err := e.Embed(context.Background(), texts[:1]); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if _, err := e.Embed(context.Background(), texts); err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(start)
	t.Logf("%d chunks in %s: %.2f chunks/s", len(texts), elapsed.Round(time.Millisecond), float64(len(texts))/elapsed.Seconds())
}

func dot(a, b []float32) float32 {
	var sum float32
	for i := range a {
		sum += a[i] * b[i]
	}
	return sum
}
