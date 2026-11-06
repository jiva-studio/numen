package service_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/service"
)

// geologyVault is the identity of the invented geology vault the write tests call.
const geologyVault = "geology-vault"

type movedRecord struct {
	From     string   `json:"from"`
	To       string   `json:"to"`
	Repaired []string `json:"repaired"`
	Dangling []string `json:"dangling"`
}

type noteRenameOutput struct {
	Path  string       `json:"path"`
	Title string       `json:"title"`
	By    string       `json:"by"`
	Moved *movedRecord `json:"moved"`
}

type noteMoveItem struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Error string `json:"error"`
}

type noteMoveOutput struct {
	Moved []noteMoveItem `json:"moved"`
}

type noteRemoveItem struct {
	Path     string   `json:"path"`
	Trashed  string   `json:"trashed"`
	Dangling []string `json:"dangling"`
	Error    string   `json:"error"`
}

type noteRemoveOutput struct {
	Removed []noteRemoveItem `json:"removed"`
}

// newGeologyVault is an invented vault of rocks and minerals. Minerals.md links to Basalt by name.
func newGeologyVault(t *testing.T) (*hosttest.Vault, *tools.Registry) {
	t.Helper()
	v := hosttest.New(geologyVault)
	v.File("Rocks/Basalt.md", "---\ntitle: Basalt\n---\nA volcanic rock.\n")
	v.File("Rocks/Granite.md", "---\ntitle: Granite\n---\nA plutonic rock.\n")
	v.File("Minerals.md", "---\ntitle: Minerals\n---\nQuartz and feldspar, found in [[Basalt]].\n")
	v.File("Fossils.md", "Shells kept in limestone.\n")
	registry, err := service.New(v).WithVaultList(v).Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	return v, registry
}

// requireContent fails the test unless the file of the vault holds exactly the content.
func requireContent(t *testing.T, v *hosttest.Vault, filePath, want string) {
	t.Helper()
	got, ok := v.Content(filePath)
	if !ok {
		t.Errorf("%s is not in the vault", filePath)
		return
	}
	if got != want {
		t.Errorf("%s holds %q, want %q", filePath, got, want)
	}
}

// requireAbsent fails the test when a file is at the path.
func requireAbsent(t *testing.T, v *hosttest.Vault, filePath string) {
	t.Helper()
	if got, ok := v.Content(filePath); ok {
		t.Errorf("%s is still in the vault: %q", filePath, got)
	}
}

// makePathsArguments is the arguments {"paths": [...] , extra...} of n invented paths.
func makePathsArguments(t *testing.T, n int, extra map[string]any) string {
	t.Helper()
	paths := make([]string, n)
	for i := range paths {
		paths[i] = fmt.Sprintf("Rocks/Pebble %d.md", i)
	}
	arguments := map[string]any{"paths": paths}
	for k, value := range extra {
		arguments[k] = value
	}
	raw, err := json.Marshal(arguments)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestNoteRenameWithATitleKeyWritesTheTitleAndMovesTheFile(t *testing.T) {
	v, registry := newGeologyVault(t)

	var out noteRenameOutput
	decodeInto(t, callTool(t, registry, geologyVault, "note_rename",
		`{"path":"Rocks/Basalt.md","title":"Flood Basalt"}`), &out)
	if out.Path != "Rocks/Flood Basalt.md" || out.Title != "Flood Basalt" || out.By != "frontmatter" {
		t.Errorf("result = %+v, want Rocks/Flood Basalt.md, Flood Basalt, by frontmatter", out)
	}
	if out.Moved == nil || out.Moved.From != "Rocks/Basalt.md" || out.Moved.To != "Rocks/Flood Basalt.md" {
		t.Errorf("moved = %+v, want from Rocks/Basalt.md to Rocks/Flood Basalt.md", out.Moved)
	}
	content, ok := v.Content("Rocks/Flood Basalt.md")
	if !ok || !strings.Contains(content, "title: Flood Basalt") {
		t.Errorf("Rocks/Flood Basalt.md holds %q, %v, want the new title key", content, ok)
	}
	requireAbsent(t, v, "Rocks/Basalt.md")
}

func TestNoteRenameWithoutATitleKeyRenamesByFilename(t *testing.T) {
	v, registry := newGeologyVault(t)

	var out noteRenameOutput
	decodeInto(t, callTool(t, registry, geologyVault, "note_rename",
		`{"path":"Fossils.md","title":"Ammonites"}`), &out)
	if out.Path != "Ammonites.md" || out.Title != "Ammonites" || out.By != "filename" {
		t.Errorf("result = %+v, want Ammonites.md, Ammonites, by filename", out)
	}
	if out.Moved == nil || out.Moved.From != "Fossils.md" || out.Moved.To != "Ammonites.md" {
		t.Errorf("moved = %+v, want from Fossils.md to Ammonites.md", out.Moved)
	}
	requireContent(t, v, "Ammonites.md", "Shells kept in limestone.\n")
	requireAbsent(t, v, "Fossils.md")
}

func TestNoteRenameOntoAnExistingNoteIsRefused(t *testing.T) {
	v, registry := newGeologyVault(t)
	before, _ := v.Content("Rocks/Basalt.md")

	requireRefusal(t, registry, geologyVault, "note_rename", `{"path":"Rocks/Basalt.md","title":"Granite"}`)
	requireContent(t, v, "Rocks/Basalt.md", before)
	requireContent(t, v, "Rocks/Granite.md", "---\ntitle: Granite\n---\nA plutonic rock.\n")
}

// getRefusalText calls the tool, fails the test unless the tool is known and refuses the call, and
// returns the text of the refusal.
func getRefusalText(t *testing.T, registry *tools.Registry, vaultID, name, arguments string) string {
	t.Helper()
	res, err := registry.InvokeTool(t.Context(), tools.Call{VaultID: vaultID, Name: name, Arguments: []byte(arguments)})
	if err != nil {
		t.Fatalf("call %s: %v", name, err)
	}
	if res.GetError() == tools.ErrorCodeUnknownTool {
		t.Fatalf("call %s: the tool is unknown", name)
	}
	if !res.GetResult().GetIsRefusal() {
		t.Fatalf("call %s %s was not refused: %s", name, arguments, res.GetResult().GetStructured())
	}
	return res.GetResult().GetText()
}

func TestNoteRenameOntoATakenFilenameNamesTheUnchangedNote(t *testing.T) {
	_, registry := newGeologyVault(t)

	text := getRefusalText(t, registry, geologyVault, "note_rename", `{"path":"Rocks/Basalt.md","title":"Granite"}`)
	for _, part := range []string{"Rocks/Basalt.md", "Rocks/Granite.md"} {
		if !strings.Contains(text, part) {
			t.Errorf("the refusal %q does not name %s", text, part)
		}
	}
	lower := strings.ToLower(text)
	if !strings.Contains(lower, "not changed") && !strings.Contains(lower, "unchanged") {
		t.Errorf("the refusal %q does not say the note was left unchanged", text)
	}
}

func TestNoteRenameOfAMissingNoteIsRefused(t *testing.T) {
	_, registry := newGeologyVault(t)
	requireRefusal(t, registry, geologyVault, "note_rename", `{"path":"Rocks/Obsidian.md","title":"Glass"}`)
}

func TestNoteMoveMovesEachPathInOrderAndReportsTheOneThatFails(t *testing.T) {
	v, registry := newGeologyVault(t)

	var out noteMoveOutput
	decodeInto(t, callTool(t, registry, geologyVault, "note_move",
		`{"paths":["Rocks/Basalt.md","Rocks/Obsidian.md","Minerals.md"],"folder":"Archive"}`), &out)
	if len(out.Moved) != 3 {
		t.Fatalf("got %d moves, want 3: %+v", len(out.Moved), out.Moved)
	}
	if got := out.Moved[0]; got.From != "Rocks/Basalt.md" || got.To != "Archive/Basalt.md" || got.Error != "" {
		t.Errorf("first = %+v, want Rocks/Basalt.md to Archive/Basalt.md", got)
	}
	if got := out.Moved[1]; got.From != "Rocks/Obsidian.md" || got.To != "" || got.Error == "" {
		t.Errorf("second = %+v, want Rocks/Obsidian.md with an error and no target", got)
	}
	if got := out.Moved[2]; got.From != "Minerals.md" || got.To != "Archive/Minerals.md" || got.Error != "" {
		t.Errorf("third = %+v, want Minerals.md to Archive/Minerals.md", got)
	}
	requireContent(t, v, "Archive/Basalt.md", "---\ntitle: Basalt\n---\nA volcanic rock.\n")
	requireAbsent(t, v, "Rocks/Basalt.md")
	requireAbsent(t, v, "Minerals.md")
}

func TestNoteMoveOntoAnExistingFileReportsAnError(t *testing.T) {
	v, registry := newGeologyVault(t)
	v.File("Archive/Granite.md", "An older granite note.\n")

	var out noteMoveOutput
	decodeInto(t, callTool(t, registry, geologyVault, "note_move",
		`{"paths":["Rocks/Granite.md"],"folder":"Archive"}`), &out)
	if len(out.Moved) != 1 || out.Moved[0].From != "Rocks/Granite.md" || out.Moved[0].Error == "" {
		t.Errorf("moved = %+v, want one error for Rocks/Granite.md", out.Moved)
	}
	requireContent(t, v, "Archive/Granite.md", "An older granite note.\n")
	requireContent(t, v, "Rocks/Granite.md", "---\ntitle: Granite\n---\nA plutonic rock.\n")
}

func TestNoteMoveWithAnEmptyFolderMovesToTheVaultRoot(t *testing.T) {
	v, registry := newGeologyVault(t)

	var out noteMoveOutput
	decodeInto(t, callTool(t, registry, geologyVault, "note_move", `{"paths":["Rocks/Granite.md"],"folder":""}`), &out)
	if len(out.Moved) != 1 || out.Moved[0].To != "Granite.md" || out.Moved[0].Error != "" {
		t.Errorf("moved = %+v, want Rocks/Granite.md to Granite.md", out.Moved)
	}
	requireContent(t, v, "Granite.md", "---\ntitle: Granite\n---\nA plutonic rock.\n")
}

func TestNoteMoveOfMoreThanFiftyPathsIsRefused(t *testing.T) {
	_, registry := newGeologyVault(t)
	requireRefusal(t, registry, geologyVault, "note_move", makePathsArguments(t, 51, map[string]any{"folder": "Archive"}))

	var out noteMoveOutput
	decodeInto(t, callTool(t, registry, geologyVault, "note_move",
		makePathsArguments(t, 50, map[string]any{"folder": "Archive"})), &out)
	if len(out.Moved) != 50 {
		t.Errorf("got %d moves, want 50", len(out.Moved))
	}
}

func TestNoteRemoveMovesTheNoteToTheTrashAndReportsDanglingLinks(t *testing.T) {
	v, registry := newGeologyVault(t)

	var out noteRemoveOutput
	decodeInto(t, callTool(t, registry, geologyVault, "note_remove", `{"paths":["Rocks/Basalt.md"]}`), &out)
	if len(out.Removed) != 1 {
		t.Fatalf("got %d removals, want 1: %+v", len(out.Removed), out.Removed)
	}
	got := out.Removed[0]
	if got.Path != "Rocks/Basalt.md" || got.Error != "" {
		t.Errorf("removal = %+v, want Rocks/Basalt.md without an error", got)
	}
	if !strings.HasPrefix(got.Trashed, ".trash/") {
		t.Errorf("trashed = %q, want a path under .trash/", got.Trashed)
	}
	if len(got.Dangling) != 1 || got.Dangling[0] != "Minerals.md" {
		t.Errorf("dangling = %v, want [Minerals.md]", got.Dangling)
	}
	requireContent(t, v, got.Trashed, "---\ntitle: Basalt\n---\nA volcanic rock.\n")
	requireAbsent(t, v, "Rocks/Basalt.md")
}

func TestNoteRemoveReportsAFolderAndAMissingPathAndRemovesTheRest(t *testing.T) {
	v, registry := newGeologyVault(t)

	var out noteRemoveOutput
	decodeInto(t, callTool(t, registry, geologyVault, "note_remove",
		`{"paths":["Rocks","Fossils.md","Rocks/Obsidian.md"]}`), &out)
	if len(out.Removed) != 3 {
		t.Fatalf("got %d removals, want 3: %+v", len(out.Removed), out.Removed)
	}
	if got := out.Removed[0]; got.Path != "Rocks" || got.Error == "" || got.Trashed != "" {
		t.Errorf("first = %+v, want the folder Rocks with an error", got)
	}
	if got := out.Removed[1]; got.Path != "Fossils.md" || got.Error != "" || !strings.HasPrefix(got.Trashed, ".trash/") {
		t.Errorf("second = %+v, want Fossils.md trashed", got)
	}
	if got := out.Removed[2]; got.Path != "Rocks/Obsidian.md" || got.Error == "" {
		t.Errorf("third = %+v, want Rocks/Obsidian.md with an error", got)
	}
	requireContent(t, v, "Rocks/Basalt.md", "---\ntitle: Basalt\n---\nA volcanic rock.\n")
	requireContent(t, v, "Rocks/Granite.md", "---\ntitle: Granite\n---\nA plutonic rock.\n")
	requireAbsent(t, v, "Fossils.md")
}

func TestNoteRemoveOfMoreThanFiftyPathsIsRefused(t *testing.T) {
	_, registry := newGeologyVault(t)
	requireRefusal(t, registry, geologyVault, "note_remove", makePathsArguments(t, 51, nil))

	var out noteRemoveOutput
	decodeInto(t, callTool(t, registry, geologyVault, "note_remove", makePathsArguments(t, 50, nil)), &out)
	if len(out.Removed) != 50 {
		t.Errorf("got %d removals, want 50", len(out.Removed))
	}
}
