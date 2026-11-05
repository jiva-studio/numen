package service_test

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

func TestSearchToolsListHasOnlyNoteSearch(t *testing.T) {
	_, svc := setupTestVault()
	registry, err := svc.Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	infos, err := registry.ListToolInfos(t.Context())
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	names := map[string]bool{}
	for _, info := range infos {
		names[info.GetName()] = true
	}
	if names["search_query"] {
		t.Error("the tool search_query is listed")
	}
	if !names["note_search"] {
		t.Error("the tool note_search is not listed")
	}
}

func TestCallOfSearchQueryIsAnUnknownTool(t *testing.T) {
	_, svc := setupTestVault()
	registry, err := svc.Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	res, err := registry.InvokeTool(t.Context(), tools.Call{
		VaultID:   testVault,
		Name:      "search_query",
		Arguments: []byte(`{"query":"thermodynamic"}`),
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if res.GetError() != tools.ErrorCodeUnknownTool {
		t.Errorf("error code = %v, want unknown tool", res.GetError())
	}
}

func TestNoteSearchAnswersWithPathAndSnippetOnly(t *testing.T) {
	_, svc := setupTestVault()
	registry, err := svc.Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	res, err := registry.InvokeTool(t.Context(), tools.Call{
		VaultID:   testVault,
		Name:      "note_search",
		Arguments: []byte(`{"query":"thermodynamic"}`),
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	var out struct {
		Matches []map[string]json.RawMessage `json:"matches"`
	}
	if err := json.Unmarshal(res.GetResult().GetStructured(), &out); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(out.Matches) == 0 {
		t.Fatal("no matches")
	}
	for _, match := range out.Matches {
		for _, kept := range []string{"path", "snippet"} {
			if _, ok := match[kept]; !ok {
				t.Errorf("match has no %q: %v", kept, match)
			}
		}
		for _, removed := range []string{"source", "text"} {
			if _, ok := match[removed]; ok {
				t.Errorf("match has the field %q: %v", removed, match)
			}
		}
	}
}

func TestNoteWriteTakesNoBodyArgument(t *testing.T) {
	_, svc := setupTestVault()
	registry, err := svc.Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	infos, err := registry.ListToolInfos(t.Context())
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	for _, info := range infos {
		if info.GetName() != "note_write" {
			continue
		}
		var schema struct {
			Properties map[string]json.RawMessage `json:"properties"`
		}
		if err := json.Unmarshal(info.GetInputSchema(), &schema); err != nil {
			t.Fatalf("unmarshal schema: %v", err)
		}
		if _, ok := schema.Properties["body"]; ok {
			t.Error("the input of note_write names body")
		}
		if _, ok := schema.Properties["content"]; !ok {
			t.Error("the input of note_write does not name content")
		}
	}
}

func TestNoteWriteWithBodyInsteadOfContentStoresNoText(t *testing.T) {
	vault, svc := setupTestVault()
	registry, err := svc.Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	res, err := registry.InvokeTool(t.Context(), tools.Call{
		VaultID:   testVault,
		Name:      "note_write",
		Arguments: []byte(`{"path":"Orbit.md","body":"# Orbit\nOrbits curve."}`),
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if content, ok := vault.Content("Orbit.md"); ok && strings.Contains(content, "Orbits curve.") {
		t.Errorf("the text of the argument body was stored: %q", content)
	}
	if !res.GetResult().GetIsRefusal() {
		t.Error("a call with body and no content was not refused")
	}
}

func getGoneToolNames() []string {
	return []string{"search_query", "note_neighbourhood", "vault_info"}
}

func TestToolsListHasOneNameForEachTool(t *testing.T) {
	_, svc := setupTestVault()
	registry, err := svc.Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	infos, err := registry.ListToolInfos(t.Context())
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	names := map[string]bool{}
	for _, info := range infos {
		names[info.GetName()] = true
	}
	for _, kept := range []string{"note_search", "link_list", "vault_get"} {
		if !names[kept] {
			t.Errorf("the tool %s is not listed", kept)
		}
	}
	for _, gone := range getGoneToolNames() {
		if names[gone] {
			t.Errorf("the tool %s is listed", gone)
		}
	}
}

func TestCallOfAGoneToolNameIsAnUnknownTool(t *testing.T) {
	_, svc := setupTestVault()
	registry, err := svc.Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	for _, gone := range getGoneToolNames() {
		res, err := registry.InvokeTool(t.Context(), tools.Call{
			VaultID:   testVault,
			Name:      gone,
			Arguments: []byte(`{"query":"thermodynamic","path":"Entropy.md"}`),
		})
		if err != nil {
			t.Fatalf("call %s: %v", gone, err)
		}
		if res.GetError() != tools.ErrorCodeUnknownTool {
			t.Errorf("%s: error code = %v, want unknown tool", gone, res.GetError())
		}
	}
}

func TestDocsNameEachToolOnce(t *testing.T) {
	doc, err := os.ReadFile(filepath.Join(getRepoRoot(t), "docs", "agents.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(doc), "second name") {
		t.Error("docs/agents.md says tools have a second name")
	}
}

func TestNoSourceNamesAGoneTool(t *testing.T) {
	root := getRepoRoot(t)
	for _, top := range []string{"plugins", "docs"} {
		err := filepath.WalkDir(filepath.Join(root, top), func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				switch entry.Name() {
				case "node_modules", ".git", "dist", "build":
					return filepath.SkipDir
				}
				return nil
			}
			if isTestFile(entry.Name()) {
				return nil
			}
			content, err := os.ReadFile(path)
			if err != nil || !utf8.Valid(content) {
				return nil //nolint:nilerr // an unreadable or binary file names no tool
			}
			for _, gone := range getGoneToolNames() {
				if bytes.Contains(content, []byte(gone)) {
					rel := strings.TrimPrefix(path, root)
					t.Errorf("%s names the gone tool %s", rel, gone)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func isTestFile(name string) bool {
	for _, mark := range []string{"_test.go", ".test.", ".spec."} {
		if strings.Contains(name, mark) {
			return true
		}
	}
	return false
}

func getRepoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	return root
}
