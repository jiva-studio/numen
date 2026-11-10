package vault_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
)

func TestCurrentRefreshAnswersWithThePublishedRefreshUntilItIsRetired(t *testing.T) {
	fallback := vault.Refresh{Force: false}
	published := vault.Refresh{Force: true}
	current := &vault.CurrentRefresh{Fallback: func() vault.Refresh { return fallback }}

	if current.Get().Force {
		t.Fatal("before a publish the refresh is not the fallback")
	}
	current.Publish(published)
	if !current.Get().Force {
		t.Error("the published refresh was not returned")
	}
	current.Retire()
	if current.Get().Force {
		t.Error("a retired refresh was returned")
	}
}
