package hosttest

import (
	"context"
	"fmt"
	"slices"

	notesdomain "github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
)

// getDefaultChecks is the checks that run when none is named. dangling is opt-in.
func getDefaultChecks() []notesdomain.Check {
	return []notesdomain.Check{
		notesdomain.CheckParse, notesdomain.CheckFrontmatter, notesdomain.CheckAmbiguous,
	}
}

// GetProblems runs the checks named over the notes the index holds, or every check that is not
// opt-in when none is named. A name that is no check is an error.
func (v *Vault) GetProblems(ctx context.Context, checks []string) (host.VaultProblems, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.gate(ctx); err != nil {
		return host.VaultProblems{}, err
	}
	wanted := getDefaultChecks()
	if len(checks) > 0 {
		wanted = make([]notesdomain.Check, 0, len(checks))
		for _, name := range checks {
			check := notesdomain.Check(name)
			if check != notesdomain.CheckDangling && !slices.Contains(getDefaultChecks(), check) {
				return host.VaultProblems{}, fmt.Errorf("there is no %q check", name)
			}
			wanted = append(wanted, check)
		}
	}
	notes := v.parsed()
	out := host.VaultProblems{Checks: make([]string, 0, len(wanted))}
	for _, check := range wanted {
		out.Checks = append(out.Checks, string(check))
		out.Found = append(out.Found, v.findProblems(notes, check)...)
	}
	return out, nil
}

// findProblems is what one check finds in the notes. The caller holds the lock.
func (v *Vault) findProblems(notes map[string]domain.Note, check notesdomain.Check) []domain.VaultProblem {
	var out []domain.VaultProblem
	for _, notePath := range sortedPaths(notes) {
		note := notes[notePath]
		switch check {
		case notesdomain.CheckParse:
			for _, detail := range note.Problems {
				out = append(out, domain.VaultProblem{Path: notePath, Kind: check, Detail: detail})
			}
		case notesdomain.CheckFrontmatter:
			if note.FrontmatterErr != "" {
				out = append(out, domain.VaultProblem{Path: notePath, Kind: check, Detail: note.FrontmatterErr})
			}
		case notesdomain.CheckAmbiguous, notesdomain.CheckDangling:
			out = append(out, v.findLinkProblems(notes, notePath, check)...)
		}
	}
	return out
}

// findLinkProblems is the links of the note that reach several notes, for the ambiguous check, or
// none, for the dangling check. The caller holds the lock.
func (v *Vault) findLinkProblems(
	notes map[string]domain.Note, notePath string, check notesdomain.Check,
) []domain.VaultProblem {
	var out []domain.VaultProblem
	for _, link := range notes[notePath].Links {
		if link.Target.Scheme != domain.SchemeName && link.Target.Scheme != domain.SchemeNote {
			continue
		}
		r := domain.ResolvedLink{Link: link, From: notePath}
		v.resolve(notes, notePath, &r)
		switch {
		case check == notesdomain.CheckAmbiguous && r.IsAmbiguous:
			out = append(out, domain.VaultProblem{
				Path: notePath, Kind: check, Target: link.Target, Candidates: candidates(notes, link.Target.Value),
				Detail: fmt.Sprintf("%s answers for several notes", link.Target.Value),
			})
		case check == notesdomain.CheckDangling && r.To == "":
			out = append(out, domain.VaultProblem{
				Path: notePath, Kind: check, Target: link.Target,
				Detail: fmt.Sprintf("nothing in this vault answers to %s", link.Target.Value),
			})
		}
	}
	return out
}
