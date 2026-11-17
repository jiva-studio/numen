package service

import (
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	numenv1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm/llmtest"
)

func newTestProvider(t *testing.T) connection.Provider {
	t.Helper()
	t.Setenv("NUMEN_PLUGIN_DATA_DIR", t.TempDir())
	r, err := openRerankingUseCase()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := r.Close(); err != nil {
			t.Error(err)
		}
	})
	return newProvider(r)
}

func TestProviderServesRerankModelsAndDetected(t *testing.T) {
	provider := newTestProvider(t)

	var names []string
	for name := range provider {
		names = append(names, name)
	}
	slices.Sort(names)

	if want := []string{connection.OperationDetected, llm.OperationModels, llm.OperationRerank}; !slices.Equal(names, want) {
		t.Errorf("operations = %v, want %v", names, want)
	}
}

func TestRerankOfflineWithoutTheModelIsTryNext(t *testing.T) {
	models := t.TempDir()
	t.Setenv("NUMEN_MODELS_DIR", models)
	t.Setenv("NUMEN_OFFLINE", "1")
	provider := newTestProvider(t)

	_, err := llmtest.Rerank(t.Context(), provider, connection.Call{ConnectionID: "local-reranker:Local reranker", Type: "local-reranker"},
		&numenv1.RerankRequest{Query: "when does the ferry leave", Documents: []string{"The ferry leaves the pier at seven.", "Basil likes warm soil."}})

	if _, isTryNext := connection.IsTryNext(err); !isTryNext {
		t.Fatalf("error = %v, want a try-next failure", err)
	}
	if !strings.HasSuffix(err.Error(), connection.ReasonUnavailable.String()) {
		t.Errorf("error = %q, want the reason unavailable", err)
	}
	var written []string
	walkErr := filepath.WalkDir(models, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			written = append(written, path)
		}
		return nil
	})
	if walkErr != nil {
		t.Fatal(walkErr)
	}
	if len(written) != 0 {
		t.Errorf("an offline call wrote %v", written)
	}
}
