package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/embedding"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/rpcerror"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/engine"
)

// recording is a model that keeps the texts it was given.
type recording struct {
	engine.MockEmbedder
	mu    *sync.Mutex
	texts *[]string
}

func (r recording) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	r.mu.Lock()
	*r.texts = append(*r.texts, texts...)
	r.mu.Unlock()
	return r.MockEmbedder.Embed(ctx, texts)
}

func newRecordingServer(t *testing.T) (*embeddingServer, *[]string) {
	t.Helper()
	srv := newEmbeddingServer()
	srv.profilesDir = t.TempDir()
	var mu sync.Mutex
	var received []string
	srv.open = func(context.Context, engine.ModelInfo, engine.LocalModel) (textEmbedder, error) {
		return recording{mu: &mu, texts: &received}, nil
	}
	t.Cleanup(func() { closeServer(t, srv) })
	return srv, &received
}

func TestALocalModelReadsDocumentsAndQueriesWithTheirOwnPrefix(t *testing.T) {
	srv, texts := newRecordingServer(t)
	srv.updateConfig(map[string]string{"provider": "local", "model": engine.DefaultModel}, nil)

	if _, err := srv.EmbedTexts(t.Context(), embedding.RoleDocument, []string{"a note", "another"}); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.EmbedTexts(t.Context(), embedding.RoleQuery, []string{"a question"}); err != nil {
		t.Fatal(err)
	}
	want := []string{"passage: a note", "passage: another", "query: a question"}
	if !slices.Equal(*texts, want) {
		t.Errorf("the model read %q, want %q", *texts, want)
	}
}

func TestAModelWithAQueryPrefixNeverRunsAQueryWithoutIt(t *testing.T) {
	profiles, err := engine.GetShippedProfiles()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range profiles.GetCurated() {
		srv, texts := newRecordingServer(t)
		srv.updateConfig(map[string]string{"provider": "local", "model": p.ID}, nil)
		if _, err := srv.EmbedTexts(t.Context(), embedding.RoleQuery, []string{"question"}); err != nil {
			t.Fatalf("%s: %v", p.ID, err)
		}
		if got := (*texts)[0]; got != p.QueryPrefix+"question" {
			t.Errorf("%s: a query ran as %q, want it behind %q", p.ID, got, p.QueryPrefix)
		}
	}
}

func TestAnUnsetRoleIsADocument(t *testing.T) {
	srv, texts := newRecordingServer(t)
	srv.updateConfig(map[string]string{"provider": "local", "model": engine.DefaultModel}, nil)
	var role embedding.Role
	if _, err := srv.EmbedTexts(t.Context(), role, []string{"text"}); err != nil {
		t.Fatal(err)
	}
	if (*texts)[0] != "passage: text" {
		t.Errorf("the model read %q", (*texts)[0])
	}
}

func TestARemoteProviderGetsNoPrefix(t *testing.T) {
	var body []byte
	stand := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		if body, err = io.ReadAll(r.Body); err != nil {
			t.Error(err)
		}
		encodeJSON(t, w, map[string]any{"data": []map[string]any{{"embedding": []float32{0.5, 0.25}, "index": 0}}})
	}))
	defer stand.Close()

	for _, provider := range []string{"openrouter", "openai", "custom"} {
		srv := newEmbeddingServer()
		srv.profilesDir = t.TempDir()
		srv.updateConfig(map[string]string{
			"provider": provider, "model": "vendor/model", "baseUrl": stand.URL,
		}, keysOf("key"))
		info, err := srv.GetModelInfo(t.Context())
		if err != nil {
			t.Fatalf("%s: %v", provider, err)
		}
		if info.GetHandling() != "" {
			t.Errorf("%s: handling %q, want none", provider, info.GetHandling())
		}
	}

	srv := newEmbeddingServer()
	srv.updateConfig(map[string]string{"provider": "custom", "model": "vendor/model", "baseUrl": stand.URL}, keysOf("key"))
	for _, role := range []embedding.Role{embedding.RoleQuery, embedding.RoleDocument} {
		if _, err := srv.EmbedTexts(t.Context(), role, []string{"a question"}); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(body), `"input":["a question"]`) {
			t.Errorf("the service was sent %s for role %v, want the text as it is", body, role)
		}
	}
}

func TestTheRecipeHandlingChangesWithTheProfileAndNotWithMinChunkLength(t *testing.T) {
	srv := newEmbeddingServer()
	dir := t.TempDir()
	srv.profilesDir = dir
	handling := func(values map[string]string) string {
		t.Helper()
		values["provider"], values["model"] = "local", engine.DefaultModel
		srv.updateConfig(values, nil)
		info, err := srv.GetModelInfo(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		return info.GetHandling()
	}

	base := handling(map[string]string{})
	if len(base) != 8 {
		t.Fatalf("handling %q is not a short hash", base)
	}
	if got := handling(map[string]string{"minChunkLength": "500"}); got != base {
		t.Errorf("minChunkLength changed the handling: %s", got)
	}
	writeFile(t, filepath.Join(dir, "e5.json"), `{"id":"Xenova/multilingual-e5-small","queryPrefix":"q: "}`)
	if got := handling(map[string]string{}); got == base {
		t.Error("a new query prefix left the handling as it was")
	}
	writeFile(t, filepath.Join(dir, "e5.json"), `{"id":"Xenova/multilingual-e5-small","onnxFile":"onnx/model_int8.onnx"}`)
	if got := handling(map[string]string{}); got == base {
		t.Error("another model file left the handling as it was")
	}
	if err := os.Remove(filepath.Join(dir, "e5.json")); err != nil {
		t.Fatal(err)
	}
	if got := handling(map[string]string{}); got != base {
		t.Errorf("without the override the handling is %s, want %s", got, base)
	}
}

func TestAChangeOfTheProfileDropsTheLoadedModel(t *testing.T) {
	srv := newEmbeddingServer()
	dir := t.TempDir()
	srv.profilesDir = dir
	openCount := 0
	srv.open = func(context.Context, engine.ModelInfo, engine.LocalModel) (textEmbedder, error) {
		openCount++
		return engine.MockEmbedder{}, nil
	}
	t.Cleanup(func() { closeServer(t, srv) })
	srv.updateConfig(map[string]string{"provider": "local", "model": engine.DefaultModel}, nil)
	embedText(t.Context(), t, srv)
	srv.updateConfig(map[string]string{"provider": "local", "model": engine.DefaultModel}, nil)
	embedText(t.Context(), t, srv)
	if openCount != 1 {
		t.Fatalf("an unchanged profile opened the model %d times", openCount)
	}
	writeFile(t, filepath.Join(dir, "e5.json"), `{"id":"Xenova/multilingual-e5-small","documentPrefix":"doc: "}`)
	srv.updateConfig(map[string]string{"provider": "local", "model": engine.DefaultModel}, nil)
	embedText(t.Context(), t, srv)
	if openCount != 2 {
		t.Errorf("a changed profile opened the model %d times in all, want 2", openCount)
	}
}

func TestAUserProfileAddsALocalModelWithoutRecompiling(t *testing.T) {
	srv := newEmbeddingServer()
	dir := t.TempDir()
	srv.profilesDir = dir
	writeFile(t, filepath.Join(dir, "mine.json"), `{"id":"owner/mine","dims":96,"maxTokens":300,"pooling":"last","onnxFile":"onnx/model.onnx","queryPrefix":"Q: "}`)
	writeFile(t, filepath.Join(dir, "broken.json"), `{"id":`)
	srv.updateConfig(map[string]string{"provider": "local", "model": "owner/mine"}, nil)

	info, err := srv.GetModelInfo(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if info.GetName() != "owner/mine" || info.GetDimensions() != 96 || info.GetMaxTokens() != 300 || info.GetPooling() != "last" {
		t.Errorf("model = %+v", info)
	}
}

func TestTheModelOfAUserProfileThatWasRemovedFailsWithAClearError(t *testing.T) {
	srv := newEmbeddingServer()
	dir := t.TempDir()
	srv.profilesDir = dir
	srv.updateConfig(map[string]string{"provider": "local", "model": "owner/gone"}, nil)
	_, err := srv.EmbedTexts(t.Context(), embedding.RoleDocument, []string{"text"})
	if rpcerror.GetCode(err) != rpcerror.FailedPrecondition || !strings.Contains(err.Error(), dir) {
		t.Errorf("error = %v, want FailedPrecondition that names %s", err, dir)
	}
}

func TestTheEnvironmentOverridesTheChosenProfile(t *testing.T) {
	t.Setenv("NUMEN_EMBEDDING_DIMENSIONS", "100")
	t.Setenv("NUMEN_EMBEDDING_MAX_TOKENS", "128")
	t.Setenv("NUMEN_EMBEDDING_POOLING", "last")
	t.Setenv("NUMEN_EMBEDDING_FILE", "model_other.onnx")
	srv := newEmbeddingServer()
	srv.updateConfig(map[string]string{"provider": "local", "model": engine.DefaultModel}, nil)
	info, err := srv.GetModelInfo(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if info.GetDimensions() != 100 || info.GetMaxTokens() != 128 || info.GetPooling() != "last" || srv.localModel.File != "model_other.onnx" {
		t.Errorf("info %+v, file %q", info, srv.localModel.File)
	}
	if info.GetHandling() == "" {
		t.Error("the handling is gone")
	}
}

func TestTheMockModelRecordsWhatItIsGiven(t *testing.T) {
	log := filepath.Join(t.TempDir(), "embed.log")
	t.Setenv(envMockML, "true")
	t.Setenv(envMockLog, log)
	srv := newEmbeddingServer()
	t.Cleanup(func() { closeServer(t, srv) })

	if _, err := srv.EmbedTexts(t.Context(), embedding.RoleDocument, []string{"a note"}); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.EmbedTexts(t.Context(), embedding.RoleQuery, []string{"a question"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 || !strings.Contains(lines[0], `"passage: a note"`) || !strings.Contains(lines[1], `"query: a question"`) {
		t.Errorf("log = %q", lines)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
