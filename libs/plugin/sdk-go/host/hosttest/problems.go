package hosttest

import (
	"context"
	"fmt"
	"slices"
	"strings"

	notesdomain "github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

func getDefaultChecks() []notesdomain.Check {
	return []notesdomain.Check{
		notesdomain.CheckParse, notesdomain.CheckFrontmatter, notesdomain.CheckAmbiguous,
	}
}

func getCheckList() string {
	names := make([]string, 0, len(getDefaultChecks())+1)
	for _, check := range append(getDefaultChecks(), notesdomain.CheckDangling) {
		names = append(names, string(check))
	}
	return strings.Join(names, ", ")
}

// GetProblems runs the named checks over the notes in the index, or every check that is not
// opt-in when none is named. An unknown check name returns an error.
func (v *Vault) GetProblems(ctx context.Context, checks []string) (host.VaultProblems, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.gate(ctx); err != nil {
		return host.VaultProblems{}, err
	}
	selectedChecks := getDefaultChecks()
	if len(checks) > 0 {
		selectedChecks = make([]notesdomain.Check, 0, len(checks))
		for _, name := range checks {
			check := notesdomain.Check(name)
			if check != notesdomain.CheckDangling && !slices.Contains(getDefaultChecks(), check) {
				return host.VaultProblems{}, fmt.Errorf("%q is not a check; the checks are %s", name, getCheckList())
			}
			selectedChecks = append(selectedChecks, check)
		}
	}
	notes := v.parsed()
	out := host.VaultProblems{Checks: make([]string, 0, len(selectedChecks))}
	for _, check := range selectedChecks {
		out.Checks = append(out.Checks, string(check))
		out.Found = append(out.Found, v.findProblems(notes, check)...)
	}
	return out, nil
}

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
				Detail: fmt.Sprintf("%s resolves to several notes", link.Target.Value),
			})
		case check == notesdomain.CheckDangling && r.To == "":
			out = append(out, domain.VaultProblem{
				Path: notePath, Kind: check, Target: link.Target,
				Detail: fmt.Sprintf("%s resolves to no note", link.Target.Value),
			})
		}
	}
	return out
}

func (v *Vault) GetParseProblems(ctx context.Context) ([]domain.VaultProblem, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.gate(ctx); err != nil {
		return nil, err
	}
	notes := v.parsed()
	var out []domain.VaultProblem
	for _, p := range sortedPaths(notes) {
		for _, detail := range notes[p].Problems {
			out = append(out, domain.VaultProblem{Path: p, Kind: domain.CheckParse, Detail: detail})
		}
	}
	return out, nil
}
