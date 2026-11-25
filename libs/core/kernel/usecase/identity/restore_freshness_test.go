package identity_test

import (
	"errors"
	"strings"
	"testing"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

func kitOfRestoredPhone(t *testing.T) (older, current []byte, kit identitydomain.Kit) {
	t.Helper()
	desktop := newRestoreKit(t, "Desktop")
	if _, err := desktop.create.Execute(t.Context(), getLegalWords()); err != nil {
		t.Fatal(err)
	}
	older = desktop.getChain(t)
	phone := newRestoreKit(t, "Phone")
	if err := phone.start.Execute(t.Context(), getLegalWords(), "", true); err != nil {
		t.Fatal(err)
	}
	if err := phone.complete.Execute(t.Context(), syncPlugin, older); err != nil {
		t.Fatal(err)
	}
	return older, phone.getChain(t), phone.kitOf(t, getLegalWords())
}

func TestSaveRecoveryKit_LengthAndHeadOfTheDeviceChain(t *testing.T) {
	t.Parallel()
	_, current, made := kitOfRestoredPhone(t)

	chain, err := identitylib.ParseChain(current)
	if err != nil {
		t.Fatal(err)
	}
	if made.ChainMark != chain.GetMark().String() {
		t.Errorf("kit chain mark = %q, want %q", made.ChainMark, chain.GetMark().String())
	}
}

func TestCompleteRestore_RefusesAChainThatIsOlderThanTheChainMarkOfTheKit(t *testing.T) {
	t.Parallel()
	older, _, made := kitOfRestoredPhone(t)
	fresh := newRestoreKit(t, "Tablet")
	if err := fresh.startKit.Execute(t.Context(), pick(t, made, 1, 2), made.ChainMark, false); err != nil {
		t.Fatal(err)
	}

	err := fresh.complete.Execute(t.Context(), syncPlugin, older)

	if !errors.Is(err, identitydomain.ErrChainOutdated) {
		t.Fatalf("error = %v, want ErrChainOutdated", err)
	}
	if got := fresh.keys.snapshot(); len(got) != 0 {
		t.Errorf("the key store holds %v after a refused restore", got)
	}
	if _, err := fresh.get.Execute(t.Context()); !errors.Is(err, identitydomain.ErrChainOutdated) {
		t.Errorf("GetIdentity error = %v, want the refusal of the restore", err)
	}
	if pending, err := fresh.pending.Execute(t.Context()); err != nil || pending.IsPending {
		t.Errorf("pending = %+v, error = %v, want nothing pending after a refusal", pending, err)
	}
}

func TestCompleteRestore_RefusesNoChainWhenTheKitNamesOne(t *testing.T) {
	t.Parallel()
	_, _, made := kitOfRestoredPhone(t)
	fresh := newRestoreKit(t, "Tablet")
	if err := fresh.start.Execute(t.Context(), getLegalWords(), made.ChainMark, false); err != nil {
		t.Fatal(err)
	}

	if err := fresh.complete.Execute(t.Context(), syncPlugin, nil); !errors.Is(err, identitydomain.ErrChainOutdated) {
		t.Errorf("error = %v, want ErrChainOutdated", err)
	}
	if got := fresh.keys.snapshot(); len(got) != 0 {
		t.Errorf("the key store holds %v after a refused restore", got)
	}
}

func TestCompleteRestore_RefusesAChainOfAnotherHistoryThatIsAsLongAsTheMark(t *testing.T) {
	t.Parallel()
	_, _, made := kitOfRestoredPhone(t)
	stranger := newRestoreKit(t, "Desktop")
	if _, err := stranger.create.Execute(t.Context(), getLegalWords()); err != nil {
		t.Fatal(err)
	}
	other := newRestoreKit(t, "Phone")
	if err := other.start.Execute(t.Context(), getLegalWords(), "", true); err != nil {
		t.Fatal(err)
	}
	if err := other.complete.Execute(t.Context(), syncPlugin, stranger.getChain(t)); err != nil {
		t.Fatal(err)
	}
	fresh := newRestoreKit(t, "Tablet")
	if err := fresh.start.Execute(t.Context(), getLegalWords(), made.ChainMark, false); err != nil {
		t.Fatal(err)
	}

	if err := fresh.complete.Execute(t.Context(), syncPlugin, other.getChain(t)); !errors.Is(err, identitydomain.ErrChainOutdated) {
		t.Errorf("error = %v, want ErrChainOutdated", err)
	}
}

func TestCompleteRestore_AcceptsTheChainOfTheMarkAndALongerOne(t *testing.T) {
	t.Parallel()
	_, current, made := kitOfRestoredPhone(t)
	fresh := newRestoreKit(t, "Tablet")
	if err := fresh.startKit.Execute(t.Context(), pick(t, made, 2, 3), made.ChainMark, false); err != nil {
		t.Fatal(err)
	}

	if err := fresh.complete.Execute(t.Context(), syncPlugin, current); err != nil {
		t.Fatalf("error = %v, want the restore to complete", err)
	}
	if _, err := fresh.get.Execute(t.Context()); err != nil {
		t.Errorf("GetIdentity error = %v", err)
	}
}

func TestStartRestore_RefusesAChainMarkThatIsNotAMark(t *testing.T) {
	t.Parallel()
	k := newRestoreKit(t, "Phone")

	err := k.start.Execute(t.Context(), getLegalWords(), "seven-ABC", false)

	if !errors.Is(err, identitydomain.ErrInvalidChainMark) {
		t.Errorf("error = %v, want ErrInvalidChainMark", err)
	}
	if pending, err := k.pending.Execute(t.Context()); err != nil || pending.IsPending {
		t.Errorf("pending = %+v, error = %v, want nothing pending", pending, err)
	}
}

func TestCompleteRestore_RotatesTheKeysSoTheKeysWrappedForTheWordsAreNotInUse(t *testing.T) {
	t.Parallel()
	desktop := newRestoreKit(t, "Desktop")
	if _, err := desktop.create.Execute(t.Context(), getLegalWords()); err != nil {
		t.Fatal(err)
	}
	before, err := identitylib.ParseChain(desktop.getChain(t))
	if err != nil {
		t.Fatal(err)
	}
	phone := newRestoreKit(t, "Phone")
	if err := phone.start.Execute(t.Context(), getLegalWords(), "", true); err != nil {
		t.Fatal(err)
	}

	if err := phone.complete.Execute(t.Context(), syncPlugin, desktop.getChain(t)); err != nil {
		t.Fatal(err)
	}

	after, err := identitylib.ParseChain(phone.getChain(t))
	if err != nil {
		t.Fatal(err)
	}
	if after.Epoch() != before.Epoch()+1 || after.AccountBoxKey() == before.AccountBoxKey() {
		t.Errorf("epoch = %d, want %d with new account keys", after.Epoch(), before.Epoch()+1)
	}
	devices, err := phone.devices.Execute(t.Context())
	if err != nil || len(devices) != 2 {
		t.Errorf("devices = %+v, error = %v, want both devices listed for the person to check", devices, err)
	}
}

func TestCompleteRestore_RefusesNoChainWhenThisDeviceHeldTheChainOfASingleDeviceIdentity(t *testing.T) {
	t.Parallel()
	k := newRestoreKit(t, "Desktop")
	if _, err := k.create.Execute(t.Context(), getLegalWords()); err != nil {
		t.Fatal(err)
	}
	for name := range k.keys.snapshot() {
		if !strings.HasPrefix(name, "chain-length-") {
			if err := k.keys.DeleteKey(t.Context(), name); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := k.start.Execute(t.Context(), getLegalWords(), "", true); err != nil {
		t.Fatal(err)
	}

	if err := k.complete.Execute(t.Context(), syncPlugin, nil); !errors.Is(err, identitydomain.ErrChainOutdated) {
		t.Errorf("error = %v, want ErrChainOutdated", err)
	}
}

func TestSignAsRecovery_RefusesAfterTheRestoreWasRefused(t *testing.T) {
	t.Parallel()
	_, _, made := kitOfRestoredPhone(t)
	fresh := newRestoreKit(t, "Tablet")
	if err := fresh.start.Execute(t.Context(), getLegalWords(), made.ChainMark, false); err != nil {
		t.Fatal(err)
	}
	if err := fresh.complete.Execute(t.Context(), syncPlugin, nil); !errors.Is(err, identitydomain.ErrChainOutdated) {
		t.Fatalf("error = %v, want ErrChainOutdated", err)
	}

	signature, err := fresh.sign.Execute(t.Context(), syncPlugin, []byte("message"))

	if !errors.Is(err, identitydomain.ErrChainOutdated) || signature != nil {
		t.Errorf("signature = %x, error = %v, want nil and ErrChainOutdated", signature, err)
	}
}
