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
		FrontID: "b",
	}
	front, ok := open.GetFrontTab()
	if !ok || front.ID != "b" || front.Path != "b.bin" {
		t.Fatalf("got %+v, %v", front, ok)
	}
}

func TestAWindowWithNoTabInFrontAnswersWithNone(t *testing.T) {
	for name, open := range map[string]domain.OpenTabs{
		"nothing open":      {},
		"no front id":       {Tabs: []domain.Tab{{ID: "a"}}},
		"front id unknown":  {Tabs: []domain.Tab{{ID: "a"}}, FrontID: "z"},
		"tab without an id": {Tabs: []domain.Tab{{Kind: "note"}}},
		"front id only":     {FrontID: "a"},
	} {
		if front, ok := open.GetFrontTab(); ok || front.ID != "" || front.Kind != "" {
			t.Errorf("%s: got %+v, %v", name, front, ok)
		}
	}
}

func TestAnEmptyIDIsNotMatchedByAnEmptyFrontID(t *testing.T) {
	open := domain.OpenTabs{Tabs: []domain.Tab{{Kind: "note"}, {ID: "a"}}}
	if front, ok := open.GetFrontTab(); ok {
		t.Errorf("got %+v", front)
	}
}

func TestTheFirstTabOfTheFrontIDWinsAfterOthersThatDoNotMatch(t *testing.T) {
	open := domain.OpenTabs{
		Tabs:    []domain.Tab{{ID: "a"}, {ID: "", Kind: "x"}, {ID: "b", Kind: "y"}},
		FrontID: "b",
	}
	if front, ok := open.GetFrontTab(); !ok || front.Kind != "y" {
		t.Errorf("got %+v, %v", front, ok)
	}
}
