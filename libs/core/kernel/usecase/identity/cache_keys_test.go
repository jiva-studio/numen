package identity_test

import (
	"errors"
	"sync"
	"testing"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/identity"
)

type cachedKit struct {
	keys     *fakeKeyStore
	public   *identity.GetPublicIdentity
	seal     *identity.SealData
	generate *identity.GenerateRecoveryWords
	create   *identity.SeededCreate
	delete   *identity.DeleteIdentity
}

func newCachedKit() cachedKit {
	keys := newFakeKeyStore()
	deps := identity.NewDeps(identity.CacheKeys(keys), fakeNames{name: "Laptop"}, testsupport.NewClockAt(getNow())).WithChains(&fakeChainStore{})
	return cachedKit{
		keys:     keys,
		public:   identity.NewGetPublicIdentity(deps),
		seal:     identity.NewSealData(deps),
		generate: identity.NewGenerateRecoveryWords(deps),
		create:   identity.NewSeededCreate(deps),
		delete:   identity.NewDeleteIdentity(deps),
	}
}

func TestCacheKeys_ReadsTheRecordOnceForManyCalls(t *testing.T) {
	t.Parallel()
	k := newCachedKit()
	if _, err := k.create.Execute(t.Context(), getLegalWords()); err != nil {
		t.Fatal(err)
	}
	before := k.keys.countGets(keyName)

	for range 20 {
		if _, err := k.public.Execute(t.Context()); err != nil {
			t.Fatal(err)
		}
		if _, err := k.seal.Execute(t.Context(), syncPlugin, []byte(sealContext), []byte("x")); err != nil {
			t.Fatal(err)
		}
	}

	if got := k.keys.countGets(keyName) - before; got > 1 {
		t.Errorf("reads of the record for 40 calls = %d, want at most 1", got)
	}
}

func TestCacheKeys_SeesTheNewRecordAfterDeleteAndCreate(t *testing.T) {
	t.Parallel()
	k := newCachedKit()
	if _, err := k.create.Execute(t.Context(), getLegalWords()); err != nil {
		t.Fatal(err)
	}
	first, err := k.public.Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if err := k.delete.Execute(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := k.public.Execute(t.Context()); !errors.Is(err, identitydomain.ErrNoIdentity) {
		t.Fatalf("after delete: error = %v, want ErrNoIdentity", err)
	}
	words, err := k.generate.Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := k.create.Execute(t.Context(), words); err != nil {
		t.Fatal(err)
	}
	second, err := k.public.Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if second.AccountFingerprint == first.AccountFingerprint {
		t.Error("after create of other words the account is still the old one")
	}
}

func TestCacheKeys_SeesAnIdentityCreatedAfterAnAbsentAnswer(t *testing.T) {
	t.Parallel()
	k := newCachedKit()
	if _, err := k.public.Execute(t.Context()); !errors.Is(err, identitydomain.ErrNoIdentity) {
		t.Fatalf("before create: error = %v, want ErrNoIdentity", err)
	}
	if _, err := k.create.Execute(t.Context(), getLegalWords()); err != nil {
		t.Fatal(err)
	}
	if _, err := k.public.Execute(t.Context()); err != nil {
		t.Errorf("after create: %v", err)
	}
}

func TestCacheKeys_SeesAReplacedRecord(t *testing.T) {
	t.Parallel()
	keys := newFakeKeyStore()
	cached := identity.CacheKeys(keys)
	if err := cached.SetKey(t.Context(), keyName, "one"); err != nil {
		t.Fatal(err)
	}
	if got := mustGetKey(t, cached, keyName); got != "one" {
		t.Fatalf("first read = %q", got)
	}
	if err := cached.SetKey(t.Context(), keyName, "two"); err != nil {
		t.Fatal(err)
	}
	if got := mustGetKey(t, cached, keyName); got != "two" {
		t.Errorf("read after replace = %q, want two", got)
	}
}

func TestCacheKeys_ForgetsTheRecordWhenAWriteFails(t *testing.T) {
	t.Parallel()
	keys := newFakeKeyStore()
	cached := identity.CacheKeys(keys)
	mustSetKey(t, cached, keyName, "one")
	_ = mustGetKey(t, cached, keyName)
	keys.mu.Lock()
	keys.keys[keyName] = "changed behind"
	keys.mu.Unlock()
	keys.setFails = errBroken
	if err := cached.SetKey(t.Context(), keyName, "two"); err == nil {
		t.Fatal("write with a failing key store: error = nil")
	}

	if got := mustGetKey(t, cached, keyName); got != "changed behind" {
		t.Errorf("read after a failed write = %q, want what the store holds", got)
	}
}

func TestCacheKeys_PassesOtherKeysThrough(t *testing.T) {
	t.Parallel()
	keys := newFakeKeyStore()
	cached := identity.CacheKeys(keys)
	mustSetKey(t, cached, "api", "a")
	for range 3 {
		if _, err := cached.GetKey(t.Context(), "api"); err != nil {
			t.Fatal(err)
		}
	}
	if got := keys.countGets("api"); got != 3 {
		t.Errorf("reads of another key = %d, want 3", got)
	}
}

func TestCacheKeys_IsSafeForConcurrentUse(t *testing.T) {
	t.Parallel()
	cached := identity.CacheKeys(newFakeKeyStore())
	mustSetKey(t, cached, keyName, "v")
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Go(func() {
			for range 50 {
				if i%2 == 0 {
					if err := cached.SetKey(t.Context(), keyName, "v"); err != nil {
						t.Error(err)
					}
				} else {
					if _, err := cached.GetKey(t.Context(), keyName); err != nil {
						t.Error(err)
					}
				}
			}
		})
	}
	wg.Wait()
}

func mustGetKey(tb testing.TB, store pluginport.KeyStore, name string) string {
	tb.Helper()
	value, err := store.GetKey(tb.Context(), name)
	if err != nil {
		tb.Fatal(err)
	}
	return value
}

func mustSetKey(tb testing.TB, store pluginport.KeyStore, name, value string) {
	tb.Helper()
	if err := store.SetKey(tb.Context(), name, value); err != nil {
		tb.Fatal(err)
	}
}
