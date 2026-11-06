package service

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

func TestRegisterTools(t *testing.T) {
	mgr := NewManager(clock.NewFake(time.Unix(1_700_000_000, 0)))
	registry := tools.New()
	if err := RegisterTools(registry, mgr, Host{}); err != nil {
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

	for _, expected := range []string{"window_tabs_get", "window_focus_tab", "edit_preview"} {
		if !slices.Contains(names, expected) {
			t.Errorf("ListTools missing %q, got %v", expected, names)
		}
	}

	// Registering duplicate tools returns an error.
	if err := RegisterTools(registry, mgr, Host{}); err == nil {
		t.Errorf("expected error when registering duplicate tools")
	}
}

func TestWindowTabsGetEmpty(t *testing.T) {
	mgr := NewManager(clock.NewFake(time.Unix(1_700_000_000, 0)))
	registry := tools.New()
	if err := RegisterTools(registry, mgr, Host{}); err != nil {
		t.Fatalf("RegisterTools: %v", err)
	}

	resp, err := registry.InvokeTool(t.Context(), tools.Call{
		Name:      "window_tabs_get",
		Arguments: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	result := resp.GetResult()
	if result.GetIsError() {
		t.Fatalf("unexpected refusal: %s", result.GetText())
	}

	var output WindowTabsGetOutput
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
		t.Errorf("the answer has a summary: %s", result.GetStructured())
	}
}

func TestWindowTabsGet_UsesContextFocus(t *testing.T) {
	mgr := NewManager(clock.NewFake(time.Unix(1_700_000_000, 0)))
	registry := tools.New()
	if err := RegisterTools(registry, mgr, Host{}); err != nil {
		t.Fatalf("RegisterTools: %v", err)
	}

	resp, err := registry.InvokeTool(t.Context(), tools.Call{
		Name:      "window_tabs_get",
		Arguments: []byte(`{}`),
		Context:   map[string]string{"focus": "notes/inferred.md"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}

	var output WindowTabsGetOutput
	if err := json.Unmarshal(resp.GetResult().GetStructured(), &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if len(output.Tabs) != 1 {
		t.Fatalf("len(Tabs) = %d, want 1", len(output.Tabs))
	}
	if output.Tabs[0].Path != "notes/inferred.md" {
		t.Errorf("Tabs[0].Path = %q, want notes/inferred.md", output.Tabs[0].Path)
	}
	if output.ActiveID != "front-tab" {
		t.Errorf("ActiveID = %q, want front-tab", output.ActiveID)
	}
	if output.Tabs[0].Title != "inferred" {
		t.Errorf("Tabs[0].Title = %q, want inferred", output.Tabs[0].Title)
	}
}

func TestEditPreviewFromVault(t *testing.T) {
	mgr := NewManager(clock.NewFake(time.Unix(1_700_000_000, 0)))
	vault := hosttest.New("test-vault")
	ctx := t.Context()
	if err := vault.Create(ctx, "notes/sample.md", []byte("hello world\nline 2\n")); err != nil {
		t.Fatalf("create vault file: %v", err)
	}

	registry := tools.New()
	if err := RegisterTools(registry, mgr, Host{Vault: vault}); err != nil {
		t.Fatalf("RegisterTools: %v", err)
	}

	input := EditPreviewInput{
		Path:            "notes/sample.md",
		ProposedContent: "hello world\nline 2 modified\nnew line\n",
	}
	args := marshalJSON(t, input)
	resp, err := registry.InvokeTool(ctx, tools.Call{
		Name:      "edit_preview",
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if resp.GetResult().GetIsError() {
		t.Fatalf("refusal: %s", resp.GetResult().GetText())
	}

	var output EditPreviewOutput
	if err := json.Unmarshal(resp.GetResult().GetStructured(), &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if output.Path != "notes/sample.md" {
		t.Errorf("output.Path = %q, want notes/sample.md", output.Path)
	}
	if output.Additions != 2 || output.Deletions != 1 {
		t.Errorf("got +%d, -%d; want +2, -1 (diff: %s)", output.Additions, output.Deletions, output.Diff)
	}
	if output.PreviewID == "" {
		t.Error("expected non-empty preview ID")
	}

	saved, ok := mgr.GetPreview(output.PreviewID)
	if !ok || saved.Path != "notes/sample.md" {
		t.Errorf("preview not found in manager: %+v", saved)
	}
}

func TestEditPreviewWithDirectDiff(t *testing.T) {
	mgr := NewManager(clock.NewFake(time.Unix(1_700_000_000, 0)))
	registry := tools.New()
	if err := RegisterTools(registry, mgr, Host{}); err != nil {
		t.Fatalf("RegisterTools: %v", err)
	}

	customDiff := "--- a/notes/custom.md\n+++ b/notes/custom.md\n@@ -1 +1 @@\n-old\n+new\n"
	args := marshalJSON(t, EditPreviewInput{
		Path: "notes/custom.md",
		Diff: customDiff,
	})
	resp, err := registry.InvokeTool(t.Context(), tools.Call{
		Name:      "edit_preview",
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}

	var output EditPreviewOutput
	if err := json.Unmarshal(resp.GetResult().GetStructured(), &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if output.Diff != customDiff {
		t.Errorf("output.Diff = %q, want %q", output.Diff, customDiff)
	}
	if output.Additions != 1 || output.Deletions != 1 {
		t.Errorf("got +%d, -%d; want +1, -1", output.Additions, output.Deletions)
	}

	// Explicit OriginalContent takes precedence over vault content, and works with nil vault.
	vault := hosttest.New("test-vault")
	if err := vault.Create(t.Context(), "notes/override.md", []byte("vault line\n")); err != nil {
		t.Fatalf("create vault file: %v", err)
	}
	registryWithVault := tools.New()
	if err := RegisterTools(registryWithVault, mgr, Host{Vault: vault}); err != nil {
		t.Fatalf("RegisterTools: %v", err)
	}
	respOverride, err := registryWithVault.InvokeTool(t.Context(), tools.Call{
		Name: "edit_preview",
		Arguments: marshalJSON(t, EditPreviewInput{
			Path:            "notes/override.md",
			OriginalContent: "explicit original\n",
			ProposedContent: "explicit proposed\n",
		}),
	})
	if err != nil {
		t.Fatalf("CallTool override: %v", err)
	}
	if err := json.Unmarshal(respOverride.GetResult().GetStructured(), &output); err != nil {
		t.Fatalf("unmarshal override output: %v", err)
	}
	if !strings.Contains(output.Diff, "-explicit original") {
		t.Errorf("expected diff against OriginalContent, got %s", output.Diff)
	}

	// Nil vault with empty OriginalContent diffs against empty string.
	respNilVault, err := registry.InvokeTool(t.Context(), tools.Call{
		Name: "edit_preview",
		Arguments: marshalJSON(t, EditPreviewInput{
			Path:            "notes/new.md",
			ProposedContent: "brand new\n",
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

func TestEditPreviewValidation(t *testing.T) {
	mgr := NewManager(clock.NewFake(time.Unix(1_700_000_000, 0)))
	registry := tools.New()
	if err := RegisterTools(registry, mgr, Host{}); err != nil {
		t.Fatalf("RegisterTools: %v", err)
	}

	// Missing path
	args := marshalJSON(t, EditPreviewInput{ProposedContent: "abc"})
	resp, err := registry.InvokeTool(t.Context(), tools.Call{
		Name:      "edit_preview",
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !resp.GetResult().GetIsError() {
		t.Errorf("expected refusal for missing path")
	}

	// Missing content and diff
	args = marshalJSON(t, EditPreviewInput{Path: "notes/a.md"})
	resp, err = registry.InvokeTool(t.Context(), tools.Call{
		Name:      "edit_preview",
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !resp.GetResult().GetIsError() {
		t.Errorf("expected refusal for missing content and diff")
	}
}

// marshalJSON encodes value as JSON and fails the test when encoding fails.
func marshalJSON(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}
