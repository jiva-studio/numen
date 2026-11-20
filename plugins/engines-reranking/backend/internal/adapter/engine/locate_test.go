package engine

import (
	"context"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/domain/profile"
)

func TestLocateRequestsTheSelectedModelAndTokenizer(t *testing.T) {
	table, err := profile.GetShippedProfiles()
	if err != nil {
		t.Fatal(err)
	}
	p := table.GetCurated()[0]
	sums := p.SHA256
	fetch := func(_ context.Context, address, sum string) (string, error) {
		name := strings.TrimPrefix(address, "https://huggingface.co/"+p.ID+"/resolve/"+p.Revision+"/")
		if sums[name] != sum {
			t.Fatalf("unexpected file %s with sum %s", name, sum)
		}
		return "/files/" + name, nil
	}
	t.Setenv("HF_ENDPOINT", "")
	files, err := Locate(t.Context(), p, fetch)
	if err != nil || files.Model != "/files/"+p.OnnxFile || files.Tokenizer != "/files/tokenizer.json" {
		t.Fatalf("Locate = %+v, %v", files, err)
	}
}
