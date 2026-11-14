package service

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-tools-window/backend/api"
)

func TestRegisterTools(t *testing.T) {
	registry := tools.New()
	if err := RegisterTools(registry, Host{}); err != nil {
		t.Fatalf("RegisterTools: %v", err)
	}

	listed, err := registry.ListTools(t.Context())
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	names := make([]string, 0, len(listed.Tools))
	for _, tool := range listed.Tools {
		names = append(names, tool.GetName())
	}

	slices.Sort(names)
	if want := []string{"note_edit_preview", "note_focus", "source_focus", "window_tab_activate", "window_tab_list"}; !slices.Equal(names, want) {
		t.Errorf("ListTools returns %v, want %v", names, want)
	}

	// Registering duplicate tools returns an error.
	if err := RegisterTools(registry, Host{}); err == nil {
		t.Errorf("expected error when registering duplicate tools")
	}
}

func TestWindowTabListReturnsNoTabsForAnEmptyWindow(t *testing.T) {
	registry := tools.New()
	if err := RegisterTools(registry, Host{}); err != nil {
		t.Fatalf("RegisterTools: %v", err)
	}

	resp, err := registry.InvokeTool(t.Context(), tools.Call{
		Name:      "window_tab_list",
		Arguments: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	result := resp.GetResult()
	if result.GetIsError() {
		t.Fatalf("unexpected refusal: %s", result.GetText())
	}

	var output api.WindowTabListOutput
	if err := json.Unmarshal(result.GetStructured(), &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if len(output.Tabs) != 0 {
		t.Errorf("len(Tabs) = %d, want 0", len(output.Tabs))
	}
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(result.GetStructured(), &keys); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if _, hasSummary := keys["summary"]; hasSummary {
		t.Errorf("the result has a summary: %s", result.GetStructured())
	}
}

func TestWindowTabListUsesTheFocusOfTheCall(t *testing.T) {
	registry := tools.New()
	if err := RegisterTools(registry, Host{}); err != nil {
		t.Fatalf("RegisterTools: %v", err)
	}

	resp, err := registry.InvokeTool(t.Context(), tools.Call{
		Name:      "window_tab_list",
		Arguments: []byte(`{}`),
		Context:   map[string]string{"focus": "notes/inferred.md"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}

	var output api.WindowTabListOutput
	if err := json.Unmarshal(resp.GetResult().GetStructured(), &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if len(output.Tabs) != 1 {
		t.Fatalf("len(Tabs) = %d, want 1", len(output.Tabs))
	}
	if output.Tabs[0].Path != "notes/inferred.md" {
		t.Errorf("Tabs[0].Path = %q, want notes/inferred.md", output.Tabs[0].Path)
	}
	var keys struct {
		ActiveTabID string `json:"activeTabId"`
		Tabs        []struct {
			TabID string `json:"tabId"`
		} `json:"tabs"`
	}
	if err := json.Unmarshal(resp.GetResult().GetStructured(), &keys); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if keys.ActiveTabID != "front-tab" || len(keys.Tabs) != 1 || keys.Tabs[0].TabID != "front-tab" {
		t.Errorf("the result is %s, want activeTabId front-tab and the tab front-tab", resp.GetResult().GetStructured())
	}
	if output.Tabs[0].Title != "inferred" {
		t.Errorf("Tabs[0].Title = %q, want inferred", output.Tabs[0].Title)
	}
}

func TestNoteEditPreviewReadsTheNoteFromTheVault(t *testing.T) {
	vault := hosttest.New("test-vault")
	ctx := t.Context()
	if err := vault.Create(ctx, "notes/sample.md", []byte("hello world\nline 2\n")); err != nil {
		t.Fatalf("create vault file: %v", err)
	}

	registry := tools.New()
	if err := RegisterTools(registry, Host{Files: filesOf(vault)}); err != nil {
		t.Fatalf("RegisterTools: %v", err)
	}

	input := api.NoteEditPreviewInput{
		Path:         "notes/sample.md",
		ProposedText: "hello world\nline 2 modified\nnew line\n",
	}
	args := marshalJSON(t, input)
	resp, err := registry.InvokeTool(ctx, tools.Call{
		Name:      "note_edit_preview",
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if resp.GetResult().GetIsError() {
		t.Fatalf("refusal: %s", resp.GetResult().GetText())
	}

	var output api.NoteEditPreviewOutput
	if err := json.Unmarshal(resp.GetResult().GetStructured(), &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if output.Path != "notes/sample.md" {
		t.Errorf("output.Path = %q, want notes/sample.md", output.Path)
	}
	if output.Additions != 2 || output.Deletions != 1 {
		t.Errorf("got +%d, -%d; want +2, -1 (diff: %s)", output.Additions, output.Deletions, output.Diff)
	}
}

func TestNoteEditPreviewAcceptsADiff(t *testing.T) {
	registry := tools.New()
	if err := RegisterTools(registry, Host{}); err != nil {
		t.Fatalf("RegisterTools: %v", err)
	}

	customDiff := "--- a/notes/custom.md\n+++ b/notes/custom.md\n@@ -1 +1 @@\n-old\n+new\n"
	args := marshalJSON(t, api.NoteEditPreviewInput{
		Path: "notes/custom.md",
		Diff: customDiff,
	})
	resp, err := registry.InvokeTool(t.Context(), tools.Call{
		Name:      "note_edit_preview",
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}

	var output api.NoteEditPreviewOutput
	if err := json.Unmarshal(resp.GetResult().GetStructured(), &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if output.Diff != customDiff {
		t.Errorf("output.Diff = %q, want %q", output.Diff, customDiff)
	}
	if output.Additions != 1 || output.Deletions != 1 {
		t.Errorf("got +%d, -%d; want +1, -1", output.Additions, output.Deletions)
	}

	// Explicit OriginalText takes precedence over vault content, and works with nil vault.
	vault := hosttest.New("test-vault")
	if err := vault.Create(t.Context(), "notes/override.md", []byte("vault line\n")); err != nil {
		t.Fatalf("create vault file: %v", err)
	}
	registryWithVault := tools.New()
	if err := RegisterTools(registryWithVault, Host{Files: filesOf(vault)}); err != nil {
		t.Fatalf("RegisterTools: %v", err)
	}
	respOverride, err := registryWithVault.InvokeTool(t.Context(), tools.Call{
		Name: "note_edit_preview",
		Arguments: marshalJSON(t, api.NoteEditPreviewInput{
			Path:         "notes/override.md",
			OriginalText: "explicit original\n",
			ProposedText: "explicit proposed\n",
		}),
	})
	if err != nil {
		t.Fatalf("CallTool override: %v", err)
	}
	if err := json.Unmarshal(respOverride.GetResult().GetStructured(), &output); err != nil {
		t.Fatalf("unmarshal override output: %v", err)
	}
	if !strings.Contains(output.Diff, "-explicit original") {
		t.Errorf("expected diff against OriginalText, got %s", output.Diff)
	}

	// Nil vault with empty OriginalText diffs against empty string.
	respNilVault, err := registry.InvokeTool(t.Context(), tools.Call{
		Name: "note_edit_preview",
		Arguments: marshalJSON(t, api.NoteEditPreviewInput{
			Path:         "notes/new.md",
			ProposedText: "brand new\n",
		}),
	})
	if err != nil {
		t.Fatalf("CallTool nil vault: %v", err)
	}
	if err := json.Unmarshal(respNilVault.GetResult().GetStructured(), &output); err != nil {
		t.Fatalf("unmarshal nil vault output: %v", err)
	}
	if output.Additions != 1 || output.Deletions != 0 {
		t.Errorf("nil vault got +%d, -%d; want +1, -0", output.Additions, output.Deletions)
	}
}

func TestNoteEditPreviewRejectsMissingArguments(t *testing.T) {
	registry := tools.New()
	if err := RegisterTools(registry, Host{}); err != nil {
		t.Fatalf("RegisterTools: %v", err)
	}

	// Missing path
	args := marshalJSON(t, api.NoteEditPreviewInput{ProposedText: "abc"})
	resp, err := registry.InvokeTool(t.Context(), tools.Call{
		Name:      "note_edit_preview",
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !resp.GetResult().GetIsError() {
		t.Errorf("expected refusal for missing path")
	}

	// Missing content and diff
	args = marshalJSON(t, api.NoteEditPreviewInput{Path: "notes/a.md"})
	resp, err = registry.InvokeTool(t.Context(), tools.Call{
		Name:      "note_edit_preview",
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !resp.GetResult().GetIsError() {
		t.Errorf("expected refusal for missing content and diff")
	}
}

func marshalJSON(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func filesOf(vault *hosttest.Vault) func(domain.VaultID) host.FileReader {
	return func(id domain.VaultID) host.FileReader { return vault.With(id) }
}
