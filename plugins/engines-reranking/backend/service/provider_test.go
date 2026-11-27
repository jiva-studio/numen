package service

import (
	"context"
	"errors"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm/llmtest"
	"github.com/jiva-studio/numen/libs/shared/rerank"
	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/domain/profile"
	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/internal/adapter/console"
	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/internal/adapter/engine"
	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/internal/adapter/profilefiles"
	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/ports"
	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/usecase/reranking"
)

func newTestProvider(t *testing.T) connection.Provider {
	t.Helper()
	t.Setenv("NUMEN_PLUGIN_DATA_DIR", t.TempDir())
	r, err := openRerankingUseCase(hosttest.NewQueue().EnqueueWork, engine.Fetch{
		Dependency: func(context.Context, string) (string, error) { return "", errors.New("model unavailable offline") },
		File: func(context.Context, string, string) (string, error) {
			return "", errors.New("model unavailable offline")
		},
	})
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
		&rerank.RerankRequest{Query: "when does the ferry leave", Documents: []string{"The ferry leaves the pier at seven.", "Basil likes warm soil."}})

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

type recordingScorer struct {
	queries []string
}

func (s *recordingScorer) ScorePairs(_ context.Context, query string, documents []string) ([]float32, error) {
	s.queries = append(s.queries, query)
	scores := make([]float32, len(documents))
	for i, document := range documents {
		scores[i] = float32(len(document))
	}
	return scores, nil
}

func (s *recordingScorer) Close() error { return nil }

func TestRerankScoresTheDocumentsAgainstTheQueryWithoutTheInstruction(t *testing.T) {
	t.Setenv("NUMEN_PLUGIN_DATA_DIR", t.TempDir())
	scorer := &recordingScorer{}
	open := func(context.Context, profile.Profile) (ports.Scorer, error) { return scorer, nil }
	use, err := reranking.New(profilefiles.GetDefaultDirectory(), open, hosttest.NewQueue().EnqueueWork, console.Report)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := use.Close(); err != nil {
			t.Error(err)
		}
	})

	result, err := llmtest.Rerank(t.Context(), newProvider(use), connection.Call{ConnectionID: "local-reranker:Local reranker", Type: "local-reranker"},
		&rerank.RerankRequest{Query: "when does the ferry leave", Instruction: "Find the notes that answer the question", Documents: []string{"ferry", "basil likes"}})

	if err != nil {
		t.Fatal(err)
	}
	if want := []float32{5, 11}; !slices.Equal(result.GetScores(), want) {
		t.Errorf("scores = %v, want %v", result.GetScores(), want)
	}
	if want := []string{"when does the ferry leave"}; !slices.Equal(scorer.queries, want) {
		t.Errorf("the model scored the queries %q, want %q", scorer.queries, want)
	}
}
