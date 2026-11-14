package service_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm/llmtest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/internal/adapter/reranker"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/service"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/usecase/preview"
)

const testVault = "harbor-vault"

func newRegistry(t *testing.T) *tools.Registry {
	t.Helper()
	vault := hosttest.New(testVault)
	vault.File("Notes/Tides.md", "---\ntitle: Tide Tables\n---\nThe tide at the quay.")
	search := hosttest.NewSearch(testVault)
	search.SetModel(hosttest.SearchModel{Name: "harbor-model", Recipe: "harbor-recipe", FloorHint: 0.4})
	search.SetEmbedder(func(text string) []float32 {
		if strings.Contains(strings.ToLower(text), "tide") {
			return []float32{1, 0}
		}
		return []float32{0, 1}
	})
	search.With(testVault).SetIndexedResults([]host.SearchResult{
		{ChunkID: "c1", Path: "Notes/Tides.md", Text: "The tide at the quay.", Start: 0, Length: 21, MatchLine: 2},
		{ChunkID: "c2", Path: "Notes/Keeper.md", Text: "The lighthouse keeper.", Start: 0, Length: 22},
	})
	caller := llmtest.NewCaller(llm.Provide(nil, llm.Roles{}), connection.Call{})
	open := func(id domain.VaultID) preview.Dependencies {
		return preview.Dependencies{Search: search.With(id), Notes: vault.With(id), Reranker: reranker.New(caller)}
	}
	registry, err := service.NewRegistry(open, preview.DefaultConfig)
	if err != nil {
		t.Fatal(err)
	}
	return registry
}

func call(t *testing.T, registry *tools.Registry, vault, arguments string) (structured map[string]any, failure string) {
	t.Helper()
	res, err := registry.InvokeTool(t.Context(), tools.Call{VaultID: vault, Name: "saved_search_preview", Arguments: []byte(arguments)})
	if err != nil {
		t.Fatal(err)
	}
	if res.GetResult().GetIsError() {
		return nil, res.GetResult().GetText()
	}
	if err := json.Unmarshal(res.GetResult().GetStructured(), &structured); err != nil {
		t.Fatalf("%v in %s", err, res.GetResult().GetStructured())
	}
	return structured, ""
}

func TestRegistryListsSavedSearchPreviewAsReadOnlyWithIndexRead(t *testing.T) {
	listing, err := newRegistry(t).ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if len(listing.Tools) != 1 {
		t.Fatalf("tools = %+v, want saved_search_preview alone", listing.Tools)
	}
	tool := listing.Tools[0]
	if tool.Name != "saved_search_preview" || tool.Effect != tools.ReadOnly || !slices.Equal(tool.Requires, []string{"index:read"}) {
		t.Errorf("tool = %+v, want saved_search_preview, read only, requiring index:read", tool)
	}
	truncated := "A passage is truncated, often mid-sentence: read more with library_read before you conclude that a source does not cover a topic."
	if !slices.Contains(tool.Instructions, truncated) || tool.Description == "" {
		t.Errorf("instructions = %q, want the sentence about truncated passages", tool.Instructions)
	}
}

func TestPreviewToolReturnsTheCompileErrorWithItsPosition(t *testing.T) {
	out, failure := call(t, newRegistry(t), testVault, `{"query":"meaning(\"tides\") &&"}`)
	if failure != "" {
		t.Fatalf("the tool failed with %q, want the problems in its output", failure)
	}

	problems, isList := out["errors"].([]any)
	if !isList || len(problems) == 0 {
		t.Fatalf("output = %v, want a list under errors", out)
	}
	problem, isObject := problems[0].(map[string]any)
	if !isObject || problem["line"] != float64(1) || problem["column"] != float64(20) || problem["message"] == "" {
		t.Errorf("problem = %v, want line 1, column 20 and a message", problems[0])
	}
}

func TestPreviewToolReturnsResultsWithTheirKeys(t *testing.T) {
	out, failure := call(t, newRegistry(t), testVault, `{"query":"meaning(\"tides\")","limit":5}`)
	if failure != "" {
		t.Fatal(failure)
	}

	results, isList := out["results"].([]any)
	if !isList || len(results) != 1 || out["order"] != "fused" || out["query"] != `meaning("tides")` || out["floor"] != 0.4 {
		t.Fatalf("output = %v, want one result in fused order with the floor", out)
	}
	if _, hasCount := out["candidateCount"]; !hasCount {
		t.Errorf("output = %v, want candidateCount", out)
	}
	result := results[0].(map[string]any)
	for key, want := range map[string]any{
		"path": "Notes/Tides.md", "title": "Tide Tables", "text": "The tide at the quay.", "start": float64(0), "length": float64(21),
		"line": float64(3), "uri": "numen:Notes%2FTides.md?start=0&length=21", "score": float64(1), "chunkCount": float64(1),
	} {
		if result[key] != want {
			t.Errorf("result[%q] = %v, want %v", key, result[key], want)
		}
	}
}

func TestPreviewToolFailsForACallWithoutAVault(t *testing.T) {
	_, failure := call(t, newRegistry(t), "", `{"query":"meaning(\"tides\")"}`)

	if !strings.Contains(failure, "the call names no vault") {
		t.Errorf("failure = %q, want the call to name no vault", failure)
	}
}

func TestPreviewToolRefusesANegativeLimit(t *testing.T) {
	_, failure := call(t, newRegistry(t), testVault, `{"query":"meaning(\"tides\")","limit":-1}`)

	if failure == "" {
		t.Error("the tool accepted the limit -1")
	}
}
