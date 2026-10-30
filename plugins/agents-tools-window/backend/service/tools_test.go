package service

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

func TestRegisterTools(t *testing.T) {
	mgr := NewManager()
	registry := tools.New()
	if err := RegisterTools(registry, mgr, nil); err != nil {
		t.Fatalf("RegisterTools: %v", err)
	}

	listed, err := registry.ListToolInfos(t.Context())
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	names := make([]string, 0, len(listed))
	for _, tool := range listed {
		names = append(names, tool.GetName())
	}

	for _, expected := range []string{"window_tabs_get", "window_focus_tab", "edit_preview"} {
		if !slices.Contains(names, expected) {
			t.Errorf("ListTools missing %q, got %v", expected, names)
		}
	}
}

func TestWindowTabsGetEmpty(t *testing.T) {
	mgr := NewManager()
	registry := tools.New()
	if err := RegisterTools(registry, mgr, nil); err != nil {
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
	if result.GetIsRefusal() {
		t.Fatalf("unexpected refusal: %s", result.GetText())
	}

	var output WindowTabsGetOutput
	if err := json.Unmarshal(result.GetStructured(), &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if len(output.Tabs) != 0 {
		t.Errorf("len(Tabs) = %d, want 0", len(output.Tabs))
	}
	if output.Summary != "the window has nothing open" {
		t.Errorf("summary = %q, want 'the window has nothing open'", output.Summary)
	}
}

func TestWindowTabsGetWithTabs(t *testing.T) {
	mgr := NewManager()
	mgr.SetOpenTabs([]TabInfo{
		{ID: "tab-1", Kind: "note", Path: "notes/daily.md", Title: "daily"},
		{ID: "tab-2", Kind: "document", Path: "docs/spec.pdf", Title: "Specification", Where: "page 3 of 10"},
	}, "tab-2")

	registry := tools.New()
	if err := RegisterTools(registry, mgr, nil); err != nil {
		t.Fatalf("RegisterTools: %v", err)
	}

	resp, err := registry.InvokeTool(t.Context(), tools.Call{
		Name:      "window_tabs_get",
		Arguments: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}

	var output WindowTabsGetOutput
	if err := json.Unmarshal(resp.GetResult().GetStructured(), &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if len(output.Tabs) != 2 {
		t.Fatalf("len(Tabs) = %d, want 2", len(output.Tabs))
	}
	if output.ActiveID != "tab-2" {
		t.Errorf("ActiveID = %q, want tab-2", output.ActiveID)
	}
	if !strings.Contains(output.Summary, "Specification") || !strings.Contains(output.Summary, "page 3 of 10") {
		t.Errorf("Summary = %q, want to mention Specification and page 3 of 10", output.Summary)
	}
}

func TestWindowTabsGet_UsesContextFocus(t *testing.T) {
	mgr := NewManager()
	registry := tools.New()
	if err := RegisterTools(registry, mgr, nil); err != nil {
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
	if !strings.Contains(output.Summary, "inferred") {
		t.Errorf("Summary = %q, want to mention inferred", output.Summary)
	}
}

func TestWindowFocusTab(t *testing.T) {
	mgr := NewManager()
	mgr.SetOpenTabs([]TabInfo{
		{ID: "t1", Kind: "note", Path: "notes/a.md", Title: "A"},
		{ID: "t2", Kind: "note", Path: "notes/b.md", Title: "B"},
	}, "t1")

	registry := tools.New()
	if err := RegisterTools(registry, mgr, nil); err != nil {
		t.Fatalf("RegisterTools: %v", err)
	}

	// Focus existing tab by ID
	args := marshalJSON(t, WindowFocusTabInput{TabID: "t2"})
	resp, err := registry.InvokeTool(t.Context(), tools.Call{
		Name:      "window_focus_tab",
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	var output WindowFocusTabOutput
	if err := json.Unmarshal(resp.GetResult().GetStructured(), &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if !output.Success || output.Tab.ID != "t2" {
		t.Errorf("output = %+v, want Success true and Tab.ID t2", output)
	}

	// Focus new note by Path
	args = marshalJSON(t, WindowFocusTabInput{Path: "notes/c.md"})
	resp, err = registry.InvokeTool(t.Context(), tools.Call{
		Name:      "window_focus_tab",
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if err := json.Unmarshal(resp.GetResult().GetStructured(), &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if !output.Success || output.Tab.Path != "notes/c.md" {
		t.Errorf("output = %+v, want Success true and Tab.Path notes/c.md", output)
	}

	// Focus with empty arguments produces refusal
	args = marshalJSON(t, WindowFocusTabInput{})
	resp, err = registry.InvokeTool(t.Context(), tools.Call{
		Name:      "window_focus_tab",
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !resp.GetResult().GetIsRefusal() {
		t.Errorf("expected refusal for empty focus arguments")
	}
}

func TestEditPreviewFromVault(t *testing.T) {
	mgr := NewManager()
	vault := hosttest.New("test-vault")
	ctx := t.Context()
	if err := vault.Create(ctx, "notes/sample.md", []byte("hello world\nline 2\n")); err != nil {
		t.Fatalf("create vault file: %v", err)
	}

	registry := tools.New()
	if err := RegisterTools(registry, mgr, vault); err != nil {
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
	if resp.GetResult().GetIsRefusal() {
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
	mgr := NewManager()
	registry := tools.New()
	if err := RegisterTools(registry, mgr, nil); err != nil {
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
	if output.Additions != 1 || output.Deletions != 1 {
		t.Errorf("got +%d, -%d; want +1, -1", output.Additions, output.Deletions)
	}
}

func TestEditPreviewValidation(t *testing.T) {
	mgr := NewManager()
	registry := tools.New()
	if err := RegisterTools(registry, mgr, nil); err != nil {
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
	if !resp.GetResult().GetIsRefusal() {
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
	if !resp.GetResult().GetIsRefusal() {
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
