package vault_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/check"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

type sized struct {
	port.NoteQueries
	summary domain.VaultSummary
	err     error
	asked   domain.VaultID
}

func (s *sized) GetSummary(_ context.Context, id domain.VaultID) (domain.VaultSummary, error) {
	s.asked = id
	return s.summary, s.err
}

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

func newChecks() *vault.ListProblems {
	return vault.NewListProblems(check.Checks{List: []check.Checker{
		fixed{name: "parse"}, fixed{name: "dangling", quiet: true},
	}})
}

func TestProblemsNamingNoCheckRunTheLoudOnes(t *testing.T) {
	list := newChecks()

	found, err := list.Execute(t.Context(), domain.Vault{})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].Kind != "parse" || !slices.Equal(list.GetDefaultChecks(), []domain.Check{"parse"}) {
		t.Errorf("an unnamed look found %+v", found)
	}
}

func TestProblemsNamingAQuietCheckRunIt(t *testing.T) {
	list := newChecks()

	found, err := list.Execute(t.Context(), domain.Vault{}, "dangling")
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].Kind != "dangling" {
		t.Errorf("a look at the quiet check found %+v", found)
	}
	if _, err := list.Execute(t.Context(), domain.Vault{}, "nonsense"); err == nil {
		t.Error("a check nobody has was run")
	}
}

func TestQueriesThatAnswerNoCheckRunNone(t *testing.T) {
	list := vault.NewStandardListProblems(&sized{})

	found, err := list.Execute(t.Context(), domain.Vault{})
	if err != nil || len(found) != 0 || len(list.GetDefaultChecks()) != 0 {
		t.Errorf("found %+v, %v, with checks %v", found, err, list.GetDefaultChecks())
	}
}
