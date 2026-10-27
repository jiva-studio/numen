package domain_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

func TestTheFrontTabIsTheOneWhoseIDIsInFront(t *testing.T) {
	open := domain.OpenTabs{
		Tabs: []domain.Tab{
			{ID: "a", Kind: "note", Path: "a.md"},
			{ID: "b", Kind: "reader:viewer", Path: "b.bin"},
			{ID: "c", Kind: "graph", State: "{\"focus\":\"a.md\"}"},
		},
		ActiveID: "b",
	}
	front, ok := open.GetActiveTab()
	if !ok || front.ID != "b" || front.Path != "b.bin" {
		t.Fatalf("got %+v, %v", front, ok)
	}
}

func TestAWindowWithNoTabInFrontAnswersWithNone(t *testing.T) {
	for name, open := range map[string]domain.OpenTabs{
		"nothing open":      {},
		"no front id":       {Tabs: []domain.Tab{{ID: "a"}}},
		"front id unknown":  {Tabs: []domain.Tab{{ID: "a"}}, ActiveID: "z"},
		"tab without an id": {Tabs: []domain.Tab{{Kind: "note"}}},
		"front id only":     {ActiveID: "a"},
	} {
		if front, ok := open.GetActiveTab(); ok || front.ID != "" || front.Kind != "" {
			t.Errorf("%s: got %+v, %v", name, front, ok)
		}
	}
}

func TestAnEmptyIDIsNotMatchedByAnEmptyFrontID(t *testing.T) {
	open := domain.OpenTabs{Tabs: []domain.Tab{{Kind: "note"}, {ID: "a"}}}
	if front, ok := open.GetActiveTab(); ok {
		t.Errorf("got %+v", front)
	}
}

func TestTheFirstTabOfTheFrontIDWinsAfterOthersThatDoNotMatch(t *testing.T) {
	open := domain.OpenTabs{
		Tabs:     []domain.Tab{{ID: "a"}, {ID: "", Kind: "x"}, {ID: "b", Kind: "y"}},
		ActiveID: "b",
	}
	if front, ok := open.GetActiveTab(); !ok || front.Kind != "y" {
		t.Errorf("got %+v, %v", front, ok)
	}
}
