package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/service"
)

// geologyVault is the identity of the invented geology vault the write tests call.
const geologyVault = "geology-vault"

type moveRecord struct {
	From          string   `json:"from"`
	Path          string   `json:"path"`
	RepairedNotes []string `json:"repairedNotes"`
	BrokenLinks   []string `json:"brokenLinks"`
}

type noteRenameOutput struct {
	Path       string      `json:"path"`
	Title      string      `json:"title"`
	NameSource string      `json:"nameSource"`
	Move       *moveRecord `json:"move"`
}

type noteMoveItem struct {
	From  string `json:"from"`
	Path  string `json:"path"`
	Error string `json:"error"`
}

type noteMoveOutput struct {
	Results []noteMoveItem `json:"results"`
}

type noteRemoveItem struct {
	Path        string   `json:"path"`
	TrashPath   string   `json:"trashPath"`
	BrokenLinks []string `json:"brokenLinks"`
	Error       string   `json:"error"`
}

type noteRemoveOutput struct {
	Results []noteRemoveItem `json:"results"`
}

func newGeologyVault(t *testing.T) (*hosttest.Vault, *tools.Registry) {
	t.Helper()
	v := hosttest.New(geologyVault)
	v.File("Rocks/Basalt.md", "---\ntitle: Basalt\n---\nA volcanic rock.\n")
	v.File("Rocks/Granite.md", "---\ntitle: Granite\n---\nA plutonic rock.\n")
	v.File("Minerals.md", "---\ntitle: Minerals\n---\nQuartz and feldspar, found in [[Basalt]].\n")
	v.File("Fossils.md", "Shells kept in limestone.\n")
	registry, err := service.New(vaultsOf(v)).WithVaultList(v).Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	return v, registry
}

func requireContent(t *testing.T, v *hosttest.Vault, filePath, want string) {
	t.Helper()
	got, ok := v.Content(filePath)
	if !ok {
		t.Errorf("%s is not in the vault", filePath)
		return
	}
	if got != want {
		t.Errorf("%s contains %q, want %q", filePath, got, want)
	}
}

func requireAbsent(t *testing.T, v *hosttest.Vault, filePath string) {
	t.Helper()
	if got, ok := v.Content(filePath); ok {
		t.Errorf("%s is still in the vault: %q", filePath, got)
	}
}

func makePathsArguments(t *testing.T, n int, extra map[string]any) string {
	t.Helper()
	paths := make([]string, n)
	for i := range paths {
		paths[i] = fmt.Sprintf("Rocks/Pebble %d.md", i)
	}
	arguments := map[string]any{"paths": paths}
	maps.Copy(arguments, extra)
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
	if out.Path != "Rocks/Flood Basalt.md" || out.Title != "Flood Basalt" || out.NameSource != "frontmatter" {
		t.Errorf("result = %+v, want Rocks/Flood Basalt.md, Flood Basalt, nameSource frontmatter", out)
	}
	if out.Move == nil || out.Move.From != "Rocks/Basalt.md" || out.Move.Path != "Rocks/Flood Basalt.md" {
		t.Errorf("move = %+v, want from Rocks/Basalt.md to Rocks/Flood Basalt.md", out.Move)
	}
	content, ok := v.Content("Rocks/Flood Basalt.md")
	if !ok || !strings.Contains(content, "title: Flood Basalt") {
		t.Errorf("Rocks/Flood Basalt.md contains %q, %v, want the new title key", content, ok)
	}
	requireAbsent(t, v, "Rocks/Basalt.md")
}

func TestNoteRenameWithoutATitleKeyRenamesByFilename(t *testing.T) {
	v, registry := newGeologyVault(t)

	var out noteRenameOutput
	decodeInto(t, callTool(t, registry, geologyVault, "note_rename",
		`{"path":"Fossils.md","title":"Ammonites"}`), &out)
	if out.Path != "Ammonites.md" || out.Title != "Ammonites" || out.NameSource != "filename" {
		t.Errorf("result = %+v, want Ammonites.md, Ammonites, nameSource filename", out)
	}
	if out.Move == nil || out.Move.From != "Fossils.md" || out.Move.Path != "Ammonites.md" {
		t.Errorf("move = %+v, want from Fossils.md to Ammonites.md", out.Move)
	}
	requireContent(t, v, "Ammonites.md", "Shells kept in limestone.\n")
	requireAbsent(t, v, "Fossils.md")
}

func TestNoteRenameOntoAnExistingNoteIsRejected(t *testing.T) {
	v, registry := newGeologyVault(t)
	before, _ := v.Content("Rocks/Basalt.md")

	requireRejection(t, registry, geologyVault, "note_rename", `{"path":"Rocks/Basalt.md","title":"Granite"}`)
	requireContent(t, v, "Rocks/Basalt.md", before)
	requireContent(t, v, "Rocks/Granite.md", "---\ntitle: Granite\n---\nA plutonic rock.\n")
}

func getRejectionText(t *testing.T, registry *tools.Registry, vaultID, name, arguments string) string {
	t.Helper()
	res, err := registry.InvokeTool(t.Context(), tools.Call{VaultID: vaultID, Name: name, Arguments: []byte(arguments)})
	if err != nil {
		t.Fatalf("call %s: %v", name, err)
	}
	if res.GetError() == tools.ErrorCodeUnknownTool {
		t.Fatalf("call %s: the tool is unknown", name)
	}
	if !res.GetResult().GetIsError() {
		t.Fatalf("call %s %s was not rejected: %s", name, arguments, res.GetResult().GetStructured())
	}
	return res.GetResult().GetText()
}

func TestNoteRenameOntoATakenFilenameReportsTheUnchangedNote(t *testing.T) {
	_, registry := newGeologyVault(t)

	text := getRejectionText(t, registry, geologyVault, "note_rename", `{"path":"Rocks/Basalt.md","title":"Granite"}`)
	for _, part := range []string{"Rocks/Basalt.md", "Rocks/Granite.md"} {
		if !strings.Contains(text, part) {
			t.Errorf("the error %q does not name %s", text, part)
		}
	}
	lower := strings.ToLower(text)
	if !strings.Contains(lower, "not changed") && !strings.Contains(lower, "unchanged") {
		t.Errorf("the error %q does not state that the note was left unchanged", text)
	}
}

func TestNoteRenameOfAMissingNoteIsRejected(t *testing.T) {
	_, registry := newGeologyVault(t)
	requireRejection(t, registry, geologyVault, "note_rename", `{"path":"Rocks/Obsidian.md","title":"Glass"}`)
}

func TestNoteMoveMovesEachPathInOrderAndReportsTheOneThatFails(t *testing.T) {
	v, registry := newGeologyVault(t)

	var out noteMoveOutput
	decodeInto(t, callTool(t, registry, geologyVault, "note_move",
		`{"paths":["Rocks/Basalt.md","Rocks/Obsidian.md","Minerals.md"],"folder":"Archive"}`), &out)
	if len(out.Results) != 3 {
		t.Fatalf("got %d moves, want 3: %+v", len(out.Results), out.Results)
	}
	if got := out.Results[0]; got.From != "Rocks/Basalt.md" || got.Path != "Archive/Basalt.md" || got.Error != "" {
		t.Errorf("first = %+v, want Rocks/Basalt.md to Archive/Basalt.md", got)
	}
	if got := out.Results[1]; got.From != "Rocks/Obsidian.md" || got.Path != "" || got.Error == "" {
		t.Errorf("second = %+v, want Rocks/Obsidian.md with an error and no target", got)
	}
	if got := out.Results[2]; got.From != "Minerals.md" || got.Path != "Archive/Minerals.md" || got.Error != "" {
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
	if len(out.Results) != 1 || out.Results[0].From != "Rocks/Granite.md" || out.Results[0].Error == "" {
		t.Errorf("moves = %+v, want one error for Rocks/Granite.md", out.Results)
	}
	requireContent(t, v, "Archive/Granite.md", "An older granite note.\n")
	requireContent(t, v, "Rocks/Granite.md", "---\ntitle: Granite\n---\nA plutonic rock.\n")
}

func TestNoteMoveWithTheFolderSlashMovesToTheVaultRoot(t *testing.T) {
	v, registry := newGeologyVault(t)

	var out noteMoveOutput
	decodeInto(t, callTool(t, registry, geologyVault, "note_move", `{"paths":["Rocks/Granite.md"],"folder":"/"}`), &out)
	if len(out.Results) != 1 || out.Results[0].Path != "Granite.md" || out.Results[0].Error != "" {
		t.Errorf("results = %+v, want Rocks/Granite.md to Granite.md", out.Results)
	}
	requireContent(t, v, "Granite.md", "---\ntitle: Granite\n---\nA plutonic rock.\n")
	requireAbsent(t, v, "Rocks/Granite.md")
}

func TestNoteMoveWithAFolderEndingInASlashMovesIntoTheFolder(t *testing.T) {
	v, registry := newGeologyVault(t)

	var out noteMoveOutput
	decodeInto(t, callTool(t, registry, geologyVault, "note_move", `{"paths":["Fossils.md"],"folder":"Archive/Old/"}`), &out)
	if len(out.Results) != 1 || out.Results[0].Path != "Archive/Old/Fossils.md" || out.Results[0].Error != "" {
		t.Errorf("results = %+v, want Fossils.md to Archive/Old/Fossils.md", out.Results)
	}
	requireContent(t, v, "Archive/Old/Fossils.md", "Shells kept in limestone.\n")
}

func TestNoteMoveRejectsAnEmptyFolder(t *testing.T) {
	for _, folder := range []string{"", "   "} {
		v, registry := newGeologyVault(t)

		text := getRejectionText(t, registry, geologyVault, "note_move", `{"paths":["Rocks/Granite.md"],"folder":"`+folder+`"}`)
		if want := `note_move: folder is empty; pass "/" for the vault root`; text != want {
			t.Errorf("note_move with the folder %q gave the error %q, want %q", folder, text, want)
		}
		requireContent(t, v, "Rocks/Granite.md", "---\ntitle: Granite\n---\nA plutonic rock.\n")
		requireAbsent(t, v, "Granite.md")
	}
}

func TestNoteMoveWithoutAFolderIsRejected(t *testing.T) {
	v, registry := newGeologyVault(t)

	text := getRejectionText(t, registry, geologyVault, "note_move", `{"paths":["Rocks/Granite.md"]}`)
	if !strings.Contains(text, "folder") {
		t.Errorf("note_move without a folder gave the error %q, want it to name folder", text)
	}
	requireContent(t, v, "Rocks/Granite.md", "---\ntitle: Granite\n---\nA plutonic rock.\n")
	requireAbsent(t, v, "Granite.md")
}

func TestNoteMoveSchemaRequiresTheFolderAndNamesTheVaultRootSlash(t *testing.T) {
	_, registry := newGeologyVault(t)
	schema := getToolSchema(t, registry, "note_move", false)
	required, ok := schema["required"].([]any)
	if !ok {
		t.Fatalf("the input schema of note_move has no required list: %v", schema)
	}
	for _, name := range []string{"paths", "folder"} {
		if !slices.Contains(required, any(name)) {
			t.Errorf("the input schema of note_move requires %v, want %s among them", required, name)
		}
	}
	folder, ok := getSchemaProperties(t, schema)["folder"].(map[string]any)
	if !ok {
		t.Fatalf("the input schema of note_move has no folder property: %v", schema)
	}
	description, isText := folder["description"].(string)
	if !isText {
		t.Fatalf("the folder of note_move has no description: %v", folder)
	}
	if !strings.Contains(description, `"/"`) || !strings.Contains(description, "vault root") {
		t.Errorf("the folder of note_move is described as %q, want it to name \"/\" as the vault root", description)
	}

	listing, err := registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, info := range listing.Tools {
		if info.GetName() != "note_move" {
			continue
		}
		if text := info.GetDescription(); !strings.Contains(text, `"/"`) || !strings.Contains(text, "vault root") {
			t.Errorf("the description of note_move is %q, want it to name \"/\" as the vault root", text)
		}
		if text := info.GetDescription(); strings.Contains(text, "empty or absent") {
			t.Errorf("the description of note_move is %q, want no empty or absent folder", text)
		}
	}
}

// limitedBatchVault is the geology vault whose host rejects a move or a removal of more than 50 notes,
// and records the number of paths of every call it receives.
type limitedBatchVault struct {
	*hosttest.Vault
	pathCounts *[]int
}

// errHostBatchLimit is the error of the host for a batch of more than 50 notes.
var errHostBatchLimit = errors.New("too many notes: at most 50 per call")

func (l limitedBatchVault) With(domain.VaultID) limitedBatchVault { return l }

func (l limitedBatchVault) MoveNotes(ctx context.Context, paths []string, folder string) ([]host.NoteMove, error) {
	*l.pathCounts = append(*l.pathCounts, len(paths))
	if len(paths) > 50 {
		return nil, errHostBatchLimit
	}
	return l.Vault.MoveNotes(ctx, paths, folder)
}

func (l limitedBatchVault) RemoveNotes(ctx context.Context, paths []string) ([]host.NoteRemoval, error) {
	*l.pathCounts = append(*l.pathCounts, len(paths))
	if len(paths) > 50 {
		return nil, errHostBatchLimit
	}
	return l.Vault.RemoveNotes(ctx, paths)
}

func newLimitedBatchRegistry(t *testing.T) (*tools.Registry, *[]int) {
	t.Helper()
	v, _ := newGeologyVault(t)
	counts := &[]int{}
	registry, err := service.New(limitedBatchVault{Vault: v, pathCounts: counts}.vaults()).Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	return registry, counts
}

func TestNoteMoveOfFiftyOnePathsReturnsTheErrorOfTheHost(t *testing.T) {
	registry, counts := newLimitedBatchRegistry(t)

	text := getRejectionText(t, registry, geologyVault, "note_move", makePathsArguments(t, 51, map[string]any{"folder": "Archive"}))
	if !slices.Equal(*counts, []int{51}) {
		t.Errorf("the host received calls of %v paths, want one call of 51", *counts)
	}
	if !strings.Contains(text, errHostBatchLimit.Error()) {
		t.Errorf("note_move of 51 paths gave the error %q, want the error of the host %q", text, errHostBatchLimit)
	}
}

func TestNoteMoveOfFiftyOnePathsReturnsEachHostResult(t *testing.T) {
	_, registry := newGeologyVault(t)

	var out noteMoveOutput
	decodeInto(t, callTool(t, registry, geologyVault, "note_move",
		makePathsArguments(t, 51, map[string]any{"folder": "Archive"})), &out)
	if len(out.Results) != 51 {
		t.Errorf("got %d results, want 51 from the host", len(out.Results))
	}
}

func TestNoteRemoveMovesTheNoteToTheTrashAndReportsDanglingLinks(t *testing.T) {
	v, registry := newGeologyVault(t)

	var out noteRemoveOutput
	decodeInto(t, callTool(t, registry, geologyVault, "note_remove", `{"paths":["Rocks/Basalt.md"]}`), &out)
	if len(out.Results) != 1 {
		t.Fatalf("got %d removals, want 1: %+v", len(out.Results), out.Results)
	}
	got := out.Results[0]
	if got.Path != "Rocks/Basalt.md" || got.Error != "" {
		t.Errorf("removal = %+v, want Rocks/Basalt.md without an error", got)
	}
	if !strings.HasPrefix(got.TrashPath, ".trash/") {
		t.Errorf("trashPath = %q, want a path under .trash/", got.TrashPath)
	}
	if len(got.BrokenLinks) != 1 || got.BrokenLinks[0] != "Minerals.md" {
		t.Errorf("brokenLinks = %v, want [Minerals.md]", got.BrokenLinks)
	}
	requireContent(t, v, got.TrashPath, "---\ntitle: Basalt\n---\nA volcanic rock.\n")
	requireAbsent(t, v, "Rocks/Basalt.md")
}

func TestNoteRemoveReportsAFolderAndAMissingPathAndRemovesTheRest(t *testing.T) {
	v, registry := newGeologyVault(t)

	var out noteRemoveOutput
	decodeInto(t, callTool(t, registry, geologyVault, "note_remove",
		`{"paths":["Rocks","Fossils.md","Rocks/Obsidian.md"]}`), &out)
	if len(out.Results) != 3 {
		t.Fatalf("got %d removals, want 3: %+v", len(out.Results), out.Results)
	}
	if got := out.Results[0]; got.Path != "Rocks" || got.Error == "" || got.TrashPath != "" {
		t.Errorf("first = %+v, want the folder Rocks with an error", got)
	}
	if got := out.Results[1]; got.Path != "Fossils.md" || got.Error != "" || !strings.HasPrefix(got.TrashPath, ".trash/") {
		t.Errorf("second = %+v, want Fossils.md trashed", got)
	}
	if got := out.Results[2]; got.Path != "Rocks/Obsidian.md" || got.Error == "" {
		t.Errorf("third = %+v, want Rocks/Obsidian.md with an error", got)
	}
	requireContent(t, v, "Rocks/Basalt.md", "---\ntitle: Basalt\n---\nA volcanic rock.\n")
	requireContent(t, v, "Rocks/Granite.md", "---\ntitle: Granite\n---\nA plutonic rock.\n")
	requireAbsent(t, v, "Fossils.md")
}

func TestNoteRemoveOfFiftyOnePathsReturnsTheErrorOfTheHost(t *testing.T) {
	registry, counts := newLimitedBatchRegistry(t)

	text := getRejectionText(t, registry, geologyVault, "note_remove", makePathsArguments(t, 51, nil))
	if !slices.Equal(*counts, []int{51}) {
		t.Errorf("the host received calls of %v paths, want one call of 51", *counts)
	}
	if !strings.Contains(text, errHostBatchLimit.Error()) {
		t.Errorf("note_remove of 51 paths gave the error %q, want the error of the host %q", text, errHostBatchLimit)
	}
}

func TestNoteRemoveOfFiftyOnePathsReturnsEachHostResult(t *testing.T) {
	_, registry := newGeologyVault(t)

	var out noteRemoveOutput
	decodeInto(t, callTool(t, registry, geologyVault, "note_remove", makePathsArguments(t, 51, nil)), &out)
	if len(out.Results) != 51 {
		t.Errorf("got %d results, want 51 from the host", len(out.Results))
	}
}

func TestNoteRenameMoveAndRemoveReturnTheRenamedKeys(t *testing.T) {
	_, registry := newGeologyVault(t)

	rename := decodeAnswer(t, callTool(t, registry, geologyVault, "note_rename", `{"path":"Rocks/Basalt.md","title":"Flood Basalt"}`))
	requireKeys(t, "the answer of note_rename", rename, []string{"path", "title", "nameSource", "move"}, []string{"newPath"})
	if move, isObject := rename["move"].(map[string]any); isObject {
		requireKeys(t, "the move of note_rename", move, []string{"from", "path"}, []string{"to"})
	}
	move := decodeAnswer(t, callTool(t, registry, geologyVault, "note_move", `{"paths":["Rocks/Granite.md"],"folder":"Archive"}`))
	requireKeys(t, "the answer of note_move", move, []string{"results"}, []string{"moves"})
	for _, item := range requireList(t, "answer[results]", move["results"]) {
		if object, isObject := item.(map[string]any); isObject {
			requireKeys(t, "a result of note_move", object, []string{"from", "path"}, []string{"to"})
		}
	}
	remove := decodeAnswer(t, callTool(t, registry, geologyVault, "note_remove", `{"paths":["Fossils.md"]}`))
	requireKeys(t, "the answer of note_remove", remove, []string{"results"}, []string{"removals"})

	requireKeys(t, "the output schema of note_rename", getSchemaProperties(t, getToolSchema(t, registry, "note_rename", true)),
		[]string{"path", "move"}, []string{"newPath"})
	requireKeys(t, "the output schema of the note_rename move", getSchemaProperties(t, getToolSchema(t, registry, "note_rename", true), "move"),
		[]string{"from", "path"}, []string{"to"})
	requireKeys(t, "the output schema of note_move", getSchemaProperties(t, getToolSchema(t, registry, "note_move", true)),
		[]string{"results"}, []string{"moves"})
	requireKeys(t, "the output schema of a note_move result", getSchemaProperties(t, getToolSchema(t, registry, "note_move", true), "results"),
		[]string{"from", "path"}, []string{"to"})
	requireKeys(t, "the output schema of note_remove", getSchemaProperties(t, getToolSchema(t, registry, "note_remove", true)),
		[]string{"results"}, []string{"removals"})
}

func TestNoteRenameOntoTakenFilenameReturnsPathAndUnchangedNote(t *testing.T) {
	_, registry := newGeologyVault(t)

	text := getRejectionText(t, registry, geologyVault, "note_rename", `{"path":"Rocks/Basalt.md","title":"Granite"}`)

	if want := "note_rename: Rocks/Granite.md already exists; Rocks/Basalt.md is unchanged"; text != want {
		t.Errorf("the error is %q, want %q", text, want)
	}
}

func TestNoteRenameToAnEmptyTitleReturnsTheHostError(t *testing.T) {
	v, registry := newGeologyVault(t)
	before, _ := v.Content("Rocks/Basalt.md")

	text := getRejectionText(t, registry, geologyVault, "note_rename", `{"path":"Rocks/Basalt.md","title":""}`)

	if want := `a note cannot be given this title: "" generates empty filename`; !strings.Contains(text, want) {
		t.Errorf("note_rename to an empty title gave %q, want the host's error %q", text, want)
	}
	requireContent(t, v, "Rocks/Basalt.md", before)
}

func (l limitedBatchVault) vaults() func(domain.VaultID) service.Vault {
	return func(id domain.VaultID) service.Vault {
		on := l.With(id)
		return service.Vault{Files: on, Writer: on, Index: on, Notes: on, Links: on, Arranger: on, Editor: on, Selector: on}
	}
}
