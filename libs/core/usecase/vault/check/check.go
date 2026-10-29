// Package check is what a vault is checked against.
//
// A vault is edited by hand, by other tools and by other programs, so it
// accumulates things the application does not act on and does not guess at: a
// link with no role, a frontmatter block that is not YAML, a name shared by two
// notes. None of it stops a scan, and without a check it goes unreported.
//
// The structure is a list of checks, one per file. A new check is a new file
// and a line in Standard; existing checks are unchanged.
package check

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/problemport"
)

// A Checker is one thing that can be wrong with a vault, and what notices it.
//
// The name it answers to is a domain.Check: the check's name is reported to
// users, and the implementation stays in this package.
type Checker interface {
	// Name is the check's name, used to select it.
	Name() domain.Check
	// FindProblems reports the problems found, each against the note to open to
	// resolve it.
	FindProblems(ctx context.Context, v domain.Vault) ([]domain.VaultProblem, error)
	// IsOptIn reports whether this check is omitted unless requested.
	IsOptIn() bool
}

// Checks is the set of checks run over one vault.
//
// Nothing is kept. Two of these checks read what a scan already stored, and two
// work the whole vault out from scratch — and those two must not be stored,
// because what they answer stops being true when a note somewhere else moves,
// and a written-down answer would go on saying it.
type Checks struct {
	List []Checker
}

func Standard(queries problemport.ProblemQueries) Checks {
	return Checks{List: []Checker{
		parseCheck{queries},
		frontmatterCheck{queries},
		ambiguousCheck{queries},
		danglingCheck{queries},
	}}
}

// Run looks the vault over.
//
// Named checks are the ones that run; naming none runs every check that is not
// opt-in. An opt-in check has findings that are ordinary in a vault under
// active editing and would bury the rest if reported unrequested.
func (c Checks) Run(ctx context.Context, v domain.Vault, named ...domain.Check) ([]domain.VaultProblem, error) {
	wanted, err := c.getWanted(named)
	if err != nil {
		return nil, err
	}

	var out []domain.VaultProblem
	for _, check := range wanted {
		found, err := check.FindProblems(ctx, v)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", check.Name(), err)
		}
		out = append(out, found...)
	}

	// Sorted by note, so the notes to open are grouped, and by check within a note,
	// so the order is stable between runs.
	slices.SortStableFunc(out, func(a, b domain.VaultProblem) int {
		return cmp.Or(cmp.Compare(a.Path, b.Path), cmp.Compare(a.Kind, b.Kind))
	})
	return out, nil
}

// Names is every check there is, quiet ones included.
func (c Checks) Names() []domain.Check {
	out := make([]domain.Check, 0, len(c.List))
	for _, check := range c.List {
		out = append(out, check.Name())
	}
	return out
}

func (c Checks) getWanted(named []domain.Check) ([]Checker, error) {
	if len(named) == 0 {
		return c.getDefaultCheckers(), nil
	}

	var out []Checker
	for _, name := range named {
		found := false
		for _, check := range c.List {
			if check.Name() == name {
				out, found = append(out, check), true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("there is no %q check; there is %s", name, list(c.Names()))
		}
	}
	return out, nil
}

func list(names []domain.Check) string {
	as := make([]string, 0, len(names))
	for _, name := range names {
		as = append(as, string(name))
	}
	return strings.Join(as, ", ")
}

// Loud is the name of every check that runs when none is named.
func (c Checks) Loud() []domain.Check {
	var out []domain.Check
	for _, check := range c.getDefaultCheckers() {
		out = append(out, check.Name())
	}
	return out
}

// getDefaultCheckers returns every check that is not opt-in.
func (c Checks) getDefaultCheckers() []Checker {
	var out []Checker
	for _, check := range c.List {
		if !check.IsOptIn() {
			out = append(out, check)
		}
	}
	return out
}
