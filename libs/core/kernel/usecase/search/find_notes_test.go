package search_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/note/domain"
)

type nameIndex []domain.NameMatch

func (n nameIndex) GetNameMatches(_ context.Context, _ domain.VaultID, _ string, limit int) ([]domain.NameMatch, error) {
	return n[:min(limit, len(n))], nil
}

type kindsOf struct {
	kinds    map[string]string
	err      error
	requests [][]string
}

func (k *kindsOf) Execute(_ context.Context, _ domain.Vault, paths []string) (map[string]string, error) {
	k.requests = append(k.requests, paths)
	return k.kinds, k.err
}

func TestNamesComeWithTheKindOfTabOfTheirNotes(t *testing.T) {
	kinds := &kindsOf{kinds: map[string]string{"Heat.md": "reader"}}
	find := search.NewFindNames(nameIndex{{Path: "Heat.md", Title: "Heat"}, {Path: "Entropy.md", Title: "Entropy"}}, kinds)

	got, err := find.Execute(t.Context(), domain.Vault{ID: "v"}, "e", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Matches) != 2 || got.TabKinds["Heat.md"] != "reader" {
		t.Errorf("found %+v", got)
	}
	if len(kinds.requests) != 1 || len(kinds.requests[0]) != 2 {
		t.Errorf("the tab kinds were requests for %v, want one call naming both notes", kinds.requests)
	}
}

func TestNamesAreLimited(t *testing.T) {
	find := search.NewFindNames(nameIndex{{Path: "a.md"}, {Path: "b.md"}, {Path: "c.md"}}, &kindsOf{})
	got, err := find.Execute(t.Context(), domain.Vault{ID: "v"}, "x", 2)
	if err != nil || len(got.Matches) != 2 {
		t.Errorf("found %d, %v, want 2", len(got.Matches), err)
	}
}

func TestNamesFailWhenTheKindsOfTabCannotBeChosen(t *testing.T) {
	broken := errors.New("plugin host gone")
	find := search.NewFindNames(nameIndex{{Path: "a.md"}}, &kindsOf{err: broken})
	if _, err := find.Execute(t.Context(), domain.Vault{ID: "v"}, "a", 5); !errors.Is(err, broken) {
		t.Errorf("answered %v", err)
	}
}
