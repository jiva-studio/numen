package vault_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/check"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

type sized struct {
	summary domain.VaultSummary
	err     error
	asked   domain.VaultID
}

func (s *sized) GetSummary(_ context.Context, id domain.VaultID) (domain.VaultSummary, error) {
	s.asked = id
	return s.summary, s.err
}

// fixed is a check that finds the same problem every time.
type fixed struct {
	name  domain.Check
	quiet bool
}

func (f fixed) Name() domain.Check { return f.name }
func (f fixed) IsOptIn() bool      { return f.quiet }
func (f fixed) FindProblems(context.Context, domain.Vault) ([]domain.VaultProblem, error) {
	return []domain.VaultProblem{{Path: string(f.name) + ".md", Kind: f.name}}, nil
}

func TestTheSummaryIsAskedOfTheVaultItIsGiven(t *testing.T) {
	queries := &sized{summary: domain.VaultSummary{Notes: 3, Headings: 7}}
	v := domain.Vault{ID: "01V"}

	got, err := vault.NewGetSummary(queries).Execute(t.Context(), v)
	if err != nil {
		t.Fatal(err)
	}
	if got.Notes != 3 || got.Headings != 7 || queries.asked != "01V" {
		t.Errorf("the summary was %+v, asked of %q", got, queries.asked)
	}

	queries.err = errors.New("index gone")
	if _, err := vault.NewGetSummary(queries).Execute(t.Context(), v); !errors.Is(err, queries.err) {
		t.Errorf("a failing index answered %v", err)
	}
}

func TestProblemsNamingNoCheckRunTheLoudOnesAndSayWhichRan(t *testing.T) {
	checks := check.Checks{List: []check.Checker{
		fixed{name: "parse"}, fixed{name: "dangling", quiet: true},
	}}

	got, err := vault.NewListProblems(checks).Execute(t.Context(), domain.Vault{})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got.Ran, []domain.Check{"parse"}) || len(got.Problems) != 1 {
		t.Errorf("an unnamed look found %+v", got)
	}
}

func TestProblemsNamingAQuietCheckRunItAndSayWhichRan(t *testing.T) {
	checks := check.Checks{List: []check.Checker{
		fixed{name: "parse"}, fixed{name: "dangling", quiet: true},
	}}

	got, err := vault.NewListProblems(checks).Execute(t.Context(), domain.Vault{}, "dangling")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got.Ran, []domain.Check{"dangling"}) || len(got.Problems) != 1 || got.Problems[0].Kind != "dangling" {
		t.Errorf("a look at the quiet check found %+v", got)
	}

	if _, err := vault.NewListProblems(checks).Execute(t.Context(), domain.Vault{}, "nonsense"); err == nil {
		t.Error("a check nobody has was run")
	}
}
