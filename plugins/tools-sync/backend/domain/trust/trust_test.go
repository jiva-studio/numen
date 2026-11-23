package trust_test

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/trust"
)

type account struct {
	recovery identity.RecoveryKey
	keys     identity.AccountKeys
	chain    identity.Chain
}

func newAccount(tb testing.TB) account {
	tb.Helper()
	words, err := identity.NewRecoveryWords(rand.Reader)
	if err != nil {
		tb.Fatal(err)
	}
	recovery, err := words.DeriveRecoveryKey()
	if err != nil {
		tb.Fatal(err)
	}
	keys, err := identity.NewAccountKeys(rand.Reader)
	if err != nil {
		tb.Fatal(err)
	}
	chain, err := identity.NewChain(recovery, keys, time.Unix(1_700_000_000, 0))
	if err != nil {
		tb.Fatal(err)
	}
	return account{recovery: recovery, keys: keys, chain: chain}
}

func (a *account) add(tb testing.TB, by identity.Signer, name string) identity.DeviceKey {
	tb.Helper()
	device, err := identity.NewDeviceKey(rand.Reader)
	if err != nil {
		tb.Fatal(err)
	}
	if a.chain, err = a.chain.AppendDevice(by, device.Public(), name, time.Unix(1_700_000_100, 0)); err != nil {
		tb.Fatal(err)
	}
	return device
}

func TestNewAuthors_TrustsTheActiveDevicesOfAChainThatExtendsTheOwnOne(t *testing.T) {
	t.Parallel()
	a := newAccount(t)
	laptop := a.add(t, a.recovery, "Laptop")
	own := a.chain
	phone := a.add(t, laptop, "Phone")

	authors, err := trust.NewAuthors(own.Bytes(), a.chain.Bytes())

	if err != nil || !authors.IsTrusted(laptop.Public().Sign) || !authors.IsTrusted(phone.Public().Sign) {
		t.Errorf("authors = %+v, error = %v, want both devices trusted", authors, err)
	}
}

func TestNewAuthors_DoesNotTrustARevokedDevice(t *testing.T) {
	t.Parallel()
	a := newAccount(t)
	laptop := a.add(t, a.recovery, "Laptop")
	phone := a.add(t, a.recovery, "Phone")
	own := a.chain
	revoked, _, err := a.chain.RevokeDevice(laptop, a.keys, phone.Public().Sign, identity.Lost, rand.Reader, time.Unix(1_700_000_200, 0))
	if err != nil {
		t.Fatal(err)
	}

	authors, err := trust.NewAuthors(own.Bytes(), revoked.Bytes())

	if err != nil || authors.IsTrusted(phone.Public().Sign) || !authors.IsTrusted(laptop.Public().Sign) {
		t.Errorf("error = %v, phone trusted = %v, want only the laptop", err, authors.IsTrusted(phone.Public().Sign))
	}
}

func TestNewAuthors_TrustsTheDevicesOfAChainOfTheRecoveryKeyThatReplacesTheOwnOneAtAFork(t *testing.T) {
	t.Parallel()
	a := newAccount(t)
	laptop := a.add(t, a.recovery, "Laptop")
	phone := a.add(t, a.recovery, "Phone")
	base := a.chain
	thief := a.add(t, laptop, "Thief phone")
	stolen := a.chain
	owner, _, err := base.RevokeDevice(a.recovery, a.keys, laptop.Public().Sign, identity.Lost, rand.Reader, time.Unix(1_700_000_300, 0))
	if err != nil {
		t.Fatal(err)
	}

	authors, err := trust.NewAuthors(stolen.Bytes(), owner.Bytes())

	if err != nil || !authors.IsTrusted(phone.Public().Sign) || authors.IsTrusted(thief.Public().Sign) || authors.IsTrusted(laptop.Public().Sign) {
		t.Errorf("error = %v, want only the phone trusted", err)
	}
}

func TestNewAuthors_RefusesAForkOfTwoDevices(t *testing.T) {
	t.Parallel()
	a := newAccount(t)
	laptop := a.add(t, a.recovery, "Laptop")
	phone := a.add(t, a.recovery, "Phone")
	base := a.chain
	a.add(t, laptop, "Tablet")
	own := a.chain
	a.chain = base
	a.add(t, phone, "Desktop")

	if _, err := trust.NewAuthors(own.Bytes(), a.chain.Bytes()); err == nil {
		t.Error("a fork of two devices was trusted")
	}
	if _, err := trust.NewAuthors(own.Bytes(), base.Bytes()); err != nil {
		t.Errorf("a server that went back: error = %v, want the own chain", err)
	}
}

func TestNewAuthors_KnowsARevokedDeviceButNeverTrustsIt(t *testing.T) {
	t.Parallel()
	a := newAccount(t)
	laptop := a.add(t, a.recovery, "Laptop")
	phone := a.add(t, a.recovery, "Phone")
	own := a.chain
	revoked, _, err := a.chain.RevokeDevice(laptop, a.keys, phone.Public().Sign, identity.Lost, rand.Reader, time.Unix(1_700_000_200, 0))
	if err != nil {
		t.Fatal(err)
	}

	authors, err := trust.NewAuthors(own.Bytes(), revoked.Bytes())

	if err != nil || !authors.IsKnown(phone.Public().Sign) || authors.IsTrusted(phone.Public().Sign) || authors.IsKnown(make([]byte, 32)) {
		t.Errorf("error = %v, want the phone known and not trusted, and a stranger not known", err)
	}
}
