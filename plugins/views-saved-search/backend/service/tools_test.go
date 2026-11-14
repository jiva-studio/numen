package service_test

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/markdown"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
)

const linkSentence = "Link the saved search note where you mention it, as a markdown link to the uri that saved_search_create returns."

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func readNote(t *testing.T, f *fixture, notePath string) (string, map[string]any) {
	t.Helper()
	content, isThere := f.vault.Content(notePath)
	if !isThere {
		t.Fatalf("no note at %s", notePath)
	}
	note := markdown.Parse(domain.FileStat{Path: notePath}, []byte(content))
	if note.Type != "search" {
		t.Errorf("type = %q, want search", note.Type)
	}
	return content, note.Frontmatter
}

func TestRegistryListsCreateAndUpdateAsReadWriteWithIndexRead(t *testing.T) {
	listing, err := newFixture(t, 10).registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	effects := map[string]tools.Effect{}
	for _, tool := range listing.Tools {
		effects[tool.Name] = tool.Effect
		if !slices.Equal(tool.Requires, []string{"index:read"}) || tool.Description == "" || tool.Title == "" {
			t.Errorf("tool = %+v, want a title, a description and index:read", tool)
		}
		if tool.Name != "saved_search_preview" && !slices.Contains(tool.Instructions, linkSentence) {
			t.Errorf("instructions of %s = %q, want the sentence about linking the note", tool.Name, tool.Instructions)
		}
	}
	want := map[string]tools.Effect{
		"saved_search_preview": tools.ReadOnly, "saved_search_create": tools.ReadWrite, "saved_search_update": tools.ReadWrite,
	}
	if len(effects) != len(want) {
		t.Errorf("tools = %v, want %v", effects, want)
	}
	for name, effect := range want {
		if got, isListed := effects[name]; !isListed || got != effect {
			t.Errorf("effect of %s = %v (listed %v), want %v", name, got, isListed, effect)
		}
	}
}

func TestCreateWritesASearchNoteThatUpdateChanges(t *testing.T) {
	f := newFixture(t, 10)
	f.addReranker()

	created, failure := f.callTool(t, testVault, "saved_search_create", mustJSON(t, map[string]any{
		"title": "Harbor master: winter storms", "query": stormsQuery, "rerank": stormsRerank, "limit": 30,
	}))
	if failure != "" {
		t.Fatal(failure)
	}

	notePath := "Harbor master- winter storms.md"
	if created["path"] != notePath || created["uri"] == "" || created["resultCount"] != float64(2) || created["order"] != "reranked" ||
		created["computedAt"] != "2026-11-14T09:30:00Z" {
		t.Errorf("output = %v, want the path, its address and the first run of two results", created)
	}
	content, frontmatter := readNote(t, f, notePath)
	if frontmatter["title"] != "Harbor master: winter storms" || frontmatter["rerank"] != stormsRerank || fmt.Sprint(frontmatter["limit"]) != "30" || frontmatter["id"] == nil {
		t.Errorf("frontmatter = %v, want the id, the title, the rerank text and the limit", frontmatter)
	}
	if !strings.Contains(content, "```numen-query\n"+stormsQuery+"\n```\n") {
		t.Errorf("note = %q, want the query in a numen-query block", content)
	}

	updated, failure := f.callTool(t, testVault, "saved_search_update", mustJSON(t, map[string]any{
		"path": notePath, "query": `words("storm")`, "limit": 1,
	}))
	if failure != "" {
		t.Fatal(failure)
	}

	if updated["path"] != notePath || updated["resultCount"] != float64(1) {
		t.Errorf("output = %v, want the run of the changed search, one result", updated)
	}
	content, frontmatter = readNote(t, f, notePath)
	if fmt.Sprint(frontmatter["limit"]) != "1" || frontmatter["rerank"] != stormsRerank || !strings.Contains(content, "```numen-query\nwords(\"storm\")\n```\n") {
		t.Errorf("note = %q, want the new query and limit and the rerank kept", content)
	}
}

func TestCreateWritesTheNoteInTheDefaultFolder(t *testing.T) {
	f := newFixture(t, 10)
	f.settings.Update(map[string]string{"defaultFolder": "Searches"})

	out, failure := f.callTool(t, testVault, "saved_search_create", mustJSON(t, map[string]any{"title": "Winter storms", "query": stormsQuery}))

	if failure != "" || out["path"] != "Searches/Winter storms.md" {
		t.Fatalf("output = %v, %q, want the note in Searches", out, failure)
	}
	if _, frontmatter := readNote(t, f, "Searches/Winter storms.md"); frontmatter["title"] != nil {
		t.Errorf("frontmatter = %v, want no title when the file name carries it", frontmatter)
	}
}

func TestCreateWritesTheNoteInTheGivenFolder(t *testing.T) {
	f := newFixture(t, 10)
	f.settings.Update(map[string]string{"defaultFolder": "Searches"})

	out, failure := f.callTool(t, testVault, "saved_search_create", mustJSON(t, map[string]any{
		"title": "Winter storms", "folder": "Harbor", "query": stormsQuery,
	}))

	if failure != "" || out["path"] != "Harbor/Winter storms.md" {
		t.Errorf("output = %v, %q, want the note in Harbor", out, failure)
	}
}

func TestCreateScansTheNote(t *testing.T) {
	f := newFixture(t, 10)

	if _, failure := f.callTool(t, testVault, "saved_search_create", mustJSON(t, map[string]any{"title": "Winter storms", "query": stormsQuery})); failure != "" {
		t.Fatal(failure)
	}

	if !slices.ContainsFunc(f.vault.Scans(), func(paths []string) bool { return slices.Equal(paths, []string{"Winter storms.md"}) }) {
		t.Errorf("scans = %v, want the new note scanned", f.vault.Scans())
	}
}

func TestCreateReturnsTheResultsOfTheFirstRun(t *testing.T) {
	f := newFixture(t, 10)

	out, failure := f.callTool(t, testVault, "saved_search_create", mustJSON(t, map[string]any{"title": "Winter storms", "query": stormsQuery}))
	if failure != "" {
		t.Fatal(failure)
	}

	if out["resultCount"] != float64(2) || out["order"] != "fused" {
		t.Errorf("output = %v, want two results in fused order", out)
	}
	files, err := f.vault.Store().List(t.Context())
	if err != nil || len(files) != 1 {
		t.Errorf("store = %v, %v, want the cache of the first run", files, err)
	}
}

func TestCreateRefusesAQueryThatDoesNotCompileAndWritesNothing(t *testing.T) {
	f := newFixture(t, 10)

	out, failure := f.callTool(t, testVault, "saved_search_create", mustJSON(t, map[string]any{"title": "Winter storms", "query": `meaning("storm") &&`}))

	if failure != "" {
		t.Fatalf("the tool failed with %q, want the problems in its output", failure)
	}
	problems, isList := out["errors"].([]any)
	if !isList || len(problems) == 0 {
		t.Fatalf("output = %v, want the problems", out)
	}
	if problem, isObject := problems[0].(map[string]any); !isObject || problem["line"] != float64(1) || problem["column"] != float64(20) {
		t.Errorf("problem = %v, want line 1, column 20 of the query", problems[0])
	}
	if _, isThere := f.vault.Content("Winter storms.md"); isThere || out["path"] != nil {
		t.Errorf("output = %v, want no note written", out)
	}
}

func TestCreateRefusesAPathThatExists(t *testing.T) {
	f := newFixture(t, 10)
	f.vault.File("Winter storms.md", "The winter of 1911.\n")

	_, failure := f.callTool(t, testVault, "saved_search_create", mustJSON(t, map[string]any{"title": "Winter storms", "query": stormsQuery}))

	if !strings.Contains(failure, "a file is at Winter storms.md; choose another title") {
		t.Errorf("failure = %q, want the existing file named", failure)
	}
	if content, _ := f.vault.Content("Winter storms.md"); content != "The winter of 1911.\n" {
		t.Errorf("file = %q, want it unchanged", content)
	}
}

func TestCreateRefusesAnEmptyTitle(t *testing.T) {
	_, failure := newFixture(t, 10).callTool(t, testVault, "saved_search_create", mustJSON(t, map[string]any{"title": "  ", "query": stormsQuery}))

	if !strings.Contains(failure, "title is empty") {
		t.Errorf("failure = %q, want the empty title named", failure)
	}
}

func TestUpdateRefusesANoteThatIsNotASearch(t *testing.T) {
	f := newFixture(t, 10)

	_, failure := f.callTool(t, testVault, "saved_search_update", mustJSON(t, map[string]any{"path": "Notes/Tides.md", "query": stormsQuery}))

	if !strings.Contains(failure, "Notes/Tides.md is not a saved search") {
		t.Errorf("failure = %q, want the note named as not a saved search", failure)
	}
}

func TestUpdateWithoutAFieldIsAnError(t *testing.T) {
	f := newFixture(t, 10)
	f.addStormsNote()

	_, failure := f.callTool(t, testVault, "saved_search_update", mustJSON(t, map[string]any{"path": stormsPath}))

	if !strings.Contains(failure, "give query, rerank or limit") {
		t.Errorf("failure = %q, want the missing fields named", failure)
	}
}

func TestUpdateRefusesAQueryThatDoesNotCompileAndChangesNothing(t *testing.T) {
	f := newFixture(t, 10)
	f.addStormsNote()
	before, _ := f.vault.Content(stormsPath)

	out, failure := f.callTool(t, testVault, "saved_search_update", mustJSON(t, map[string]any{"path": stormsPath, "query": `meaning("storm") &&`, "limit": 3}))

	if failure != "" {
		t.Fatalf("the tool failed with %q, want the problems in its output", failure)
	}
	if problems, isList := out["errors"].([]any); !isList || len(problems) == 0 {
		t.Errorf("output = %v, want the problems", out)
	}
	if after, _ := f.vault.Content(stormsPath); after != before {
		t.Errorf("note = %q, want it unchanged", after)
	}
}

func TestUpdateRemovesTheRerankTextGivenEmpty(t *testing.T) {
	f := newFixture(t, 10)
	f.addStormsNote()

	if _, failure := f.callTool(t, testVault, "saved_search_update", mustJSON(t, map[string]any{"path": stormsPath, "rerank": ""})); failure != "" {
		t.Fatal(failure)
	}

	if _, frontmatter := readNote(t, f, stormsPath); frontmatter["rerank"] != nil || frontmatter["id"] != stormsID {
		t.Errorf("frontmatter = %v, want the rerank text removed and the id kept", frontmatter)
	}
}

func TestSavedSearchToolsFailForACallWithoutAVault(t *testing.T) {
	f := newFixture(t, 10)
	f.addStormsNote()

	for name, arguments := range map[string]string{
		"saved_search_create":  mustJSON(t, map[string]any{"title": "Winter storms", "query": stormsQuery}),
		"saved_search_update":  mustJSON(t, map[string]any{"path": stormsPath, "limit": 3}),
		"saved_search_preview": `{"query":"meaning(\"tides\")"}`,
	} {
		if _, failure := f.callTool(t, "", name, arguments); !strings.Contains(failure, "the call names no vault") {
			t.Errorf("failure of %s = %q, want the call to name no vault", name, failure)
		}
	}
}

func TestPreviewToolReturnsTheCompileErrorWithItsPosition(t *testing.T) {
	out, failure := newFixture(t, 10).callTool(t, testVault, "saved_search_preview", `{"query":"meaning(\"tides\") &&"}`)
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
	out, failure := newFixture(t, 10).callTool(t, testVault, "saved_search_preview", `{"query":"meaning(\"tides\")","limit":5}`)
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
	result, isObject := results[0].(map[string]any)
	if !isObject {
		t.Fatalf("results[0] = %v, want an object", results[0])
	}
	for key, want := range map[string]any{
		"path": "Notes/Tides.md", "title": "Tide Tables", "text": "The tide at the quay.", "start": float64(0), "length": float64(21),
		"line": float64(3), "uri": "numen:Notes%2FTides.md?start=0&length=21", "score": float64(1), "chunkCount": float64(1),
	} {
		if result[key] != want {
			t.Errorf("result[%q] = %v, want %v", key, result[key], want)
		}
	}
}

func TestPreviewToolRefusesANegativeLimit(t *testing.T) {
	_, failure := newFixture(t, 10).callTool(t, testVault, "saved_search_preview", `{"query":"meaning(\"tides\")","limit":-1}`)

	if failure == "" {
		t.Error("the tool accepted the limit -1")
	}
}
