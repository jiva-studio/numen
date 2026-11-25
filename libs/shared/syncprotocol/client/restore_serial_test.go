package client_test

import (
	"context"
	"crypto/ed25519"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/epochtrust"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

type memorySerials struct{ highest uint64 }

func (m *memorySerials) ReadSerial(context.Context) (uint64, error) { return m.highest, nil }

func (m *memorySerials) WriteSerial(_ context.Context, serial uint64) error {
	m.highest = serial
	return nil
}

func newResponseOfSerial(tb testing.TB, serial uint64) *syncv1.GetRestoreEpochResponse {
	tb.Helper()
	root, _ := newRoot(tb, 1)
	private, public := newRoot(tb, 2)
	cert, err := epochtrust.SignCert(root, public, serial, getTestNow().Add(-time.Hour), getTestNow().Add(time.Hour))
	if err != nil {
		tb.Fatal(err)
	}
	id := []byte("0123456789abcdef")
	return &syncv1.GetRestoreEpochResponse{
		EpochSignKey: public, EpochSerial: cert.Serial, EpochNotBefore: cert.NotBefore, EpochNotAfter: cert.NotAfter, RootSignature: cert.RootSignature,
		Epoch: &syncv1.RestoreEpoch{Id: id, IssuedAt: 1_800_000_000, Signature: ed25519.Sign(private, syncprotocol.GetRestoreEpochMessage(id, 1_800_000_000))},
	}
}

func TestRestoreClient_RefusesACertificateWithALowerSerialThanOneSeenBefore(t *testing.T) {
	t.Parallel()
	_, rootPublic := newRoot(t, 1)
	serials := &memorySerials{}
	stub, first := serveRestoreStub(t, newResponseOfSerial(t, 5), rootPublic)
	first.WithSerials(serials)
	if _, _, err := first.GetRestoreEpoch(t.Context()); err != nil {
		t.Fatal(err)
	}
	if serials.highest != 5 {
		t.Fatalf("kept serial = %d, want 5", serials.highest)
	}

	stub.mu.Lock()
	stub.response = newResponseOfSerial(t, 4)
	stub.mu.Unlock()
	_, _, err := first.GetRestoreEpoch(t.Context())

	if !errors.Is(err, epochtrust.ErrUntrusted) {
		t.Errorf("GetRestoreEpoch = %v, want ErrUntrusted", err)
	}
	if serials.highest != 5 {
		t.Errorf("kept serial = %d, want 5", serials.highest)
	}
}

func TestRestoreClient_AcceptsTheSameAndAHigherSerialAndKeepsTheHighest(t *testing.T) {
	t.Parallel()
	_, rootPublic := newRoot(t, 1)
	serials := &memorySerials{highest: 5}
	stub, client := serveRestoreStub(t, newResponseOfSerial(t, 5), rootPublic)
	client.WithSerials(serials)
	if _, _, err := client.GetRestoreEpoch(t.Context()); err != nil {
		t.Fatalf("same serial: %v", err)
	}

	stub.mu.Lock()
	stub.response = newResponseOfSerial(t, 6)
	stub.mu.Unlock()
	if _, _, err := client.GetRestoreEpoch(t.Context()); err != nil {
		t.Fatalf("higher serial: %v", err)
	}

	if serials.highest != 6 {
		t.Errorf("kept serial = %d, want 6", serials.highest)
	}
}
