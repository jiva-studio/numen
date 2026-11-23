package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/api"
)

const fixtureDir = "../../testdata/api"

func getResultPaths(res api.GetResultsResponse) []string {
	paths := make([]string, len(res.Results))
	for i, result := range res.Results {
		paths[i] = result.Path
	}
	return paths
}

func getKeys(t *testing.T, raw []byte) []string {
	t.Helper()
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		t.Fatal(err)
	}
	return slices.Sorted(maps.Keys(object))
}

func TestGetResultsReturnsTheCacheWithoutSearching(t *testing.T) {
	f := newFixture(t, 10)
	f.addStormsNote()
	f.handleIndexChange(t)
	starts := f.getStartCount()

	res := f.getResults(t, stormsPath)

	if got := getResultPaths(res); !slices.Equal(got, []string{"Harbor/Gale.md", "Logbooks/1911.md"}) {
		t.Errorf("results = %v, want the cached results in their order", got)
	}
	if res.Path != stormsPath || res.Title != "Winter storms" || res.Query != stormsQuery {
		t.Errorf("response = %+v, want the note's path, title and query", res)
	}
	if res.ComputedAt != "2026-11-14T09:30:00Z" || res.IsStale || res.IsRunning || res.Order != "reranked" {
		t.Errorf("response = %+v, want computed now, not stale, not running, reranked", res)
	}
	if res.FailedAt != "" || res.Failure != "" || res.Warnings == nil || res.Errors == nil {
		t.Errorf("response = %+v, want no failure and empty lists", res)
	}
	if res.Results[0].Title != "Gale" || res.Results[0].Score == nil || *res.Results[0].Score != 0.9 {
		t.Errorf("result = %+v, want the title and the score the core gave", res.Results[0])
	}
	if f.getStartCount() != starts {
		t.Error("GetResults started a search, want none")
	}
}

func TestGetResultsOfASearchNeverComputedHasNoTime(t *testing.T) {
	f := newFixture(t, 10)
	f.addStormsNote()

	res := f.getResults(t, stormsPath)

	if res.ComputedAt != "" || res.Order != "" || res.Results == nil || len(res.Results) != 0 || !res.IsStale {
		t.Errorf("response = %+v, want no time, no order, no result and stale", res)
	}
	if f.getStartCount() != 0 {
		t.Error("GetResults ran the search")
	}
}

func TestGetResultsReadsTheQueryFromTheNoteAsItIsNow(t *testing.T) {
	f := newFixture(t, 10)
	f.addStormsNote()
	f.handleIndexChange(t)

	f.vault.WriteFile(stormsPath, formatSearchNote(stormsID, wordsQuery))
	res := f.getResults(t, stormsPath)

	if res.Query != wordsQuery || !res.IsStale {
		t.Errorf("response = %+v, want the query of the note now, stale", res)
	}
	if len(res.Results) != 2 || res.ComputedAt != "2026-11-14T09:30:00Z" {
		t.Errorf("response = %+v, want the cached results of the last run", res)
	}
}

func TestGetResultsKeepsTheLastGoodResultsWithTheProblemsOfTheQuery(t *testing.T) {
	f := newFixture(t, 10)
	f.addStormsNote()
	f.handleIndexChange(t)

	f.clock.Advance(time.Minute)
	f.vault.WriteFile(stormsPath, formatSearchNote(stormsID, brokenQuery))
	f.handleIndexChange(t)

	res := f.getResults(t, stormsPath)
	if len(res.Errors) != 1 || res.Errors[0].Line != 1 || res.Errors[0].Column != 20 {
		t.Errorf("errors = %+v, want the problem at line 1, column 20 of the query", res.Errors)
	}
	if got := getResultPaths(res); len(got) != 2 || res.ComputedAt != "2026-11-14T09:30:00Z" || res.FailedAt != "2026-11-14T09:31:00Z" {
		t.Errorf("response = %+v, want the last good results and the time of the failed run", res)
	}
}

func TestGetResultsCountsTheProblemsFromTheQueryAfterTheFrontmatterGrows(t *testing.T) {
	f := newFixture(t, 10)
	f.vault.WriteFile(stormsPath, formatSearchNote(stormsID, brokenQuery))
	f.handleIndexChange(t)

	grown := strings.Replace(formatSearchNote(stormsID, brokenQuery), "---\n", "---\ntitle: Winter storms\nsource: logbooks\n", 1)
	f.vault.WriteFile(stormsPath, grown)
	res := f.getResults(t, stormsPath)

	if len(res.Errors) != 1 || res.Errors[0].Line != 1 || res.Errors[0].Column != 20 {
		t.Errorf("errors = %+v, want the problem at line 1, column 20 of the query", res.Errors)
	}
}

func TestGetResultsReturnsAProblemOfTheNoteAtLineOneColumnOneOfTheQuery(t *testing.T) {
	f := newFixture(t, 10)
	f.vault.WriteFile(stormsPath, "---\nid: "+stormsID+"\ntype: search\n---\nStories of storms.\n\n```numen-query\n```\n")

	res := f.getResults(t, stormsPath)

	if len(res.Errors) != 1 || res.Errors[0].Line != 1 || res.Errors[0].Column != 1 || !strings.Contains(res.Errors[0].Message, "empty") {
		t.Errorf("errors = %+v, want the empty block at line 1, column 1 of the query", res.Errors)
	}
}

func TestGetResultsKeepsTheRangeOfAProblemAndNamesItsSuggestionOnce(t *testing.T) {
	f := newFixture(t, 0)
	f.searches.SetAnswer(stormsQuery, host.SearchState{Errors: []host.SearchProblem{
		{Line: 1, Column: 17, EndLine: 1, EndColumn: 22, Message: "Put this text in quotes: Кришна"},
		{Line: 2, Column: 4, Message: "Unknown function meanin", Suggestion: "meaning"},
		{Line: 2, Column: 20, Message: "Unknown stage rerenk: did you mean rerank?", Suggestion: "rerank"},
	}})
	f.addStormsNote()
	f.handleIndexChange(t)

	res := f.getResults(t, stormsPath)

	want := []api.Problem{
		{Line: 1, Column: 17, EndLine: 1, EndColumn: 22, Message: "Put this text in quotes: Кришна"},
		{Line: 2, Column: 4, Message: "Unknown function meanin: did you mean meaning?"},
		{Line: 2, Column: 20, Message: "Unknown stage rerenk: did you mean rerank?"},
	}
	if !slices.Equal(res.Errors, want) {
		t.Errorf("errors = %+v, want %+v", res.Errors, want)
	}
}

func TestGetResultsShowsAFailingProviderAsAProblemWithTheLastGoodResults(t *testing.T) {
	f := newFixture(t, 0)
	f.addStormsNote()
	f.handleIndexChange(t)

	f.searches.SetAnswer(stormsQuery, host.SearchState{Errors: []host.SearchProblem{{Message: "the reranking service did not answer"}}})
	f.clock.Advance(time.Minute)
	f.handleIndexChange(t)

	res := f.getResults(t, stormsPath)
	if len(res.Errors) != 1 || res.Errors[0].Message != "the reranking service did not answer" || res.FailedAt != "2026-11-14T09:31:00Z" {
		t.Errorf("response = %+v, want the problem of the provider and the time of the failed run", res)
	}
	if got := getResultPaths(res); !slices.Equal(got, []string{"Harbor/Gale.md", "Logbooks/1911.md"}) || res.Results[1].Text != mooringText {
		t.Errorf("results = %+v, want the last good results", res.Results)
	}
}

func TestGetResultsIsStaleAfterAnIndexChange(t *testing.T) {
	f := newFixture(t, 10)
	f.addStormsNote()
	f.handleIndexChange(t)

	f.clock.Advance(time.Minute)
	f.handleIndexChange(t)

	if res := f.getResults(t, stormsPath); !res.IsStale || res.ComputedAt != "2026-11-14T09:30:00Z" {
		t.Errorf("response = %+v, want the first results, stale", res)
	}
}

func TestGetResultsIsStaleWhenTheCoreReportsTheIndexChangedDuringTheRun(t *testing.T) {
	f := newFixture(t, 10)
	answer := getStormsAnswer(mooringText)
	answer.IsStale = true
	f.searches.SetAnswer(stormsQuery, answer)
	f.addStormsNote()

	f.handleIndexChange(t)

	if res := f.getResults(t, stormsPath); !res.IsStale || len(res.Results) != 2 {
		t.Errorf("response = %+v, want the results of the run, stale", res)
	}
}

func TestGetResultsShowsTheFailureOfTheLastRunWithTheLastGoodResults(t *testing.T) {
	f := newFixture(t, 0)
	f.addStormsNote()
	f.handleIndexChange(t)

	f.search.setFailure(errors.New("the core did not answer"))
	f.clock.Advance(time.Minute)
	f.handleIndexChange(t)

	res := f.getResults(t, stormsPath)
	if res.ComputedAt != "2026-11-14T09:30:00Z" || res.FailedAt != "2026-11-14T09:31:00Z" || !strings.Contains(res.Failure, "the core did not answer") {
		t.Errorf("response = %+v, want the last good time, the time and reason of the failure", res)
	}
	if got := getResultPaths(res); !slices.Equal(got, []string{"Harbor/Gale.md", "Logbooks/1911.md"}) {
		t.Errorf("results = %+v, want the last good results", res.Results)
	}
}

func TestGetResultsOfAMissingNoteIsNotFound(t *testing.T) {
	failure := newFixture(t, 10).call(t, api.MethodGetResults, api.GetResultsRequest{Path: "Searches/Gales.md"}, &api.GetResultsResponse{})

	if failure.Code != server.CodeNotFound || failure.Message != "there is no note at Searches/Gales.md" {
		t.Errorf("failure = %+v, want not_found naming the path", failure)
	}
}

func TestGetResultsOfAnotherTypeIsInvalid(t *testing.T) {
	failure := newFixture(t, 10).call(t, api.MethodGetResults, api.GetResultsRequest{Path: "Notes/Tides.md"}, &api.GetResultsResponse{})

	if failure.Code != server.CodeInvalidArgument || failure.Message != "Notes/Tides.md is not a saved search" {
		t.Errorf("failure = %+v, want invalid_argument naming the path", failure)
	}
}

func TestGetResultsWithoutAShownVaultFailsThePrecondition(t *testing.T) {
	f := newFixture(t, 10)
	f.addStormsNote()
	f.vault.Hide()

	failure := f.call(t, api.MethodGetResults, api.GetResultsRequest{Path: stormsPath}, &api.GetResultsResponse{})

	if failure.Code != server.CodeFailedPrecondition || failure.Message != "the window shows no vault" {
		t.Errorf("failure = %+v, want failed_precondition", failure)
	}
}

func TestGetResultsMatchesTheFixture(t *testing.T) {
	f := newFixture(t, 10)
	f.addStormsNote()
	f.handleIndexChange(t)
	want, err := os.ReadFile(fixtureDir + "/GetResults.response.json")
	if err != nil {
		t.Fatal(err)
	}

	response, err := f.web.Client().Do(f.newRequest(t, api.MethodGetResults, api.GetResultsRequest{Path: stormsPath}))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Error(err)
		}
	}()
	var got map[string]json.RawMessage
	if err := json.NewDecoder(response.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}

	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if gotKeys, wantKeys := getKeys(t, raw), getKeys(t, want); !slices.Equal(gotKeys, wantKeys) {
		t.Errorf("keys = %v, want those of the fixture %v", gotKeys, wantKeys)
	}
	var results []json.RawMessage
	if err := json.Unmarshal(got["results"], &results); err != nil || len(results) == 0 {
		t.Fatalf("results = %s, want some", got["results"])
	}
	for _, key := range []string{"path", "title", "kind", "text", "start", "length", "line", "uri", "score"} {
		if !slices.Contains(getKeys(t, results[0]), key) {
			t.Errorf("result %s has no key %q", results[0], key)
		}
	}
}

func TestWatchResultsSendsTheStateFirstAndThenEachChange(t *testing.T) {
	f := newFixture(t, 0)
	f.addStormsNote()
	f.handleIndexChange(t)
	want, err := os.ReadFile(fixtureDir + "/WatchResults.response.json")
	if err != nil {
		t.Fatal(err)
	}

	events := f.openStream(t, api.MethodWatchResults, api.WatchResultsRequest{Path: stormsPath})

	first := receiveEvent(t, events)
	if first.name != "" || first.data != `{"computedAt":"2026-11-14T09:30:00Z","isRunning":false}` {
		t.Errorf("first event = %+v, want the state of the cache", first)
	}
	if gotKeys, wantKeys := getKeys(t, []byte(first.data)), getKeys(t, want); !slices.Equal(gotKeys, wantKeys) {
		t.Errorf("keys = %v, want those of the fixture %v", gotKeys, wantKeys)
	}

	f.searches.SetAnswer(stormsQuery, getStormsAnswer(mooringsText))
	f.clock.Advance(time.Minute)
	f.handleIndexChange(t)

	if start := receiveEvent(t, events); !strings.Contains(start.data, `"isRunning":true`) {
		t.Errorf("event = %+v, want the start of the run", start)
	}
	if write := receiveEvent(t, events); write.data != `{"computedAt":"2026-11-14T09:31:00Z","isRunning":false}` {
		t.Errorf("event = %+v, want the write of the run", write)
	}
}

func TestWatchResultsOfAMissingNoteEndsWithNotFound(t *testing.T) {
	f := newFixture(t, 0)

	events := f.openStream(t, api.MethodWatchResults, api.WatchResultsRequest{Path: "Searches/Gales.md"})

	received := receiveEvent(t, events)
	var failure server.APIError
	if err := json.Unmarshal([]byte(received.data), &failure); err != nil || received.name != "error" || failure.Code != server.CodeNotFound {
		t.Errorf("event = %+v, want an error event not_found", received)
	}
}

func TestUpdateSearchWritesTheNote(t *testing.T) {
	f := newFixture(t, 10)
	f.addStormsNote()
	f.handleIndexChange(t)

	var res api.UpdateSearchResponse
	failure := f.call(t, api.MethodUpdateSearch, api.UpdateSearchRequest{Path: stormsPath, Query: wordsQuery}, &res)
	if failure.Code != "" {
		t.Fatal(failure)
	}

	if res.Errors == nil || len(res.Errors) != 0 {
		t.Errorf("response = %+v, want an empty list of problems", res)
	}
	content, frontmatter := readNote(t, f, stormsPath)
	if !strings.Contains(content, "```numen-query\nwords(\"storm\")\n```\n") || frontmatter["id"] != stormsID {
		t.Errorf("note = %q, want the new query and the id kept", content)
	}
}

func TestUpdateSearchRunsTheSearchInteractivelyWithinTheCooldown(t *testing.T) {
	f := newFixture(t, 10)
	f.addStormsNote()
	events := f.watchFirstRun(t)
	f.clock.Advance(time.Minute)
	before := f.getStartCount()

	if failure := f.call(t, api.MethodUpdateSearch, api.UpdateSearchRequest{Path: stormsPath, Query: wordsQuery}, &api.UpdateSearchResponse{}); failure.Code != "" {
		t.Fatal(failure)
	}

	wantEvents := []string{
		`{"computedAt":"","isRunning":true}`,
		`{"computedAt":"","isRunning":false}`,
		`{"computedAt":"","isRunning":true}`,
		`{"computedAt":"2026-11-14T09:31:00Z","isRunning":false}`,
	}
	for _, data := range wantEvents {
		if event := receiveEvent(t, events); event.data != data {
			t.Errorf("event = %+v, want %s: the check of the edit, its end, the start of the run, its write", event, data)
		}
	}
	if res := f.getResults(t, stormsPath); len(res.Results) != 1 || res.ComputedAt != "2026-11-14T09:31:00Z" || res.IsStale {
		t.Errorf("response = %+v, want one result computed after the edit", res)
	}
	for _, start := range f.searches.ListStarted()[before:] {
		if start.Query != wordsQuery || start.IsBackground || start.Priority != host.SearchPriorityInteractive {
			t.Errorf("start = %+v, want the new query, interactive, shown as a task of the core", start)
		}
	}
}

func TestUpdateSearchReturnsTheProblemsOfTheCoreAndChangesNothing(t *testing.T) {
	f := newFixture(t, 10)
	f.addStormsNote()
	before, _ := f.vault.ReadContent(stormsPath)

	var res api.UpdateSearchResponse
	failure := f.call(t, api.MethodUpdateSearch, api.UpdateSearchRequest{Path: stormsPath, Query: brokenQuery}, &res)
	if failure.Code != "" {
		t.Fatal(failure)
	}

	if len(res.Errors) != 1 || res.Errors[0].Line != 1 || res.Errors[0].Column != 20 || !strings.Contains(res.Errors[0].Message, "unexpected end") {
		t.Errorf("errors = %+v, want the problem at line 1, column 20 of the query", res.Errors)
	}
	if after, _ := f.vault.ReadContent(stormsPath); after != before {
		t.Errorf("note = %q, want it unchanged", after)
	}
}

func TestUpdateSearchOfANoteThatIsNotASearchIsInvalid(t *testing.T) {
	f := newFixture(t, 10)

	failure := f.call(t, api.MethodUpdateSearch, api.UpdateSearchRequest{Path: "Notes/Tides.md", Query: stormsQuery}, &api.UpdateSearchResponse{})

	if failure.Code != server.CodeInvalidArgument || failure.Message != "Notes/Tides.md is not a saved search" {
		t.Errorf("failure = %+v, want invalid_argument naming the path", failure)
	}
}

func TestUpdateSearchOfAMissingNoteIsNotFound(t *testing.T) {
	failure := newFixture(t, 10).call(t, api.MethodUpdateSearch, api.UpdateSearchRequest{Path: "Searches/Gales.md", Query: stormsQuery}, &api.UpdateSearchResponse{})

	if failure.Code != server.CodeNotFound {
		t.Errorf("failure = %+v, want not_found", failure)
	}
}

func TestUpdateSearchReplacedByANewerOneBeforeItsCheckEndsWritesNothingAndAnswersNoProblem(t *testing.T) {
	f := newFixture(t, 10)
	f.addStormsNote()
	release := f.searches.HoldSearch(wordsQuery)
	defer release()
	before := f.getStartCount()
	first := make(chan server.APIError, 1)
	var firstResponse api.UpdateSearchResponse

	go func() {
		first <- f.call(t, api.MethodUpdateSearch, api.UpdateSearchRequest{Path: stormsPath, Query: wordsQuery}, &firstResponse)
	}()
	waiting, stopWaiting := context.WithTimeout(t.Context(), 5*time.Second)
	defer stopWaiting()
	for f.getStartCount() == before {
		if waiting.Err() != nil {
			t.Fatal("the check of the first change did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if failure := f.call(t, api.MethodUpdateSearch, api.UpdateSearchRequest{Path: stormsPath, Query: brokenQuery}, &api.UpdateSearchResponse{}); failure.Code != "" {
		t.Fatal(failure)
	}

	if failure := <-first; failure.Code != "" || firstResponse.Errors == nil || len(firstResponse.Errors) != 0 {
		t.Errorf("the replaced change answered %+v, %+v, want no problem", failure, firstResponse)
	}
	if content, _ := readNote(t, f, stormsPath); strings.Contains(content, wordsQuery) {
		t.Errorf("note = %q, want the query of the replaced change not written", content)
	}
}
