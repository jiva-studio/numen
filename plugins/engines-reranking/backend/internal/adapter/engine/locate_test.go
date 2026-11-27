package engine

import (
	"context"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/domain/profile"
)

func TestLocateRequestsTheFilesOfEachShippedModel(t *testing.T) {
	table, err := profile.GetShippedProfiles()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range table.GetCurated() {
		var requested []string
		fetch := Fetch{
			Dependency: func(_ context.Context, id string) (string, error) {
				requested = append(requested, id)
				return "/files/" + id, nil
			},
			File: func(context.Context, string, string) (string, error) {
				t.Fatalf("%s is fetched by address", p.ID)
				return "", nil
			},
		}
		files, err := Locate(t.Context(), p, fetch)
		if err != nil {
			t.Fatalf("%s: %v", p.ID, err)
		}
		if files.Model != "/files/"+p.Dependencies[p.OnnxFile] || files.Tokenizer != "/files/"+p.Dependencies[tokenizerFile] {
			t.Errorf("%s: Locate = %+v", p.ID, files)
		}
		if want := slices.Sorted(maps.Values(p.Dependencies)); !slices.Equal(slices.Sorted(slices.Values(requested)), want) || len(want) != 2 {
			t.Errorf("%s: requested %v, want %v", p.ID, requested, want)
		}
	}
}

func TestCustomRerankerFetchesOnlyItsGraphAndExternalData(t *testing.T) {
	root := t.TempDir()
	t.Setenv("NUMEN_PLUGIN_CACHE_DIR", root)
	t.Setenv("NUMEN_SHARED_CACHE_DIR", root)
	t.Setenv("HF_ENDPOINT", "")
	list := filepath.Join(root, "index.json")
	if err := os.WriteFile(list, []byte(`{"sha":"current",
 "siblings":[
 {"rfilename":"encoder.onnx"},
 {"rfilename":"encoder.onnx_data"},
 {"rfilename":"tokenizer.json"},
 {"rfilename":"named-weights.bin"},
 {"rfilename":"pytorch_model.bin"}]}`), 0o600); err != nil {
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
	files, err := Locate(t.Context(), profile.Profile{ID: "team/model", OnnxFile: "encoder.onnx", SHA256: map[string]string{"named-weights.bin": "owner pin"}}, Fetch{Dependency: nil, File: fetch})
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
