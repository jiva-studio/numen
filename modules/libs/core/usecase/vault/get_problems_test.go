package vault_test

import (
	"context"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/vault/check"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

type fixed struct {
	name  domain.Check
	quiet bool
}

func (f fixed) Name() domain.Check { return f.name }
func (f fixed) IsOptIn() bool      { return f.quiet }
func (f fixed) FindProblems(context.Context, domain.Vault) ([]domain.VaultProblem, error) {
	return []domain.VaultProblem{{Path: string(f.name) + ".md", Kind: f.name}}, nil
}

func newChecks() *vault.GetProblems {
	return vault.NewGetProblems(check.Checks{List: []check.Checker{
		fixed{name: "parse"}, fixed{name: "dangling", quiet: true},
	}})
}

func TestProblemsNamingNoCheckRunTheLoudOnes(t *testing.T) {
	report, err := newChecks().Execute(t.Context(), domain.Vault{})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Found) != 1 || report.Found[0].Kind != "parse" || !slices.Equal(report.Ran, []domain.Check{"parse"}) {
		t.Errorf("an unnamed look found %+v", report)
	}
}

func TestProblemsNamingAQuietCheckRunIt(t *testing.T) {
	list := newChecks()

	report, err := list.Execute(t.Context(), domain.Vault{}, "dangling")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Found) != 1 || report.Found[0].Kind != "dangling" || !slices.Equal(report.Ran, []domain.Check{"dangling"}) {
		t.Errorf("a look at the quiet check found %+v", report)
	}
	if _, err := list.Execute(t.Context(), domain.Vault{}, "nonsense"); err == nil {
		t.Error("a check nobody has was run")
	}
}
