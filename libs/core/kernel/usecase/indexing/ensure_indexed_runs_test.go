package indexing_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func getUnread() domain.Vault {
	return domain.Vault{ID: "unread", Name: "Sanskrit", Path: "/vaults/Sanskrit"}
}

// A vault forgotten after a clean indexing is claimable again.
func TestAVaultThatWasReadIsClaimedAgainOnceForgotten(t *testing.T) {
	r := &indexing.Runs{}
	r.SetIndexer(t.Context(), func(context.Context, domain.Vault) error { return nil })

	r.Record(getUnread(), nil)
	if read, _, _, _ := r.Claim(getUnread()); read != nil {
		t.Fatalf("a vault indexed clean was claimed before it was forgotten")
	}

	r.Clear(getUnread().ID)

	read, _, _, _ := r.Claim(getUnread())
	if read == nil {
		t.Fatalf("a vault forgotten after a clean indexing was not claimable again")
	}
}

// A vault is claimed by one indexer at a time: the claim that begins an indexing
// reports it is underway, a claim while it runs reports so and begins nothing, a
// failed indexing returns why and begins nothing, and a vault indexed cleanly
// begins nothing and is not underway.
func TestAVaultIsClaimedForOneIndexingAtATime(t *testing.T) {
	r := &indexing.Runs{}
	r.SetIndexer(t.Context(), func(context.Context, domain.Vault) error { return nil })
	v := domain.Vault{ID: "one"}

	read, limit, underway, reason := r.Claim(v)
	if read == nil || limit == nil || !underway || reason != "" {
		t.Fatalf("the first claim came back %v %v %v %q", read != nil, limit != nil, underway, reason)
	}

	read, _, underway, reason = r.Claim(v)
	if read != nil || !underway || reason != "" {
		t.Errorf("a claim while it ran came back %v %v %q", read != nil, underway, reason)
	}

	r.Record(v, errors.New("the folder is not there"))
	read, _, underway, reason = r.Claim(v)
	if read != nil || underway || reason != "the folder is not there" {
		t.Errorf("a claim after a failure came back %v %v %q", read != nil, underway, reason)
	}

	r.Clear(v.ID)
	r.Record(v, nil)
	read, _, underway, reason = r.Claim(v)
	if read != nil || underway || reason != "" {
		t.Errorf("a claim after a clean indexing came back %v %v %q", read != nil, underway, reason)
	}
}

func TestAWindowThatReadsNothingClaimsNothing(t *testing.T) {
	r := &indexing.Runs{}
	read, _, underway, reason := r.Claim(domain.Vault{ID: "one"})
	if read != nil || underway || reason != "" {
		t.Errorf("the claim came back %v %v %q", read != nil, underway, reason)
	}
}
