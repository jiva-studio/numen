package identity_test

import (
	"sync/atomic"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/identity"
)

type presenceKit struct {
	account *identity.Account
	create  *identity.SeededCreate
	delete  *identity.DeleteIdentity
	start   *identity.StartRestore
	changes *atomic.Int32
}

func newPresenceKit() presenceKit {
	keys, chains, progress := newFakeKeyStore(), &fakeChainStore{}, identity.NewProgress()
	clock := testsupport.NewClockAt(getNow())
	deps := identity.NewDeps(keys, fakeNames{name: "Laptop"}, clock).WithChains(chains).WithProgress(progress)
	account := identity.NewAccount(func() pluginport.KeyStore { return keys }, fakeNames{name: "Laptop"}, clock, progress, chains)
	changes := &atomic.Int32{}
	account.OnChange(func() { changes.Add(1) })
	return presenceKit{
		account: account, create: identity.NewSeededCreate(deps), delete: identity.NewDeleteIdentity(deps),
		start: identity.NewStartRestore(deps), changes: changes,
	}
}

func TestIsPresent_IsFalseWhileTheDeviceHasNoIdentity(t *testing.T) {
	t.Parallel()
	k := newPresenceKit()

	if k.account.IsPresent(t.Context()) {
		t.Error("the device with no identity is present")
	}
}

func TestIsPresent_FollowsTheCreationAndTheDeletionAndTellsEachChange(t *testing.T) {
	t.Parallel()
	k := newPresenceKit()

	if _, err := k.create.Execute(t.Context(), getLegalWords()); err != nil {
		t.Fatal(err)
	}
	created, afterCreate := k.account.IsPresent(t.Context()), k.changes.Load()
	if err := k.delete.Execute(t.Context()); err != nil {
		t.Fatal(err)
	}

	if !created || afterCreate == 0 {
		t.Errorf("after the creation present = %v and %d changes told, want present and some", created, afterCreate)
	}
	if k.account.IsPresent(t.Context()) || k.changes.Load() <= afterCreate {
		t.Errorf("after the deletion present = %v and %d changes told, want absent and more than %d", k.account.IsPresent(t.Context()), k.changes.Load(), afterCreate)
	}
}

func TestIsPresent_TrueWhileRestoreAwaitsServerChain(t *testing.T) {
	t.Parallel()
	k := newPresenceKit()

	if err := k.start.Execute(t.Context(), getLegalWords(), "", true); err != nil {
		t.Fatal(err)
	}

	if !k.account.IsPresent(t.Context()) || k.changes.Load() == 0 {
		t.Errorf("during the restore present = %v and %d changes told, want present and some", k.account.IsPresent(t.Context()), k.changes.Load())
	}
}
