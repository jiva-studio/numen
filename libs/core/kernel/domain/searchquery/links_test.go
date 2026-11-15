package searchquery_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func TestChooseTargetRefusesANameSeveralNotesAnswerTo(t *testing.T) {
	candidates := []string{"b/Entropy.md", "a/Entropy.md"}
	_, err := searchquery.ChooseTarget("Entropy", candidates)
	ambiguous, ok := errors.AsType[*searchquery.AmbiguousTargetError](err)
	if !ok || !slices.Equal(ambiguous.Paths, []string{"a/Entropy.md", "b/Entropy.md"}) {
		t.Fatalf("an ambiguous name returned %v, want both paths in order", err)
	}
	for _, name := range []string{"a/Entropy", "a/Entropy.md"} {
		if chosen, err := searchquery.ChooseTarget(name, candidates); err != nil || chosen != "a/Entropy.md" {
			t.Errorf("%s chose %q, %v, want a/Entropy.md", name, chosen, err)
		}
	}
	if chosen, err := searchquery.ChooseTarget("Entropy", []string{"x/Entropy.md"}); err != nil || chosen != "x/Entropy.md" {
		t.Errorf("one candidate chose %q, %v", chosen, err)
	}
}

func TestChooseFilePathPrefersAnIndexedPathAndKeepsAURL(t *testing.T) {
	from := "scans/index.md"
	letter := domain.ParseAddress("letter.pdf")
	if got := searchquery.ChooseFilePath(from, letter, map[string]bool{"scans/letter.pdf": true}); got != "scans/letter.pdf" {
		t.Errorf("got %s, want the path in the folder of the note", got)
	}
	if got := searchquery.ChooseFilePath(from, letter, nil); got != "letter.pdf" {
		t.Errorf("got %s, want the path from the root", got)
	}
	if got := searchquery.ChooseFilePath(from, domain.ParseAddress("https://example.org/x"), nil); got != "https://example.org/x" {
		t.Errorf("got %s, want the URL as written", got)
	}
	if got := searchquery.CleanPath("/a/./b/../c.pdf"); got != "a/c.pdf" {
		t.Errorf("CleanPath returned %s", got)
	}
	if got := searchquery.CleanPath("../outside.pdf"); got != "../outside.pdf" {
		t.Errorf("a path above the root became %s", got)
	}
}

func TestGetRelativesReadsTheEdgeAtEitherEnd(t *testing.T) {
	links := map[string][]domain.ResolvedLink{"Mechanics.md": {
		{Link: domain.Link{Role: domain.RoleParent}, From: "Mechanics.md", To: "Physics.md", ToVault: "v"},
		{Link: domain.Link{Role: domain.RoleParent}, From: "Mechanics.md", To: "Elsewhere.md", ToVault: "other"},
	}}
	backlinks := map[string][]domain.ResolvedLink{"Mechanics.md": {
		{Link: domain.Link{Role: domain.RoleChild}, From: "Science.md", To: "Mechanics.md"},
		{Link: domain.Link{Role: domain.RoleParent}, From: "Statics.md", To: "Mechanics.md"},
	}}
	if got := searchquery.GetRelatives(searchquery.DirectionAbove, "", "v", links, backlinks); !slices.Equal(got, []string{"Physics.md", "Science.md"}) {
		t.Errorf("above: %v", got)
	}
	if got := searchquery.GetRelatives(searchquery.DirectionBelow, "", "v", links, backlinks); !slices.Equal(got, []string{"Statics.md"}) {
		t.Errorf("below: %v", got)
	}
}

func TestEveryCatalogEntryHasADescriptionAndAnExample(t *testing.T) {
	for _, f := range searchquery.GetBuiltinCatalog().Functions {
		if f.Description == "" || f.Example == "" {
			t.Errorf("%s has no description or example", f.Name)
		}
	}
}
