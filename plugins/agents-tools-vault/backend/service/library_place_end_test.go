package service_test

import (
	"encoding/json"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/service"
)

const (
	riverTalkPath  = "recordings/River Talk.mp3"
	riverGuidePath = "library/River Guide.pdf"
	riverFirst     = "The river bends at the mill. "
	riverSecond    = "Otters live under the bank. "
	riverPage      = "The kingfisher dives for minnows. "
)

func newRiverLibrary(t *testing.T) *tools.Registry {
	t.Helper()
	vault := hosttest.New(testVault)
	vault.File(riverTalkPath, talkBytes)
	vault.File(riverGuidePath, pdfBytes)

	texts := hosttest.NewTexts(testVault, readerID)
	texts.File(riverTalkPath, []byte(talkBytes))
	texts.File(riverGuidePath, []byte(pdfBytes))
	putText(t, texts.As(scribeID), scribeID, riverTalkPath, "transcript", []byte(talkBytes), []domain.TextSegment{
		{Text: riverFirst, Selector: `{"t":[12.5,20]}`},
		{Text: riverSecond, Selector: `{"t":[20]}`},
	})
	putText(t, texts, readerID, riverGuidePath, "page-text", []byte(pdfBytes), []domain.TextSegment{
		{Text: riverPage, Selector: `{"page":1}`},
	})

	search := hosttest.NewSearch(testVault)
	search.SetIndexedResults([]host.SearchResult{
		{Path: riverTalkPath, Text: "The river bends at the mill.", PlaceLabel: "00:12", Start: 0, Length: 28, Selector: `{"t":[12.5,20]}`},
		{Path: riverTalkPath, Text: "Otters live under the bank.", PlaceLabel: "00:20", Start: 29, Length: 27, Selector: `{"t":[20]}`},
		{Path: riverGuidePath, Text: "The kingfisher dives for minnows.", PlaceLabel: "page 1", Start: 0, Length: 33, Selector: `{"page":1}`},
	})

	registry, err := service.New(vaultsOf(vault)).WithLibrary(libraryOf(texts, search)).Registry()
	if err != nil {
		t.Fatal(err)
	}
	return registry
}

func getPlaces(t *testing.T, registry *tools.Registry, name, arguments, key string) []map[string]any {
	t.Helper()
	structured := callTool(t, registry, testVault, name, arguments)
	var answer map[string]any
	if err := json.Unmarshal(structured, &answer); err != nil {
		t.Fatalf("%s: %v in %s", name, err, structured)
	}
	list := requireList(t, name+" "+key, answer[key])
	places := make([]map[string]any, len(list))
	for i, item := range list {
		element, isObject := item.(map[string]any)
		if !isObject {
			t.Fatalf("%s %s[%d] is %v, not an object", name, key, i, item)
		}
		place, isObject := element["place"].(map[string]any)
		if !isObject {
			t.Fatalf("%s %s[%d] is %v, want a place", name, key, i, element)
		}
		places[i] = place
	}
	return places
}

func requirePlaceSeconds(t *testing.T, what string, place map[string]any, position, end float64) {
	t.Helper()
	if got, isNumber := place["positionSeconds"].(float64); !isNumber || got != position {
		t.Errorf("%s has positionSeconds %v, want %v: %v", what, place["positionSeconds"], position, place)
	}
	if got, isNumber := place["endSeconds"].(float64); !isNumber || got != end {
		t.Errorf("%s has endSeconds %v, want %v: %v", what, place["endSeconds"], end, place)
	}
}

func requireNoEndSeconds(t *testing.T, what string, place map[string]any) {
	t.Helper()
	if end, has := place["endSeconds"]; has {
		t.Errorf("%s has endSeconds %v, want no endSeconds key: %v", what, end, place)
	}
}

func TestLibraryReadOfARecordingReturnsTheEndOfEachRange(t *testing.T) {
	registry := newRiverLibrary(t)

	places := getPlaces(t, registry, "library_read", `{"path":"`+riverTalkPath+`"}`, "ranges")

	if len(places) != 2 {
		t.Fatalf("library_read returned %d ranges, want 2: %v", len(places), places)
	}
	requirePlaceSeconds(t, "the range of {\"t\":[12.5,20]}", places[0], 12.5, 20)
	if got, isNumber := places[1]["positionSeconds"].(float64); !isNumber || got != 20 {
		t.Errorf("the range of {\"t\":[20]} has positionSeconds %v, want 20: %v", places[1]["positionSeconds"], places[1])
	}
	requireNoEndSeconds(t, "the range of {\"t\":[20]}", places[1])
}

func TestLibraryReadOfAPDFPageReturnsNoEndSeconds(t *testing.T) {
	registry := newRiverLibrary(t)

	places := getPlaces(t, registry, "library_read", `{"path":"`+riverGuidePath+`"}`, "ranges")

	if len(places) != 1 {
		t.Fatalf("library_read returned %d ranges, want 1: %v", len(places), places)
	}
	if page, isNumber := places[0]["page"].(float64); !isNumber || page != 1 {
		t.Errorf("the range of the PDF has page %v, want 1: %v", places[0]["page"], places[0])
	}
	requireNoEndSeconds(t, "the range of the PDF page", places[0])
}

func TestLibrarySearchMatchOfARecordingReturnsTheEndOfTheRange(t *testing.T) {
	registry := newRiverLibrary(t)

	places := getPlaces(t, registry, "library_search", `{"query":"the"}`, "matches")

	if len(places) != 3 {
		t.Fatalf("library_search returned %d matches, want 3: %v", len(places), places)
	}
	requirePlaceSeconds(t, "the match of {\"t\":[12.5,20]}", places[0], 12.5, 20)
	requireNoEndSeconds(t, "the match of {\"t\":[20]}", places[1])
	requireNoEndSeconds(t, "the match of the PDF page", places[2])
}

func TestLibraryPlaceSchemaDescribesEndSeconds(t *testing.T) {
	registry := newRiverLibrary(t)

	for _, c := range []struct{ tool, list string }{{"library_read", "ranges"}, {"library_search", "matches"}} {
		properties := getSchemaProperties(t, getToolSchema(t, registry, c.tool, true), c.list, "place")
		requireKeys(t, "the place of "+c.tool, properties, []string{"positionSeconds", "endSeconds"}, nil)
	}
}
