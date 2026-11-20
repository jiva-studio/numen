package onnx

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareModelPreservesExternalDataPaths(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"metadata": `{"sha":"immutable",
 "siblings":[
 {"rfilename":"onnx/a.onnx","lfs":{"sha256":"lfs digest"}},
 {"rfilename":"onnx/a.onnx_data"},
 {"rfilename":"onnx/a.onnx/other"},
 {"rfilename":"onnx/b.onnx_data"},
 {"rfilename":"onnx/b.onnx"},
 {"rfilename":"onnx/chunks/weights.bin"},
 {"rfilename":"tokenizer.json"}]}`,
		"graph": "model bytes", "tokens": "tokenizer bytes", "data": "external bytes",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	type result struct{ file, digest string }
	answers := map[string]result{
		"https://huggingface.co/api/models/team/repo/revision/immutable?blobs=true":  {"metadata", ""},
		"https://huggingface.co/team/repo/resolve/immutable/onnx/chunks/weights.bin": {"data", ""},
		"https://huggingface.co/team/repo/resolve/immutable/onnx/a.onnx":             {"graph", "graph pin"},
		"https://huggingface.co/team/repo/resolve/immutable/onnx/a.onnx_data":        {"data", ""},
		"https://huggingface.co/team/repo/resolve/immutable/tokenizer.json":          {"tokens", "owner pin"},
	}
	fetch := func(_ context.Context, url, sum string) (string, error) {
		got, ok := answers[url]
		if !ok || got.digest != sum {
			t.Fatalf("unexpected request %q with %q", url, sum)
		}
		return filepath.Join(root, got.file), nil
	}
	spec := RepositoryModel{ID: "team/repo", Revision: "immutable", Graph: "onnx/a.onnx", Sums: map[string]string{"tokenizer.json": "owner pin", "onnx/a.onnx": "graph pin"}}
	for range 2 {
		located, err := PrepareModel(t.Context(), spec, root, fetch)
		if err != nil {
			t.Fatal(err)
		}
		if located.Tokenizer != filepath.Join(root, "tokens") || filepath.Base(located.Model) != "a.onnx" {
			t.Fatalf("paths = %+v", located)
		}
		for name, body := range map[string]string{"a.onnx": "model bytes", "a.onnx_data": "external bytes", "chunks/weights.bin": "external bytes"} {
			b, err := os.ReadFile(filepath.Join(filepath.Dir(located.Model), name))
			if err != nil || string(b) != body {
				t.Fatalf("%s = %q, %v", name, b, err)
			}
		}
	}
}

func TestPrepareModelReportsFetchFailuresAndRejectsInvalidNames(t *testing.T) {
	failed := errors.New("model unavailable")
	fetch := func(context.Context, string, string) (string, error) { return "", failed }
	_, err := PrepareModel(t.Context(), RepositoryModel{ID: "team/repo", Graph: "onnx/model.onnx"}, "", fetch)
	if !errors.Is(err, failed) {
		t.Fatalf("fetch error = %v", err)
	}
	_, err = PrepareModel(t.Context(), RepositoryModel{ID: "team/repo", Graph: "../outside"}, "", fetch)
	if err == nil || errors.Is(err, failed) {
		t.Fatalf("invalid path = %v", err)
	}
}
