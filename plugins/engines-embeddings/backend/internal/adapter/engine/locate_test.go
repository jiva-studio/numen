package engine

import (
	"context"
	"errors"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocateRequestsTheSelectedModelAndTokenizer(t *testing.T) {
	sums := map[string]string{"onnx/model.onnx": "graph digest", "tokenizer.json": "tokenizer digest"}
	fetch := func(_ context.Context, address, sum string) (string, error) {
		name := strings.TrimPrefix(address, "https://huggingface.co/owner/repo/resolve/commit/")
		if sums[name] != sum {
			t.Fatalf("unexpected file %s with sum %s", name, sum)
		}
		return "/files/" + name, nil
	}
	t.Setenv("HF_ENDPOINT", "")
	files, err := Locate(t.Context(), model.LocalModel{Name: "owner/repo", Revision: "commit", ShouldDownload: true, File: "model.onnx", Sums: sums}, fetch)
	if err != nil || files.Model != "/files/onnx/model.onnx" || files.Tokenizer != "/files/tokenizer.json" {
		t.Fatalf("Locate = %+v, %v", files, err)
	}
}
func TestLocateFromDir(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, model.ModelFile)
	tokenPath := filepath.Join(dir, model.TokenizerFile)
	if err := os.WriteFile(modelPath, []byte("weights"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Locate(t.Context(), model.LocalModel{Dir: dir}, nil); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Locate() with missing tokenizer = %v, want fs.ErrNotExist", err)
	}
	if err := os.WriteFile(tokenPath, []byte("tokens"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := Locate(t.Context(), model.LocalModel{Dir: dir}, nil)
	if err != nil {
		t.Fatalf("Locate() error = %v", err)
	}
	if got.Model != modelPath {
		t.Errorf("model = %q, want %q", got.Model, modelPath)
	}
	if got.Tokenizer != tokenPath {
		t.Errorf("tokenizer = %q, want %q", got.Tokenizer, tokenPath)
	}
}
