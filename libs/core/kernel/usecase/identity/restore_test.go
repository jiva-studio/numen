package identity_test

import (
	"bytes"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

const restoreFallback = 30 * time.Second

func getOtherWords() []string {
	return strings.Fields("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about")
}

type restoreKit struct {
	deps     identity.Deps
	keys     *fakeKeyStore
	clock    *testsupport.ManualClock
	get      *identity.GetIdentity
	create   *identity.SeededCreate
	public   *identity.GetPublicIdentity
	start    *identity.StartRestore
	pending  *identity.GetPendingRestore
	sign     *identity.SignAsRecovery
	complete *identity.CompleteRestore
	adopt    *identity.AdoptChain
	devices  *identity.ListDevices
	saveKit  *identity.SaveRecoveryKit
	saver    *fakeKitSaver
	startKit *identity.StartRestoreFromKit
	seal     *identity.SealData
	open     *identity.OpenData
}

func newRestoreKit(tb testing.TB, name string) restoreKit {
	tb.Helper()
	keys := newFakeKeyStore()
	clock := testsupport.NewClockAt(getNow())
	deps := identity.NewDeps(keys, fakeNames{name: name}, clock).WithChains(&fakeChainStore{})
	saver := &fakeKitSaver{}
	return restoreKit{
		deps: deps, keys: keys, clock: clock, saver: saver,
		get:      identity.NewGetIdentity(deps),
		create:   identity.NewSeededCreate(deps),
		public:   identity.NewGetPublicIdentity(deps),
		start:    identity.NewStartRestore(deps),
		pending:  identity.NewGetPendingRestore(deps),
		sign:     identity.NewSignAsRecovery(deps),
		complete: identity.NewCompleteRestore(deps),
		adopt:    identity.NewAdoptChain(deps),
		devices:  identity.NewListDevices(deps),
		saveKit:  identity.NewSaveRecoveryKit(deps, saver),
		startKit: identity.NewStartRestoreFromKit(deps),
		seal:     identity.NewSealData(deps),
		open:     identity.NewOpenData(deps),
	}
}

func (k restoreKit) getChain(tb testing.TB) []byte {
	tb.Helper()
	public, err := k.public.Execute(tb.Context())
	if err != nil {
		tb.Fatal(err)
	}
	return public.Chain
}

func (k restoreKit) getRootSignKey(tb testing.TB, words []string) []byte {
	tb.Helper()
	parsed, err := identitylib.ParseRecoveryWords(words)
	if err != nil {
		tb.Fatal(err)
	}
	root, err := parsed.DeriveRecoveryKey()
	if err != nil {
		tb.Fatal(err)
	}
	return root.Public().Sign
}

func TestStartRestore_KeepsAPendingRestoreInMemoryAndWritesNothing(t *testing.T) {
	t.Parallel()
	k := newRestoreKit(t, "Phone")

	if err := k.start.Execute(t.Context(), getLegalWords(), "", true); err != nil {
		t.Fatal(err)
	}

	if got := k.keys.snapshot(); len(got) != 0 || k.keys.setCalled != 0 {
		t.Errorf("the key store holds %v after %d writes, want nothing", got, k.keys.setCalled)
	}
	pending, err := k.pending.Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !pending.IsPending || !bytes.Equal(pending.RootSignKey, k.getRootSignKey(t, getLegalWords())) || len(pending.DeviceSignKey) != 32 {
		t.Errorf("pending = %+v, want the root key of the words and a device key of 32 bytes", pending)
	}
	if _, err := k.get.Execute(t.Context()); !errors.Is(err, identitydomain.ErrNoIdentity) {
		t.Errorf("GetIdentity() error = %v, want ErrNoIdentity while the restore is pending", err)
	}
}

func TestGetPendingRestore_ReportsNothingPendingBeforeAStart(t *testing.T) {
	t.Parallel()
	k := newRestoreKit(t, "Phone")

	pending, err := k.pending.Execute(t.Context())

	if err != nil || pending.IsPending || len(pending.RootSignKey) != 0 || len(pending.DeviceSignKey) != 0 {
		t.Errorf("pending = %+v, error = %v, want nothing pending and no keys", pending, err)
	}
}

func TestStartRestore_RefusesWordsThatAreNotValidAndNothingIsPending(t *testing.T) {
	t.Parallel()
	k := newRestoreKit(t, "Phone")

	for name, words := range map[string][]string{
		"eleven words":   getLegalWords()[:11],
		"a wrong word":   append(getLegalWords()[:11:11], "zoo"),
		"a wrong sum":    strings.Fields(strings.Repeat("abandon ", 12)),
		"no word":        nil,
		"thirteen words": append(getLegalWords(), "legal"),
	} {
		err := k.start.Execute(t.Context(), words, "", true)
		if !errors.Is(err, identitydomain.ErrInvalidWords) {
			t.Errorf("%s: error = %v, want ErrInvalidWords", name, err)
		}
		if pending, perr := k.pending.Execute(t.Context()); perr != nil || pending.IsPending {
			t.Errorf("%s: pending = %+v, error = %v, want nothing pending", name, pending, perr)
		}
	}
}

func TestStartRestore_RefusesWhenTheDeviceHasAnIdentityAndItStaysAsItWas(t *testing.T) {
	t.Parallel()
	k := newRestoreKit(t, "Phone")
	if _, err := k.create.Execute(t.Context(), getLegalWords()); err != nil {
		t.Fatal(err)
	}
	before := k.keys.snapshot()

	err := k.start.Execute(t.Context(), getOtherWords(), "", true)

	if !errors.Is(err, identitydomain.ErrIdentityExists) {
		t.Errorf("error = %v, want ErrIdentityExists", err)
	}
	if pending, perr := k.pending.Execute(t.Context()); perr != nil || pending.IsPending {
		t.Errorf("pending = %+v, error = %v, want nothing pending", pending, perr)
	}
	if after := k.keys.snapshot(); after[keyName] != before[keyName] {
		t.Error("the identity changed")
	}
}

func TestStartRestore_ReplacesAnEarlierPendingRestore(t *testing.T) {
	t.Parallel()
	k := newRestoreKit(t, "Phone")
	if err := k.start.Execute(t.Context(), getLegalWords(), "", true); err != nil {
		t.Fatal(err)
	}
	first, err := k.pending.Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if err := k.start.Execute(t.Context(), getOtherWords(), "", true); err != nil {
		t.Fatal(err)
	}

	second, err := k.pending.Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(second.RootSignKey, k.getRootSignKey(t, getOtherWords())) || bytes.Equal(second.RootSignKey, first.RootSignKey) ||
		bytes.Equal(second.DeviceSignKey, first.DeviceSignKey) {
		t.Errorf("first = %+v, second = %+v, want the root of the second words and a new device key", first, second)
	}
}

func TestSignAsRecovery_RefusesWhenNothingIsPending(t *testing.T) {
	t.Parallel()
	k := newRestoreKit(t, "Phone")

	signature, err := k.sign.Execute(t.Context(), syncPlugin, []byte("message"))

	if !errors.Is(err, identitydomain.ErrNoPendingRestore) || signature != nil {
		t.Errorf("signature = %x, error = %v, want nil and ErrNoPendingRestore", signature, err)
	}
}

func TestSignAsRecovery_SignsForThePluginWithTheRecoveryKeyOfThePendingWords(t *testing.T) {
	t.Parallel()
	k := newRestoreKit(t, "Phone")
	if err := k.start.Execute(t.Context(), getLegalWords(), "", true); err != nil {
		t.Fatal(err)
	}
	root := k.getRootSignKey(t, getLegalWords())

	signature, err := k.sign.Execute(t.Context(), syncPlugin, []byte("message"))
	if err != nil {
		t.Fatal(err)
	}

	if len(signature) != 64 || !identitylib.VerifyForRecovery(root, syncPlugin, []byte("message"), [64]byte(signature)) {
		t.Error("the signature does not verify for the plugin under the recovery key of the words")
	}
	if identitylib.VerifyForRecovery(root, "md.numen.tools.link", []byte("message"), [64]byte(signature)) {
		t.Error("the signature verifies for another plugin")
	}
	if identitylib.VerifyForPlugin(root, syncPlugin, []byte("message"), [64]byte(signature)) {
		t.Error("the signature verifies as the signature of a device for a plugin")
	}
	if identitylib.VerifyForRecovery(k.getRootSignKey(t, getOtherWords()), syncPlugin, []byte("message"), [64]byte(signature)) {
		t.Error("the signature verifies under the key of other words")
	}
}

func TestSignAsRecovery_SignsNothingForAnEmptyPluginID(t *testing.T) {
	t.Parallel()
	k := newRestoreKit(t, "Phone")
	if err := k.start.Execute(t.Context(), getLegalWords(), "", true); err != nil {
		t.Fatal(err)
	}

	if _, err := k.sign.Execute(t.Context(), "", []byte("message")); err == nil {
		t.Error("a signature was made with no plugin id")
	}
}

func TestCompleteRestore_JoinsTheAccountOfAnotherDeviceAndKeepsItsKeys(t *testing.T) {
	t.Parallel()
	first := newRestoreKit(t, "Desktop")
	created, err := first.create.Execute(t.Context(), getLegalWords())
	if err != nil {
		t.Fatal(err)
	}
	second := newRestoreKit(t, "Phone")
	if err := second.start.Execute(t.Context(), getLegalWords(), "", true); err != nil {
		t.Fatal(err)
	}
	pending, err := second.pending.Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if err := second.complete.Execute(t.Context(), syncPlugin, first.getChain(t)); err != nil {
		t.Fatal(err)
	}

	got, err := second.get.Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if got.AccountFingerprint != created.AccountFingerprint || got.DeviceFingerprint == created.DeviceFingerprint || got.DeviceName != "Phone" {
		t.Errorf("identity = %+v, want the account of the first device, a device of its own, named Phone", got)
	}
	if second.keys.setCalled != 2 {
		t.Errorf("the record and the chain length were written %d times, want twice", second.keys.setCalled)
	}
	chain, err := identitylib.ParseChain(second.getChain(t))
	if err != nil {
		t.Fatalf("the chain of the restored device does not verify: %v", err)
	}
	firstChain, err := identitylib.ParseChain(first.getChain(t))
	if err != nil {
		t.Fatal(err)
	}
	devices := chain.ListDevices()
	if !chain.IsExtensionOf(firstChain) || len(devices) != 1 || !bytes.Equal(devices[0].Public.Sign, pending.DeviceSignKey) {
		t.Errorf("devices = %d, want the chain of the first device extended by the pending device, which is the only active one", len(devices))
	}
	if after, err := second.pending.Execute(t.Context()); err != nil || after.IsPending {
		t.Errorf("pending = %+v, error = %v, want nothing pending", after, err)
	}
}

func TestCompleteRestore_TheRestoredDeviceOpensWhatTheRevokedDeviceSealedAndNotTheOtherWayRound(t *testing.T) {
	t.Parallel()
	first := newRestoreKit(t, "Desktop")
	if _, err := first.create.Execute(t.Context(), getLegalWords()); err != nil {
		t.Fatal(err)
	}
	second := newRestoreKit(t, "Phone")
	if err := second.start.Execute(t.Context(), getLegalWords(), "", true); err != nil {
		t.Fatal(err)
	}
	if err := second.complete.Execute(t.Context(), syncPlugin, first.getChain(t)); err != nil {
		t.Fatal(err)
	}
	if err := first.adopt.Execute(t.Context(), second.getChain(t)); err != nil {
		t.Fatal(err)
	}

	sealedByFirst, err := first.seal.Execute(t.Context(), syncPlugin, []byte(sealContext), []byte("catalog of the first"))
	if err != nil {
		t.Fatal(err)
	}
	sealedBySecond, err := second.seal.Execute(t.Context(), syncPlugin, []byte(sealContext), []byte("catalog of the second"))
	if err != nil {
		t.Fatal(err)
	}

	openedBySecond, err := second.open.Execute(t.Context(), syncPlugin, []byte(sealContext), sealedByFirst)
	if err != nil || string(openedBySecond) != "catalog of the first" {
		t.Errorf("the second device opened %q with error %v, want the text of the first", openedBySecond, err)
	}
	if openedByFirst, err := first.open.Execute(t.Context(), syncPlugin, []byte(sealContext), sealedBySecond); err == nil {
		t.Errorf("the revoked first device opened %q of the second", openedByFirst)
	}
}

func TestCompleteRestore_WithNoChainMakesANewIdentityOfTheWords(t *testing.T) {
	t.Parallel()
	other := newKit("Desktop")
	expected, err := other.create.Execute(t.Context(), getLegalWords())
	if err != nil {
		t.Fatal(err)
	}
	k := newRestoreKit(t, "Phone")
	if err := k.start.Execute(t.Context(), getLegalWords(), "", true); err != nil {
		t.Fatal(err)
	}

	if err := k.complete.Execute(t.Context(), syncPlugin, nil); err != nil {
		t.Fatal(err)
	}

	got, err := k.get.Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if got.AccountFingerprint != expected.AccountFingerprint || k.keys.setCalled != 1 {
		t.Errorf("identity = %+v after %d writes, want the account of the words from one write", got, k.keys.setCalled)
	}
	chain, err := identitylib.ParseChain(k.getChain(t))
	if err != nil || len(chain.ListDevices()) != 1 {
		t.Errorf("chain = %v with error %v, want a genesis and one device", chain.ListDevices(), err)
	}
	sealed, err := k.seal.Execute(t.Context(), syncPlugin, []byte(sealContext), []byte("milk"))
	if err != nil {
		t.Fatal(err)
	}
	if opened, err := k.open.Execute(t.Context(), syncPlugin, []byte(sealContext), sealed); err != nil || string(opened) != "milk" {
		t.Errorf("opened = %q, error = %v, want milk", opened, err)
	}
	if after, err := k.pending.Execute(t.Context()); err != nil || after.IsPending {
		t.Errorf("pending = %+v, error = %v, want nothing pending", after, err)
	}
}

func TestCompleteRestore_RefusesWhenNothingIsPending(t *testing.T) {
	t.Parallel()
	first := newRestoreKit(t, "Desktop")
	if _, err := first.create.Execute(t.Context(), getLegalWords()); err != nil {
		t.Fatal(err)
	}
	k := newRestoreKit(t, "Phone")

	err := k.complete.Execute(t.Context(), syncPlugin, first.getChain(t))

	if !errors.Is(err, identitydomain.ErrNoPendingRestore) || len(k.keys.snapshot()) != 0 {
		t.Errorf("error = %v with %v in the key store, want ErrNoPendingRestore and nothing written", err, k.keys.snapshot())
	}
}

func TestCompleteRestore_RefusesAChainOfAnotherRootAndWritesNothing(t *testing.T) {
	t.Parallel()
	stranger := newRestoreKit(t, "Stranger")
	if _, err := stranger.create.Execute(t.Context(), getOtherWords()); err != nil {
		t.Fatal(err)
	}
	k := newRestoreKit(t, "Phone")
	if err := k.start.Execute(t.Context(), getLegalWords(), "", true); err != nil {
		t.Fatal(err)
	}

	err := k.complete.Execute(t.Context(), syncPlugin, stranger.getChain(t))

	if !errors.Is(err, identitydomain.ErrInvalidChain) || len(k.keys.snapshot()) != 0 {
		t.Errorf("error = %v with %v in the key store, want ErrInvalidChain and nothing written", err, k.keys.snapshot())
	}
}

func TestCompleteRestore_RefusesABrokenChainAndWritesNothing(t *testing.T) {
	t.Parallel()
	first := newRestoreKit(t, "Desktop")
	if _, err := first.create.Execute(t.Context(), getLegalWords()); err != nil {
		t.Fatal(err)
	}
	valid := first.getChain(t)
	changed := bytes.Clone(valid)
	changed[len(changed)/2] ^= 1

	for name, chain := range map[string][]byte{
		"bytes that are no chain": []byte("not a chain"),
		"a cut chain":             valid[:len(valid)-1],
		"a changed byte":          changed,
	} {
		k := newRestoreKit(t, "Phone")
		if err := k.start.Execute(t.Context(), getLegalWords(), "", true); err != nil {
			t.Fatal(err)
		}

		err := k.complete.Execute(t.Context(), syncPlugin, chain)

		if !errors.Is(err, identitydomain.ErrInvalidChain) || len(k.keys.snapshot()) != 0 {
			t.Errorf("%s: error = %v with %v in the key store, want ErrInvalidChain and nothing written", name, err, k.keys.snapshot())
		}
	}
}

func TestCompleteRestore_ReportsAKeyStoreThatRefusesAndWritesNothing(t *testing.T) {
	t.Parallel()
	first := newRestoreKit(t, "Desktop")
	if _, err := first.create.Execute(t.Context(), getLegalWords()); err != nil {
		t.Fatal(err)
	}
	k := newRestoreKit(t, "Phone")
	if err := k.start.Execute(t.Context(), getLegalWords(), "", true); err != nil {
		t.Fatal(err)
	}
	k.keys.setFails = errBroken

	err := k.complete.Execute(t.Context(), syncPlugin, first.getChain(t))

	if !errors.Is(err, identitydomain.ErrKeyStore) || !errors.Is(err, errBroken) || len(k.keys.snapshot()) != 0 {
		t.Errorf("error = %v with %v in the key store, want ErrKeyStore wrapping the failure and nothing written", err, k.keys.snapshot())
	}
}

func TestCompleteRestore_NeverOverwritesAnIdentityMadeWhileTheRestoreWasPending(t *testing.T) {
	t.Parallel()
	k := newRestoreKit(t, "Phone")
	if err := k.start.Execute(t.Context(), getLegalWords(), "", true); err != nil {
		t.Fatal(err)
	}
	if _, err := k.create.Execute(t.Context(), getOtherWords()); err != nil {
		t.Fatal(err)
	}
	before := k.keys.snapshot()

	err := k.complete.Execute(t.Context(), syncPlugin, nil)

	if !errors.Is(err, identitydomain.ErrIdentityExists) || k.keys.snapshot()[keyName] != before[keyName] {
		t.Errorf("error = %v, want ErrIdentityExists and the identity as it was", err)
	}
}

func TestCompleteRestore_ClearsThePendingRestoreSoThatSigningIsRefusedAgain(t *testing.T) {
	t.Parallel()
	k := newRestoreKit(t, "Phone")
	if err := k.start.Execute(t.Context(), getLegalWords(), "", true); err != nil {
		t.Fatal(err)
	}
	if err := k.complete.Execute(t.Context(), syncPlugin, nil); err != nil {
		t.Fatal(err)
	}

	if _, err := k.sign.Execute(t.Context(), syncPlugin, []byte("message")); !errors.Is(err, identitydomain.ErrNoPendingRestore) {
		t.Errorf("SignAsRecovery() error = %v, want ErrNoPendingRestore", err)
	}
	if err := k.complete.Execute(t.Context(), syncPlugin, nil); !errors.Is(err, identitydomain.ErrNoPendingRestore) {
		t.Errorf("a second CompleteRestore() error = %v, want ErrNoPendingRestore", err)
	}
}

func TestStartRestore_CompletesAsAnOfflineRestoreWhenNoPluginCompletesItIn30Seconds(t *testing.T) {
	t.Parallel()
	expected, err := newKit("Desktop").create.Execute(t.Context(), getLegalWords())
	if err != nil {
		t.Fatal(err)
	}
	k := newRestoreKit(t, "Phone")
	if err := k.start.Execute(t.Context(), getLegalWords(), "", true); err != nil {
		t.Fatal(err)
	}

	k.clock.Advance(restoreFallback - time.Second)
	if pending, err := k.pending.Execute(t.Context()); err != nil || !pending.IsPending || len(k.keys.snapshot()) != 0 {
		t.Fatalf("after 29 seconds: pending = %+v, error = %v, keys = %v, want the restore still pending and nothing written", pending, err, k.keys.snapshot())
	}
	k.clock.Advance(time.Second)

	testsupport.WaitFor(t, func() bool {
		got, err := k.get.Execute(t.Context())
		return err == nil && got.AccountFingerprint == expected.AccountFingerprint
	})
	if pending, err := k.pending.Execute(t.Context()); err != nil || pending.IsPending {
		t.Errorf("pending = %+v, error = %v, want nothing pending after the fallback", pending, err)
	}
	if k.keys.setCalled != 1 {
		t.Errorf("the record was written %d times, want once", k.keys.setCalled)
	}
}

func TestStartRestore_ADeadlineThatPassesAfterTheRestoreCompletedChangesNothing(t *testing.T) {
	t.Parallel()
	k := newRestoreKit(t, "Phone")
	if err := k.start.Execute(t.Context(), getLegalWords(), "", true); err != nil {
		t.Fatal(err)
	}
	if err := k.complete.Execute(t.Context(), syncPlugin, nil); err != nil {
		t.Fatal(err)
	}
	before := k.keys.snapshot()

	k.clock.Advance(2 * restoreFallback)

	if k.keys.setCalled != 1 || k.keys.snapshot()[keyName] != before[keyName] {
		t.Errorf("the record was written %d times and changed, want the one write of CompleteRestore", k.keys.setCalled)
	}
}

func TestStartRestore_ANewStartMovesTheDeadlineOfTheFallback(t *testing.T) {
	t.Parallel()
	expected, err := newKit("Desktop").create.Execute(t.Context(), getOtherWords())
	if err != nil {
		t.Fatal(err)
	}
	k := newRestoreKit(t, "Phone")
	if err := k.start.Execute(t.Context(), getLegalWords(), "", true); err != nil {
		t.Fatal(err)
	}
	k.clock.Advance(20 * time.Second)
	if err := k.start.Execute(t.Context(), getOtherWords(), "", true); err != nil {
		t.Fatal(err)
	}

	k.clock.Advance(15 * time.Second)
	if pending, err := k.pending.Execute(t.Context()); err != nil || !pending.IsPending || len(k.keys.snapshot()) != 0 {
		t.Fatalf("35 seconds after the first start: pending = %+v, error = %v, keys = %v, want the second restore pending and nothing written", pending, err, k.keys.snapshot())
	}
	k.clock.Advance(15 * time.Second)

	testsupport.WaitFor(t, func() bool {
		got, err := k.get.Execute(t.Context())
		return err == nil && got.AccountFingerprint == expected.AccountFingerprint
	})
}

func (k restoreKit) kitOf(tb testing.TB, words []string) identitydomain.Kit {
	tb.Helper()
	if isSaved, err := k.saveKit.Execute(tb.Context(), words, nil); err != nil || !isSaved {
		tb.Fatalf("save recovery kit: %v, %v", isSaved, err)
	}
	made, ok := k.saver.getLast()
	if !ok {
		tb.Fatal("no kit reached the saver")
	}
	return made
}

func TestStartRestore_DropsAPendingRestoreThatStaysUncompletedFor10Minutes(t *testing.T) {
	t.Parallel()
	k := newRestoreKit(t, "Phone")
	k.keys.setFails = errors.New("key store is locked")
	if err := k.start.Execute(t.Context(), getLegalWords(), "", true); err != nil {
		t.Fatal(err)
	}
	k.clock.Advance(restoreFallback)
	if pending, err := k.pending.Execute(t.Context()); err != nil || !pending.IsPending {
		t.Fatalf("pending = %+v, error = %v, want the restore to stay pending after a failed completion", pending, err)
	}

	areKeysZeroed := identity.GetPendingKeyCheck(k.deps)
	if areKeysZeroed() {
		t.Fatal("keys of the pending restore are zero before the expiry")
	}

	k.clock.Advance(10 * time.Minute)

	testsupport.WaitFor(t, func() bool {
		pending, err := k.pending.Execute(t.Context())
		return err == nil && !pending.IsPending
	})
	if !areKeysZeroed() {
		t.Error("keys of the expired restore are not zeroed")
	}
	if _, err := k.sign.Execute(t.Context(), syncPlugin, []byte("message")); !errors.Is(err, identitydomain.ErrNoPendingRestore) {
		t.Errorf("SignAsRecovery after the expiry: error = %v, want ErrNoPendingRestore", err)
	}
}

func TestCompleteRestore_ConcurrentCreateResultsInOneIdentity(t *testing.T) {
	t.Parallel()
	k := newRestoreKit(t, "Phone")
	other := identity.NewSeededCreate(k.deps)
	if err := k.start.Execute(t.Context(), getLegalWords(), "", true); err != nil {
		t.Fatal(err)
	}
	var arrived atomic.Int32
	both := make(chan struct{})
	var once sync.Once
	k.keys.onGet = func() {
		if arrived.Add(1) >= 2 {
			once.Do(func() { close(both) })
		}
		select {
		case <-both:
		case <-time.After(300 * time.Millisecond):
		}
	}
	errs := make([]error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		errs[0] = k.complete.Execute(t.Context(), syncPlugin, nil)
	}()
	go func() {
		defer wg.Done()
		_, errs[1] = other.Execute(t.Context(), getOtherWords())
	}()
	wg.Wait()

	succeeded, refused := 0, 0
	for _, err := range errs {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, identitydomain.ErrIdentityExists):
			refused++
		}
	}
	if succeeded != 1 || refused != 1 || k.keys.setCalled != 1 {
		t.Errorf("succeeded = %d, refused = %d, writes = %d, want 1, 1, 1", succeeded, refused, k.keys.setCalled)
	}
}

func TestStartRestore_TriesTheFallbackAgainWhenTheKeyStoreRefusedIt(t *testing.T) {
	t.Parallel()
	expected, err := newKit("Desktop").create.Execute(t.Context(), getLegalWords())
	if err != nil {
		t.Fatal(err)
	}
	k := newRestoreKit(t, "Phone")
	if err := k.start.Execute(t.Context(), getLegalWords(), "", true); err != nil {
		t.Fatal(err)
	}
	k.keys.failSets(errBroken)
	k.clock.Advance(restoreFallback)
	testsupport.WaitFor(t, func() bool {
		pending, err := k.pending.Execute(t.Context())
		return err == nil && pending.IsPending && k.keys.writes() == 1
	})
	k.keys.failSets(nil)

	testsupport.WaitFor(t, func() bool {
		k.clock.Advance(restoreFallback)
		got, err := k.get.Execute(t.Context())
		return err == nil && got.AccountFingerprint == expected.AccountFingerprint
	})
}
