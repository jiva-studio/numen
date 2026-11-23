package identity_test

import (
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/identity"
)

func growChain(tb testing.TB) (identity.Chain, error) {
	tb.Helper()
	a := newChainOf(tb)
	signer := newDeviceOf(tb)
	chain, err := appended(a.chain, a.recovery, signer.Public(), "Signer", epoch())
	if err != nil {
		tb.Fatal(err)
	}
	keys := a.keys
	for i := range 2000 {
		at := epoch().Add(time.Duration(i+1) * time.Second)
		next, err := appended(chain, signer, newDeviceOf(tb).Public(), "Guest", at)
		if err != nil {
			return chain, err
		}
		guest := next.ListDevices()[len(next.ListDevices())-1].Public.Sign
		revoked, nextKeys, err := next.RevokeDevice(signer, keys, guest, rand.Reader, at)
		if err != nil {
			return chain, err
		}
		chain, keys = revoked, nextKeys
	}
	tb.Fatal("2000 devices did not reach the limit of the chain")
	return chain, nil
}

func TestChain_RefusesAnEntryThatMakesItLongerThan192KiB(t *testing.T) {
	t.Parallel()
	chain, err := growChain(t)
	if !errors.Is(err, identity.ErrChainTooLong) {
		t.Fatalf("error = %v, want ErrChainTooLong", err)
	}
	if size := len(chain.Bytes()); size > identity.MaxChainBytes || size < identity.MaxChainBytes-2048 {
		t.Errorf("the last chain that fits has %d bytes, want just under %d", size, identity.MaxChainBytes)
	}
	if _, err := identity.ParseChain(chain.Bytes()); err != nil {
		t.Errorf("a chain at the limit does not parse: %v", err)
	}
}

func TestParseChain_RefusesMoreThan192KiBBeforeReadingIt(t *testing.T) {
	t.Parallel()
	chain, _ := growChain(t)
	oversized := append(chain.Bytes(), make([]byte, 4096)...)
	if _, err := identity.ParseChain(oversized); !errors.Is(err, identity.ErrChainTooLong) {
		t.Errorf("error = %v, want ErrChainTooLong", err)
	}
}
