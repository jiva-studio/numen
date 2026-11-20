package engine

import (
	"context"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/domain/profile"
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
	files, err := Locate(t.Context(), profile.Profile{ID: "owner/repo", Revision: "commit", OnnxFile: "onnx/model.onnx", SHA256: sums}, fetch)
	if err != nil || files.Model != "/files/onnx/model.onnx" || files.Tokenizer != "/files/tokenizer.json" {
		t.Fatalf("Locate = %+v, %v", files, err)
	}
}
