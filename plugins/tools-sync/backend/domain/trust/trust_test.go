package trust_test

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/trust"
)

func TestNewAuthors_TrustsOnlyTheDevicesThatRemainAfterARevoke(t *testing.T) {
	t.Parallel()
	words, err := identity.NewRecoveryWords(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	recovery, err := words.DeriveRecoveryKey()
	if err != nil {
		t.Fatal(err)
	}
	keys, err := identity.NewAccountKeys(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	chain, err := identity.NewChain(recovery, keys, time.Unix(1_700_000_000, 0))
	if err != nil {
		t.Fatal(err)
	}
	laptop, err := identity.NewDeviceKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if chain, err = chain.AppendDevice(recovery, laptop.Public(), "Laptop", time.Unix(1_700_000_100, 0)); err != nil {
		t.Fatal(err)
	}
	phone, err := identity.NewDeviceKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if chain, err = chain.AppendDevice(laptop, phone.Public(), "Phone", time.Unix(1_700_000_200, 0)); err != nil {
		t.Fatal(err)
	}
	own := chain
	revoked, _, err := chain.RevokeDevice(laptop, keys, phone.Public().Sign, rand.Reader, time.Unix(1_700_000_300, 0))
	if err != nil {
		t.Fatal(err)
	}

	authors, err := trust.NewAuthors(own.Bytes(), revoked.Bytes())

	if err != nil || !authors.IsTrusted(laptop.Public().Sign) || authors.IsTrusted(phone.Public().Sign) {
		t.Errorf("laptop trusted = %v, phone trusted = %v, error = %v, want the laptop only", authors.IsTrusted(laptop.Public().Sign), authors.IsTrusted(phone.Public().Sign), err)
	}
}
