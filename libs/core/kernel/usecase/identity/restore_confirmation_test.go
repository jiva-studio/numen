package identity_test

import (
	"errors"
	"testing"
	"time"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/identity"
)

func TestStartRestore_RefusesARestoreWithoutAChainMarkAndWithoutConfirmation(t *testing.T) {
	t.Parallel()
	k := newRestoreKit(t, "Phone")

	err := k.start.Execute(t.Context(), getLegalWords(), "", false)

	if !errors.Is(err, identitydomain.ErrInvalidChainMark) {
		t.Fatalf("error = %v, want ErrInvalidChainMark", err)
	}
	if pending, err := k.pending.Execute(t.Context()); err != nil || pending.IsPending {
		t.Errorf("pending = %+v, error = %v, want nothing pending", pending, err)
	}
}

func TestStartRestoreFromKit_RefusesARestoreWithoutAChainMarkAndWithoutConfirmation(t *testing.T) {
	t.Parallel()
	_, _, made := kitOfRestoredPhone(t)
	fresh := newRestoreKit(t, "Tablet")

	err := fresh.startKit.Execute(t.Context(), pick(t, made, 1, 2), "", false)

	if !errors.Is(err, identitydomain.ErrInvalidChainMark) {
		t.Fatalf("error = %v, want ErrInvalidChainMark", err)
	}
}

func TestStartRestore_ASilentServerWithAChainMarkCreatesNoIdentity(t *testing.T) {
	t.Parallel()
	_, _, made := kitOfRestoredPhone(t)
	k := newRestoreKit(t, "Tablet")
	if err := k.start.Execute(t.Context(), getLegalWords(), made.ChainMark, false); err != nil {
		t.Fatal(err)
	}

	k.clock.Advance(2 * restoreFallback)

	time.Sleep(50 * time.Millisecond)
	if got := k.keys.snapshot(); len(got) != 0 {
		t.Errorf("the key store holds %v after a silent server, want nothing", got)
	}
}

func TestCompleteRestore_RefusesAChainShorterThanTheLongestOfTheAccount(t *testing.T) {
	t.Parallel()
	older, current, _ := kitOfRestoredPhone(t)
	phone := newRestoreKit(t, "Phone")
	if err := phone.start.Execute(t.Context(), getLegalWords(), "", true); err != nil {
		t.Fatal(err)
	}
	if err := phone.complete.Execute(t.Context(), syncPlugin, current); err != nil {
		t.Fatal(err)
	}
	if err := identity.NewDeleteIdentity(phone.deps).Execute(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := phone.start.Execute(t.Context(), getLegalWords(), "", true); err != nil {
		t.Fatal(err)
	}

	err := phone.complete.Execute(t.Context(), syncPlugin, older)

	if !errors.Is(err, identitydomain.ErrChainOutdated) {
		t.Fatalf("error = %v, want ErrChainOutdated", err)
	}
}

func TestAdoptChain_RefusesAChainDatedFarAfterTheClockOfThisDevice(t *testing.T) {
	t.Parallel()
	first := newRestoreKit(t, "Desktop")
	if _, err := first.create.Execute(t.Context(), getLegalWords()); err != nil {
		t.Fatal(err)
	}
	second := newRestoreKit(t, "Phone")
	second.clock.Advance(72 * time.Hour)
	if err := second.start.Execute(t.Context(), getLegalWords(), "", true); err != nil {
		t.Fatal(err)
	}
	if err := second.complete.Execute(t.Context(), syncPlugin, first.getChain(t)); err != nil {
		t.Fatal(err)
	}

	err := first.adopt.Execute(t.Context(), second.getChain(t))

	if !errors.Is(err, identitydomain.ErrInvalidChain) {
		t.Errorf("error = %v, want ErrInvalidChain", err)
	}
}

func TestCompleteRestore_RefusesAChainDatedFarAfterTheClockOfThisDevice(t *testing.T) {
	t.Parallel()
	first := newRestoreKit(t, "Desktop")
	first.clock.Advance(72 * time.Hour)
	if _, err := first.create.Execute(t.Context(), getLegalWords()); err != nil {
		t.Fatal(err)
	}
	second := newRestoreKit(t, "Phone")
	if err := second.start.Execute(t.Context(), getLegalWords(), "", true); err != nil {
		t.Fatal(err)
	}

	err := second.complete.Execute(t.Context(), syncPlugin, first.getChain(t))

	if !errors.Is(err, identitydomain.ErrInvalidChain) {
		t.Errorf("error = %v, want ErrInvalidChain", err)
	}
}
