package service_test

import (
	"encoding/json"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/service"
)

func setupTestVault() (*hosttest.Vault, *service.Tools) {
	v := hosttest.New("physics-vault")
	v.File("Entropy.md", "---\nid: 01H7X1234567890ABCDEFGHJK\ntitle: Entropy Concept\n---\n# Entropy\nEntropy is thermodynamic disorder.\n## Second Law\nEntropy increases.\nSee also [[Heat]].")
	v.File("Heat.md", "# Heat\nHeat flows from hot to cold.\nReferences [[Entropy]].")
	v.File("data.txt", "line 1\nline 2\nline 3\n")
	return v, service.New(v)
}

func TestNoteRead(t *testing.T) {
	ctx := t.Context()
	_, svc := setupTestVault()
	registry, err := svc.Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}

	callReq := tools.Call{
		VaultID:   testVault,
		Name:      "note_read",
		Arguments: []byte(`{"path":"Entropy.md"}`),
	}
	res, err := registry.InvokeTool(ctx, callReq)
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if res.GetError() != tools.ErrorCodeUnspecified {
		t.Fatalf("unexpected tool error code: %v", res.GetError())
	}
	if res.GetResult().GetIsRefusal() {
		t.Fatalf("tool refused: %s", res.GetResult().GetText())
	}

	var out service.NoteReadOutput
	if err := json.Unmarshal(res.GetResult().GetStructured(), &out); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if out.Title != "Entropy Concept" {
		t.Errorf("title = %q, want 'Entropy Concept'", out.Title)
	}
	if out.Path != "Entropy.md" {
		t.Errorf("path = %q, want 'Entropy.md'", out.Path)
	}
	if out.Fingerprint == "" {
		t.Error("expected non-empty fingerprint")
	}
	if out.Frontmatter == nil || out.Frontmatter["title"] != "Entropy Concept" {
		t.Errorf("expected frontmatter with title, got %v", out.Frontmatter)
	}

	// Test note resolution without .md
	callReq2 := tools.Call{
		VaultID:   testVault,
		Name:      "note_read",
		Arguments: []byte(`{"path":"Heat"}`),
	}
	res2, err := registry.InvokeTool(ctx, callReq2)
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if res2.GetResult().GetIsRefusal() {
		t.Fatalf("tool refused for Heat: %s", res2.GetResult().GetText())
	}
	var out2 service.NoteReadOutput
	if err := json.Unmarshal(res2.GetResult().GetStructured(), &out2); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if out2.Path != "Heat.md" {
		t.Errorf("resolved path = %q, want 'Heat.md'", out2.Path)
	}
}

func TestNoteWrite(t *testing.T) {
	ctx := t.Context()
	_, svc := setupTestVault()
	registry, err := svc.Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}

	// 1. Create a new note
	callReq := tools.Call{
		VaultID:   testVault,
		Name:      "note_write",
		Arguments: []byte(`{"path":"Gravity.md","content":"# Gravity\nGravity attracts."}`),
	}
	res, err := registry.InvokeTool(ctx, callReq)
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if res.GetResult().GetIsRefusal() {
		t.Fatalf("tool refused: %s", res.GetResult().GetText())
	}
	var writeOut service.NoteWriteOutput
	if err := json.Unmarshal(res.GetResult().GetStructured(), &writeOut); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if writeOut.Fingerprint == "" {
		t.Fatal("expected non-empty fingerprint")
	}

	// 2. Atomic update with valid file stat
	callReq2 := tools.Call{
		VaultID: testVault,
		Name:    "note_write",
		Arguments: []byte(`{"path":"Gravity.md","content":"# Gravity\nGravity bends spacetime.","fingerprint":"` +
			writeOut.Fingerprint + `"}`),
	}
	res2, err := registry.InvokeTool(ctx, callReq2)
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if res2.GetResult().GetIsRefusal() {
		t.Fatalf("tool refused: %s", res2.GetResult().GetText())
	}

	// 3. Stale update with old file stat fails
	callReq3 := tools.Call{
		VaultID: testVault,
		Name:    "note_write",
		Arguments: []byte(`{"path":"Gravity.md","content":"# Gravity\nConflict.","fingerprint":"` +
			writeOut.Fingerprint + `"}`),
	}
	res3, err := registry.InvokeTool(ctx, callReq3)
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if !res3.GetResult().GetIsRefusal() {
		t.Error("expected refusal on stale fingerprint write")
	}
}

func TestNoteHeadings(t *testing.T) {
	ctx := t.Context()
	_, svc := setupTestVault()
	registry, err := svc.Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}

	callReq := tools.Call{
		VaultID:   testVault,
		Name:      "note_headings",
		Arguments: []byte(`{"path":"Entropy.md"}`),
	}
	res, err := registry.InvokeTool(ctx, callReq)
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if res.GetResult().GetIsRefusal() {
		t.Fatalf("tool refused: %s", res.GetResult().GetText())
	}

	var out service.NoteHeadingsOutput
	if err := json.Unmarshal(res.GetResult().GetStructured(), &out); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if len(out.Headings) != 2 {
		t.Fatalf("headings count = %d, want 2", len(out.Headings))
	}
	if out.Headings[0].Level != 1 || out.Headings[0].Text != "Entropy" {
		t.Errorf("heading 0 = %+v", out.Headings[0])
	}
	if out.Headings[1].Level != 2 || out.Headings[1].Text != "Second Law" {
		t.Errorf("heading 1 = %+v", out.Headings[1])
	}
}

func testSearchQuery(t *testing.T, registry *tools.Registry, toolName string) {
	t.Helper()
	callReq := tools.Call{
		VaultID:   testVault,
		Name:      toolName,
		Arguments: []byte(`{"query":"thermodynamic"}`),
	}
	res, err := registry.InvokeTool(t.Context(), callReq)
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if res.GetResult().GetIsRefusal() {
		t.Fatalf("tool refused: %s", res.GetResult().GetText())
	}

	var out service.NoteSearchOutput
	if err := json.Unmarshal(res.GetResult().GetStructured(), &out); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if out.Total < 1 {
		t.Fatalf("expected at least 1 match, got %d", out.Total)
	}
	found := false
	for _, m := range out.Matches {
		if m.Path == "Entropy.md" {
			found = true
			if m.Snippet == "" {
				t.Error("expected non-empty snippet")
			}
			break
		}
	}
	if !found {
		t.Error("Entropy.md not found in search matches")
	}
}

func TestNoteSearch(t *testing.T) {
	_, svc := setupTestVault()
	registry, err := svc.Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}

	for _, toolName := range []string{"note_search", "search_query"} {
		t.Run(toolName, func(t *testing.T) {
			testSearchQuery(t, registry, toolName)
		})
	}
}

func testNeighbourhoodQuery(t *testing.T, registry *tools.Registry, toolName string) {
	t.Helper()
	callReq := tools.Call{
		VaultID:   testVault,
		Name:      toolName,
		Arguments: []byte(`{"path":"Entropy.md"}`),
	}
	res, err := registry.InvokeTool(t.Context(), callReq)
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if res.GetResult().GetIsRefusal() {
		t.Fatalf("tool refused: %s", res.GetResult().GetText())
	}

	var out service.LinkListOutput
	if err := json.Unmarshal(res.GetResult().GetStructured(), &out); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if len(out.Links) == 0 {
		t.Errorf("expected outgoing links for Entropy.md, got none")
	}
	if len(out.Backlinks) == 0 {
		t.Errorf("expected backlinks pointing to Entropy.md, got none")
	}
}

func TestNoteNeighbourhood(t *testing.T) {
	_, svc := setupTestVault()
	registry, err := svc.Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}

	for _, toolName := range []string{"note_neighbourhood", "link_list"} {
		t.Run(toolName, func(t *testing.T) {
			testNeighbourhoodQuery(t, registry, toolName)
		})
	}
}

func testVaultInfoQuery(t *testing.T, registry *tools.Registry, toolName string) {
	t.Helper()
	callReq := tools.Call{
		VaultID:   testVault,
		Name:      toolName,
		Arguments: []byte(`{}`),
	}
	res, err := registry.InvokeTool(t.Context(), callReq)
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if res.GetResult().GetIsRefusal() {
		t.Fatalf("tool refused: %s", res.GetResult().GetText())
	}

	var out service.VaultInfoOutput
	if err := json.Unmarshal(res.GetResult().GetStructured(), &out); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if out.Name != "physics-vault" {
		t.Errorf("vault name = %q, want 'physics-vault'", out.Name)
	}
	if out.NoteCount < 2 {
		t.Errorf("vault note count = %d, expected at least 2", out.NoteCount)
	}
}

func TestVaultInfo(t *testing.T) {
	_, svc := setupTestVault()
	registry, err := svc.Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}

	for _, toolName := range []string{"vault_info", "vault_get"} {
		t.Run(toolName, func(t *testing.T) {
			testVaultInfoQuery(t, registry, toolName)
		})
	}
}

func TestFileReadAndCreate(t *testing.T) {
	ctx := t.Context()
	_, svc := setupTestVault()
	registry, err := svc.Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}

	// 1. file_read
	readReq := tools.Call{
		VaultID:   testVault,
		Name:      "file_read",
		Arguments: []byte(`{"path":"data.txt","start":0,"length":6}`),
	}
	res, err := registry.InvokeTool(ctx, readReq)
	if err != nil {
		t.Fatalf("call file_read: %v", err)
	}
	if res.GetResult().GetIsRefusal() {
		t.Fatalf("file_read refused: %s", res.GetResult().GetText())
	}
	var readOut service.FileReadOutput
	if err := json.Unmarshal(res.GetResult().GetStructured(), &readOut); err != nil {
		t.Fatalf("unmarshal file_read: %v", err)
	}
	if readOut.Content != "line 1" {
		t.Errorf("file_read content = %q, want 'line 1'", readOut.Content)
	}
	if readOut.Length != 6 {
		t.Errorf("file_read length = %d, want 6", readOut.Length)
	}

	// 2. file_create
	createReq := tools.Call{
		VaultID:   testVault,
		Name:      "file_create",
		Arguments: []byte(`{"path":"new_data.txt","content":"hello world"}`),
	}
	createRes, err := registry.InvokeTool(ctx, createReq)
	if err != nil {
		t.Fatalf("call file_create: %v", err)
	}
	if createRes.GetResult().GetIsRefusal() {
		t.Fatalf("file_create refused: %s", createRes.GetResult().GetText())
	}

	// 3. creating already existing file should fail
	createAgainRes, err := registry.InvokeTool(ctx, createReq)
	if err != nil {
		t.Fatalf("call file_create duplicate: %v", err)
	}
	if !createAgainRes.GetResult().GetIsRefusal() {
		t.Error("expected refusal creating file that already exists")
	}
}

func TestListTools(t *testing.T) {
	ctx := t.Context()
	_, svc := setupTestVault()
	registry, err := svc.Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}

	listed, err := registry.ListToolInfos(ctx)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}

	expectedTools := map[string]bool{
		"note_read":          false,
		"note_write":         false,
		"note_headings":      false,
		"note_search":        false,
		"search_query":       false,
		"note_neighbourhood": false,
		"link_list":          false,
		"vault_info":         false,
		"vault_get":          false,
		"file_read":          false,
		"file_create":        false,
	}

	for _, tool := range listed {
		if _, ok := expectedTools[tool.GetName()]; ok {
			expectedTools[tool.GetName()] = true
		}
	}

	for name, found := range expectedTools {
		if !found {
			t.Errorf("expected tool %q not found in ListTools", name)
		}
	}
}

// testVault is the identity of the vault setupTestVault makes, which every call names.
const testVault = "physics-vault"

// A call works the vault it names, though the host shows another, and a call that names none is
// refused.
func TestACallWorksTheVaultItNames(t *testing.T) {
	shown, svc := setupTestVault()
	garden := hosttest.New("garden-vault")
	garden.File("Beds.md", "# Beds\nRaised beds of oak.")
	shown.Beside(garden)
	registry, err := svc.Registry()
	if err != nil {
		t.Fatal(err)
	}

	res, err := registry.InvokeTool(t.Context(), tools.Call{
		VaultID: "garden-vault", Name: "note_read", Arguments: []byte(`{"path":"Beds.md"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	var out service.NoteReadOutput
	if err := json.Unmarshal(res.GetResult().GetStructured(), &out); err != nil || out.Path != "Beds.md" {
		t.Errorf("the call read %s (%v)", res.GetResult().GetText(), err)
	}

	res, err = registry.InvokeTool(t.Context(), tools.Call{
		Name: "note_read", Arguments: []byte(`{"path":"Entropy.md"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.GetResult().GetIsRefusal() || res.GetResult().GetText() != "the call names no vault" {
		t.Errorf("a call naming no vault answered %v", res.GetResult())
	}
}
