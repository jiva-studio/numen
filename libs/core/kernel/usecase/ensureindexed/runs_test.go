package ensureindexed_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/ensureindexed"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func getUnreadVault() domain.Vault {
	return domain.Vault{ID: "unread", Name: "Sanskrit", Path: "/vaults/Sanskrit"}
}

// A vault forgotten after a clean indexing is claimable again.
func TestAVaultThatWasReadIsClaimedAgainOnceForgotten(t *testing.T) {
	r := ensureindexed.NewRuns(t.Context(), func(context.Context, domain.Vault) error { return nil })

	r.Record(getUnreadVault(), nil)
	if r.Claim(getUnreadVault()).Read != nil {
		t.Fatalf("a vault indexed clean was claimed before it was forgotten")
	}

	r.Clear(getUnreadVault().ID)

	if r.Claim(getUnreadVault()).Read == nil {
		t.Fatalf("a vault forgotten after a clean indexing was not claimable again")
	}
}

// A vault is claimed by one indexer at a time: the claim that begins an indexing
// reports it is underway, a claim while it runs reports so and begins nothing, a
// failed indexing returns why and begins nothing, and a vault indexed cleanly
// begins nothing and is not underway.
func TestAVaultIsClaimedForOneIndexingAtATime(t *testing.T) {
	r := ensureindexed.NewRuns(t.Context(), func(context.Context, domain.Vault) error { return nil })
	v := domain.Vault{ID: "one"}

	claim := r.Claim(v)
	if claim.Read == nil || claim.Closing == nil || !claim.Underway || claim.Reason != "" {
		t.Fatalf("the first claim came back %v %v %v %q", claim.Read != nil, claim.Closing != nil, claim.Underway, claim.Reason)
	}

	claim = r.Claim(v)
	if claim.Read != nil || !claim.Underway || claim.Reason != "" {
		t.Errorf("a claim while it ran came back %v %v %q", claim.Read != nil, claim.Underway, claim.Reason)
	}

	r.Record(v, errors.New("the folder is not there"))
	claim = r.Claim(v)
	if claim.Read != nil || claim.Underway || claim.Reason != "the folder is not there" {
		t.Errorf("a claim after a failure came back %v %v %q", claim.Read != nil, claim.Underway, claim.Reason)
	}

	r.Clear(v.ID)
	r.Record(v, nil)
	claim = r.Claim(v)
	if claim.Read != nil || claim.Underway || claim.Reason != "" {
		t.Errorf("a claim after a clean indexing came back %v %v %q", claim.Read != nil, claim.Underway, claim.Reason)
	}
}

func TestAWindowThatReadsNothingClaimsNothing(t *testing.T) {
	r := &ensureindexed.Runs{}
	claim := r.Claim(domain.Vault{ID: "one"})
	if claim.Read != nil || claim.Underway || claim.Reason != "" {
		t.Errorf("the claim came back %v %v %q", claim.Read != nil, claim.Underway, claim.Reason)
	}
}

// A vault read into the index drops the failure shown for it and tells the subscribers.
func TestAVaultReadReleasesItsRunAndReportsAChange(t *testing.T) {
	runs, changes := &ensureindexed.Runs{}, &ensureindexed.Changes{}
	runs.SetIndexer(t.Context(), func(context.Context, domain.Vault) error { return nil })
	runs.Record(getUnreadVault(), nil)
	line, stop := changes.Subscribe()
	defer stop()

	ensureindexed.NewReportRead(runs, changes).Execute(getUnreadVault())

	if runs.Claim(getUnreadVault()).Read == nil {
		t.Error("the vault read was not released")
	}
	select {
	case <-line:
	default:
		t.Error("no change was reported")
	}
}
