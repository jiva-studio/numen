package engine

import (
	"context"
	"os"
	"path"
	"path/filepath"
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

func TestCustomRerankerFetchesOnlyItsGraphAndExternalData(t *testing.T) {
	root := t.TempDir()
	t.Setenv("NUMEN_PLUGIN_CACHE_DIR", root)
	t.Setenv("HF_ENDPOINT", "")
	list := filepath.Join(root, "index.json")
	if err := os.WriteFile(list, []byte(`{"sha":"current","siblings":[{"rfilename":"encoder.onnx"},{"rfilename":"encoder.onnx_data"},{"rfilename":"tokenizer.json"},{"rfilename":"named-weights.bin"},{"rfilename":"pytorch_model.bin"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	fetch := func(_ context.Context, address, sum string) (string, error) {
		if strings.Contains(address, "/api/models/") {
			return list, nil
		}
		name := path.Base(address)
		if name == "pytorch_model.bin" {
			t.Fatal("requested alternate model weights")
		}
		if name == "named-weights.bin" && sum != "owner pin" {
			t.Fatalf("named weight sum = %q", sum)
		}
		file := filepath.Join(root, name)
		if err := os.WriteFile(file, []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
		return file, nil
	}
	files, err := Locate(t.Context(), profile.Profile{ID: "team/model", OnnxFile: "encoder.onnx", SHA256: map[string]string{"named-weights.bin": "owner pin"}}, fetch)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"encoder.onnx_data", "named-weights.bin"} {
		got, err := os.ReadFile(filepath.Join(filepath.Dir(files.Model), name))
		if err != nil || string(got) != name {
			t.Fatalf("%s = %q, %v", name, got, err)
		}
	}
}
