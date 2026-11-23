package indexing_test

import (
	"context"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// namedRefresh is a refresh that answers with its name, as the notes it refreshed.
type namedRefresh string

func (n namedRefresh) Execute(context.Context, domain.Vault, []string) (indexing.RefreshResult, error) {
	return indexing.RefreshResult{Indexed: []string{string(n)}}, nil
}

func getRefreshedBy(t *testing.T, refresh indexing.Refreshing) string {
	t.Helper()
	result, err := refresh.Execute(t.Context(), domain.Vault{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return result.Indexed[0]
}

func TestCurrentRefreshAnswersWithThePublishedRefreshUntilItIsRetired(t *testing.T) {
	current := &indexing.CurrentRefresh{Fallback: func() indexing.Refreshing { return namedRefresh("fallback") }}

	if got := getRefreshedBy(t, current.Get()); got != "fallback" {
		t.Fatalf("before a publish the refresh is %q, want the fallback", got)
	}
	current.Publish(namedRefresh("published"))
	if got := getRefreshedBy(t, current.Get()); got != "published" {
		t.Errorf("the refresh is %q after a publish, want the published one", got)
	}
	current.Retire()
	if got := getRefreshedBy(t, current.Get()); got != "fallback" {
		t.Errorf("the refresh is %q after it was retired, want the fallback", got)
	}
}
