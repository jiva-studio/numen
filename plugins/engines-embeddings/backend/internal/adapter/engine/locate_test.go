package engine

import (
	"context"
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/plugintest"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
)

func TestLocateRequestsTheFilesOfEachShippedModel(t *testing.T) {
	plugintest.GrantStorage(t)
	table, err := model.GetShippedProfiles()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range table.GetCurated() {
		_, local := model.NewConfig(p)
		files := t.TempDir()
		var requested []string
		fetch := Fetch{
			Dependency: func(_ context.Context, id string) (string, error) {
				requested = append(requested, id)
				stored := filepath.Join(files, id)
				return stored, os.WriteFile(stored, []byte(id), 0o644)
			},
			File: func(context.Context, string, string) (string, error) {
				t.Fatalf("%s is fetched by address", p.ID)
				return "", nil
			},
		}
		got, err := Locate(t.Context(), local, fetch)
		if err != nil {
			t.Fatalf("%s: %v", p.ID, err)
		}
		if want := slices.Sorted(maps.Values(p.Dependencies)); !slices.Equal(slices.Sorted(slices.Values(requested)), want) {
			t.Errorf("%s: requested %v, want %v", p.ID, requested, want)
		}
		if got.Tokenizer != filepath.Join(files, p.Dependencies[model.TokenizerFile]) {
			t.Errorf("%s: tokenizer = %q", p.ID, got.Tokenizer)
		}
		if filepath.Base(got.Model) != filepath.Base(p.OnnxFile) && got.Model != filepath.Join(files, p.Dependencies[p.OnnxFile]) {
			t.Errorf("%s: graph = %q", p.ID, got.Model)
		}
		for name, id := range p.Dependencies {
			if name == model.TokenizerFile || name == p.OnnxFile {
				continue
			}
			placed, err := os.ReadFile(filepath.Join(filepath.Dir(got.Model), strings.TrimPrefix(name, model.ModelFolder+"/")))
			if err != nil || string(placed) != id {
				t.Errorf("%s: %s is not next to the graph: %q, %v", p.ID, name, placed, err)
			}
		}
	}
}

func TestLocateFetchesAnotherRepositoryByAddress(t *testing.T) {
	_, local := newCuratedConfig(t)
	local = local.WithRepository("owner/repo")
	local.Revision = "0123456789012345678901234567890123456789"
	t.Setenv("HF_ENDPOINT", "")
	var addresses []string
	fetch := Fetch{
		Dependency: func(_ context.Context, id string) (string, error) {
			t.Fatalf("owner/repo requested the dependency %s", id)
			return "", nil
		},
		File: func(_ context.Context, address, _ string) (string, error) {
			addresses = append(addresses, address)
			return "", errors.New("offline")
		},
	}
	if _, err := Locate(t.Context(), local, fetch); err == nil {
		t.Fatal("Locate of an unreachable repository succeeded")
	}
	if len(addresses) == 0 || !strings.HasPrefix(addresses[0], "https://huggingface.co/api/models/owner/repo/") {
		t.Errorf("addresses = %v, want the index of owner/repo", addresses)
	}
}

func newCuratedConfig(t *testing.T) (model.ModelInfo, model.LocalModel) {
	t.Helper()
	table, err := model.GetShippedProfiles()
	if err != nil {
		t.Fatal(err)
	}
	return model.NewConfig(table.GetCurated()[0])
}

func TestLocateFromDir(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, model.ModelFile)
	tokenPath := filepath.Join(dir, model.TokenizerFile)
	if err := os.WriteFile(modelPath, []byte("weights"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Locate(t.Context(), model.LocalModel{Dir: dir}, Fetch{}); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Locate() with missing tokenizer = %v, want fs.ErrNotExist", err)
	}
	if err := os.WriteFile(tokenPath, []byte("tokens"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := Locate(t.Context(), model.LocalModel{Dir: dir}, Fetch{})
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
