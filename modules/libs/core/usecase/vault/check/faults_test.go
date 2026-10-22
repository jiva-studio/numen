package check_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/vault/check"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

var errUnreachable = errors.New("the index cannot be reached")

// failingQueries is an index that answers every question with an error.
type failingQueries struct{}

func (failingQueries) GetParseProblems(context.Context, domain.VaultID) ([]domain.VaultProblem, error) {
	return nil, errUnreachable
}

func (failingQueries) GetUnreadable(context.Context, domain.VaultID) ([]domain.VaultProblem, error) {
	return nil, errUnreachable
}

func (failingQueries) GetAmbiguousLinks(context.Context, domain.VaultID) ([]domain.AmbiguousLink, error) {
	return nil, errUnreachable
}

func (failingQueries) GetDanglingLinks(context.Context, domain.VaultID) ([]domain.ResolvedLink, error) {
	return nil, errUnreachable
}

// fixed is a check that finds what it was given, or fails with err.
type fixed struct {
	name  domain.Check
	silent bool
	found []domain.VaultProblem
	err   error
}

func (f fixed) Name() domain.Check { return f.name }
func (f fixed) IsOptIn() bool      { return f.silent }

func (f fixed) FindProblems(context.Context, domain.Vault) ([]domain.VaultProblem, error) {
	return f.found, f.err
}

// A check that cannot look for its question is an error naming the check, and
// no partial list of what the others found.
func TestACheckThatCannotLookFailsTheRunAndNamesItself(t *testing.T) {
	for _, name := range []domain.Check{
		domain.CheckParse, domain.CheckFrontmatter, domain.CheckAmbiguous, domain.CheckDangling,
	} {
		t.Run(string(name), func(t *testing.T) {
			found, err := check.Standard(failingQueries{}).Run(t.Context(), domain.Vault{}, name)
			if !errors.Is(err, errUnreachable) {
				t.Fatalf("the run failed with %v, want the index's error", err)
			}
			if !strings.HasPrefix(err.Error(), string(name)+": ") {
				t.Errorf("the error is %q, want it to begin with the check's name", err)
			}
			if found != nil {
				t.Errorf("a failed run came back with %+v", found)
			}
		})
	}
}

// The checks before a failing one do not stand in for the run.
func TestAFailingCheckAfterAFindingFailsTheRun(t *testing.T) {
	l := check.Checks{List: []check.Checker{
		fixed{name: "first", found: []domain.VaultProblem{{Path: "A.md", Kind: "first"}}},
		fixed{name: "second", err: errUnreachable},
	}}

	found, err := l.Run(t.Context(), domain.Vault{})
	if !errors.Is(err, errUnreachable) || found != nil {
		t.Errorf("the run came back with %+v and %v", found, err)
	}
	if !strings.HasPrefix(err.Error(), "second: ") {
		t.Errorf("the error is %q, want it to name the check that failed", err)
	}
}

// What a person has to open is together: the problems stand by note, and by
// check within a note, whatever order the checks found them in.
func TestProblemsStandByNoteAndThenByCheck(t *testing.T) {
	l := check.Checks{List: []check.Checker{
		fixed{name: "first", found: []domain.VaultProblem{
			{Path: "b.md", Kind: "first"},
			{Path: "a.md", Kind: "third"},
		}},
		fixed{name: "second", found: []domain.VaultProblem{
			{Path: "a.md", Kind: "second"},
			{Path: "c.md", Kind: "second"},
			{Path: "a.md", Kind: "first"},
		}},
	}}

	found, err := l.Run(t.Context(), domain.Vault{})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, p := range found {
		got = append(got, p.Path+"/"+string(p.Kind))
	}
	want := []string{"a.md/first", "a.md/second", "a.md/third", "b.md/first", "c.md/second"}
	if !slices.Equal(got, want) {
		t.Errorf("the problems stand as %v, want %v", got, want)
	}
}

// Every check there is is named, and nothing else.
func TestEveryCheckIsNamedAndNothingElse(t *testing.T) {
	l := check.Checks{List: []check.Checker{
		fixed{name: "first"},
		fixed{name: "second", silent: true},
	}}

	if got := l.Names(); !slices.Equal(got, []domain.Check{"first", "second"}) {
		t.Errorf("the names are %q, want both checks", got)
	}
	if got := l.Loud(); !slices.Equal(got, []domain.Check{"first"}) {
		t.Errorf("the checks that run unasked are %q, want the one", got)
	}
}

// Naming a check runs the one that answers to the name, once.
func TestNamingACheckRunsItOnce(t *testing.T) {
	l := check.Checks{List: []check.Checker{
		fixed{name: "twin", found: []domain.VaultProblem{{Path: "A.md", Kind: "twin"}}},
		fixed{name: "twin", found: []domain.VaultProblem{{Path: "B.md", Kind: "twin"}}},
	}}

	found, err := l.Run(t.Context(), domain.Vault{}, "twin")
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].Path != "A.md" {
		t.Errorf("naming a check found %+v, want what the first of that name found", found)
	}
}

// and nothing else in the list.
func TestAnUnknownCheckIsRefusedWithTheNamesThereAre(t *testing.T) {
	l := check.Checks{List: []check.Checker{fixed{name: "first"}, fixed{name: "second"}}}

	_, err := l.Run(t.Context(), domain.Vault{}, "spelling")
	if err == nil {
		t.Fatal("want an error")
	}
	if want := `there is no "spelling" check; there is first, second`; err.Error() != want {
		t.Errorf("the error is %q, want %q", err, want)
	}
}

// Problems of one note and one check keep the order they were found in.
func TestProblemsOfOneNoteAndOneCheckKeepTheirOrder(t *testing.T) {
	l := check.Checks{List: []check.Checker{fixed{name: "first", found: []domain.VaultProblem{
		{Path: "a.md", Kind: "first", Detail: "one"},
		{Path: "a.md", Kind: "first", Detail: "two"},
		{Path: "a.md", Kind: "second", Detail: "three"},
		{Path: "a.md", Kind: "second", Detail: "four"},
	}}}}

	found, err := l.Run(t.Context(), domain.Vault{})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, p := range found {
		got = append(got, p.Detail)
	}
	if want := []string{"one", "two", "three", "four"}; !slices.Equal(got, want) {
		t.Errorf("the problems stand as %v, want %v", got, want)
	}
}
