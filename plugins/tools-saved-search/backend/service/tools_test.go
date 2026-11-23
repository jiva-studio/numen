package service_test

import (
	"encoding/json"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
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
	content, isThere := f.vault.ReadContent(notePath)
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
		if !slices.Contains(tool.Instructions, linkSentence) {
			t.Errorf("instructions of %s = %q, want the sentence about linking the note", tool.Name, tool.Instructions)
		}
	}
	want := map[string]tools.Effect{"saved_search_create": tools.ReadWrite, "saved_search_update": tools.ReadWrite}
	if !maps.Equal(effects, want) {
		t.Errorf("tools = %v, want %v", effects, want)
	}
}

func TestCreateDescriptionPointsToLibraryQueryAndTheStages(t *testing.T) {
	listing, err := newFixture(t, 10).registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	i := slices.IndexFunc(listing.Tools, func(tool *tools.ToolInfo) bool { return tool.Name == "saved_search_create" })
	if i < 0 {
		t.Fatalf("tools = %+v, want saved_search_create", listing.Tools)
	}
	for _, part := range []string{"library_query", ".rerank(", ".top(N)", "numen-query"} {
		if !strings.Contains(listing.Tools[i].Description, part) {
			t.Errorf("description = %q, want %q", listing.Tools[i].Description, part)
		}
	}
}

func TestCreateWritesASearchNoteThatUpdateChanges(t *testing.T) {
	f := newFixture(t, 10)

	created, failure := f.callTool(t, testVault, "saved_search_create", mustJSON(t, map[string]any{
		"title": "Harbor master: winter storms", "query": stormsQuery,
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
	if frontmatter["title"] != "Harbor master: winter storms" || frontmatter["id"] == nil {
		t.Errorf("frontmatter = %v, want the id and the title", frontmatter)
	}
	for _, key := range []string{"rerank", "limit"} {
		if _, isThere := frontmatter[key]; isThere {
			t.Errorf("frontmatter = %v, want no %s", frontmatter, key)
		}
	}
	if !strings.Contains(content, "```numen-query\n"+stormsQuery+"\n```\n") {
		t.Errorf("note = %q, want the query in a numen-query block", content)
	}

	updated, failure := f.callTool(t, testVault, "saved_search_update", mustJSON(t, map[string]any{"path": notePath, "query": wordsQuery}))
	if failure != "" {
		t.Fatal(failure)
	}

	if updated["path"] != notePath || updated["resultCount"] != float64(1) || updated["order"] != "fused" {
		t.Errorf("output = %v, want the run of the changed search, one result", updated)
	}
	if content, _ = readNote(t, f, notePath); !strings.Contains(content, "```numen-query\nwords(\"storm\")\n```\n") {
		t.Errorf("note = %q, want the new query", content)
	}
}

func TestCreateAndUpdateRunInteractiveSearchesTheCoreShows(t *testing.T) {
	f := newFixture(t, 10)

	if _, failure := f.callTool(t, testVault, "saved_search_create", mustJSON(t, map[string]any{"title": "Winter storms", "query": stormsQuery})); failure != "" {
		t.Fatal(failure)
	}
	if _, failure := f.callTool(t, testVault, "saved_search_update", mustJSON(t, map[string]any{"path": "Winter storms.md", "query": wordsQuery})); failure != "" {
		t.Fatal(failure)
	}

	started := f.searches.ListStarted()
	if len(started) == 0 {
		t.Fatal("no search started")
	}
	for _, start := range started {
		if start.IsBackground || start.Priority != host.SearchPriorityInteractive {
			t.Errorf("start = %+v, want interactive and shown as a task of the core", start)
		}
	}
	if removed := f.searches.ListRemoved(); len(removed) != len(started) {
		t.Errorf("removed %v of %d searches, want every search removed", removed, len(started))
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

	out, failure := f.callTool(t, testVault, "saved_search_create", mustJSON(t, map[string]any{"title": "Winter storms", "query": wordsQuery}))
	if failure != "" {
		t.Fatal(failure)
	}

	if out["resultCount"] != float64(1) || out["order"] != "fused" {
		t.Errorf("output = %v, want one result in fused order", out)
	}
	files, err := f.vault.Store().List(t.Context())
	if err != nil || len(files) != 1 {
		t.Errorf("store = %v, %v, want the cache of the first run", files, err)
	}
}

func TestCreateRejectsAQueryTheCoreReportsAndWritesNothing(t *testing.T) {
	f := newFixture(t, 10)

	out, failure := f.callTool(t, testVault, "saved_search_create", mustJSON(t, map[string]any{"title": "Winter storms", "query": brokenQuery}))

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
	if _, isThere := f.vault.ReadContent("Winter storms.md"); isThere || out["path"] != nil {
		t.Errorf("output = %v, want no note written", out)
	}
}

func TestCreateRejectsAPathThatExists(t *testing.T) {
	f := newFixture(t, 10)
	f.vault.WriteFile("Winter storms.md", "The winter of 1911.\n")

	_, failure := f.callTool(t, testVault, "saved_search_create", mustJSON(t, map[string]any{"title": "Winter storms", "query": stormsQuery}))

	if !strings.Contains(failure, "a file is at Winter storms.md; choose another title") {
		t.Errorf("failure = %q, want the existing file named", failure)
	}
	if content, _ := f.vault.ReadContent("Winter storms.md"); content != "The winter of 1911.\n" {
		t.Errorf("file = %q, want it unchanged", content)
	}
}

func TestCreateRejectsAnEmptyTitle(t *testing.T) {
	_, failure := newFixture(t, 10).callTool(t, testVault, "saved_search_create", mustJSON(t, map[string]any{"title": "  ", "query": stormsQuery}))

	if !strings.Contains(failure, "title is empty") {
		t.Errorf("failure = %q, want the empty title named", failure)
	}
}

func TestUpdateRejectsANoteThatIsNotASearch(t *testing.T) {
	f := newFixture(t, 10)

	_, failure := f.callTool(t, testVault, "saved_search_update", mustJSON(t, map[string]any{"path": "Notes/Tides.md", "query": stormsQuery}))

	if !strings.Contains(failure, "Notes/Tides.md is not a saved search") {
		t.Errorf("failure = %q, want the note named as not a saved search", failure)
	}
}

func TestUpdateWithoutAQueryIsAnError(t *testing.T) {
	f := newFixture(t, 10)
	f.addStormsNote()

	_, failure := f.callTool(t, testVault, "saved_search_update", mustJSON(t, map[string]any{"path": stormsPath}))

	if !strings.Contains(failure, "give query") {
		t.Errorf("failure = %q, want the missing query named", failure)
	}
}

func TestUpdateRejectsAQueryTheCoreReportsAndChangesNothing(t *testing.T) {
	f := newFixture(t, 10)
	f.addStormsNote()
	before, _ := f.vault.ReadContent(stormsPath)

	out, failure := f.callTool(t, testVault, "saved_search_update", mustJSON(t, map[string]any{"path": stormsPath, "query": brokenQuery}))

	if failure != "" {
		t.Fatalf("the tool failed with %q, want the problems in its output", failure)
	}
	if problems, isList := out["errors"].([]any); !isList || len(problems) == 0 {
		t.Errorf("output = %v, want the problems", out)
	}
	if after, _ := f.vault.ReadContent(stormsPath); after != before {
		t.Errorf("note = %q, want it unchanged", after)
	}
}

func TestSavedSearchToolsFailForACallWithoutAVault(t *testing.T) {
	f := newFixture(t, 10)
	f.addStormsNote()

	for name, arguments := range map[string]string{
		"saved_search_create": mustJSON(t, map[string]any{"title": "Winter storms", "query": stormsQuery}),
		"saved_search_update": mustJSON(t, map[string]any{"path": stormsPath, "query": wordsQuery}),
	} {
		if _, failure := f.callTool(t, "", name, arguments); !strings.Contains(failure, "the call names no vault") {
			t.Errorf("failure of %s = %q, want the call to name no vault", name, failure)
		}
	}
}
