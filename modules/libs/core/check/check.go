// Package check is what a vault is checked against.
//
// A vault is edited by hand, by other tools and by whatever else the person
// runs, so it accumulates things the application will not act on and will not
// guess at: a link with no role, a frontmatter block that is not YAML, a name
// two notes answer to. None of it stops a scan, and without somewhere to look
// all of it is invisible.
//
// The shape is a list of checks, one to a file. A new thing worth noticing is a
// new file and a line in Standard, and nothing that already works changes.
package check

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// A Checker is one thing that can be wrong with a vault, and what notices it.
//
// The name it answers to is a domain.Check: what the check is called travels
// out to a person, and the thing that runs stays in here.
type Checker interface {
	// Name is what it is called, and what a person asks for by.
	Name() domain.Check
	// Look reports what it found, each finding against the note somebody would
	// open to settle it.
	Look(ctx context.Context, v domain.Vault) ([]domain.VaultProblem, error)
	// Quiet reports whether this check is left out unless it is asked for.
	Quiet() bool
}

// Checks is the set of checks run over one vault.
//
// Nothing is kept. Two of these checks read what a scan already stored, and two
// work the whole vault out from scratch — and those two must not be stored,
// because what they answer stops being true when a note somewhere else moves,
// and a written-down answer would go on saying it.
type Checks struct {
	Checks []Checker
}

// Standard is the set of checks a vault is held to.
func Standard(queries port.ProblemQueries) Checks {
	return Checks{Checks: []Checker{
		parseCheck{queries},
		frontmatterCheck{queries},
		ambiguousCheck{queries},
		danglingCheck{queries},
	}}
}

// Run looks the vault over.
//
// Named checks are the ones that run; naming none runs every check that is not
// quiet. A quiet check is one whose findings are ordinary in a vault somebody is
// still writing, and which would bury the rest if it arrived unasked.
func (c Checks) Run(ctx context.Context, v domain.Vault, named ...domain.Check) ([]domain.VaultProblem, error) {
	wanted, err := c.wanted(named)
	if err != nil {
		return nil, err
	}

	var out []domain.VaultProblem
	for _, check := range wanted {
		found, err := check.Look(ctx, v)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", check.Name(), err)
		}
		out = append(out, found...)
	}

	// By note, so that what a person has to open is together, and by check
	// within it, so that the order does not move between two runs.
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Check < out[j].Check
	})
	return out, nil
}

// Names is every check there is, quiet ones included.
func (c Checks) Names() []domain.Check {
	out := make([]domain.Check, 0, len(c.Checks))
	for _, check := range c.Checks {
		out = append(out, check.Name())
	}
	return out
}

func (c Checks) wanted(named []domain.Check) ([]Checker, error) {
	if len(named) == 0 {
		return c.loud(), nil
	}

	var out []Checker
	for _, name := range named {
		found := false
		for _, check := range c.Checks {
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
	for _, check := range c.loud() {
		out = append(out, check.Name())
	}
	return out
}

// loud is every check that does not wait to be asked for.
func (c Checks) loud() []Checker {
	var out []Checker
	for _, check := range c.Checks {
		if !check.Quiet() {
			out = append(out, check)
		}
	}
	return out
}
