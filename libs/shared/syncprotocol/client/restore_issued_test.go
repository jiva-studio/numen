package client_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol/epochtrust"
)

func TestRestoreClient_RefusesAnAnnouncementOlderThanTheLastAcceptedOne(t *testing.T) {
	t.Parallel()
	_, rootPublic := newRoot(t, 1)
	serials := &memorySerials{}
	stub, client := serveRestoreStub(t, newResponseIssuedAt(t, 5, 1_800_000_500), rootPublic)
	client.WithSerials(serials)
	if _, _, err := client.GetRestoreEpoch(t.Context()); err != nil {
		t.Fatal(err)
	}

	stub.mu.Lock()
	stub.response = newResponseIssuedAt(t, 5, 1_800_000_100)
	stub.mu.Unlock()
	epoch, _, err := client.GetRestoreEpoch(t.Context())

	if !errors.Is(err, epochtrust.ErrUntrusted) || epoch != nil {
		t.Errorf("GetRestoreEpoch = %v, %v, want ErrUntrusted and no epoch for a replayed announcement", epoch, err)
	}
	if serials.issuedAt != 1_800_000_500 {
		t.Errorf("kept issued_at = %d, want 1800000500", serials.issuedAt)
	}
}

func TestRestoreClient_AcceptsTheSameAndALaterAnnouncementAndKeepsTheLatest(t *testing.T) {
	t.Parallel()
	_, rootPublic := newRoot(t, 1)
	serials := &memorySerials{issuedAt: 1_800_000_500}
	stub, client := serveRestoreStub(t, newResponseIssuedAt(t, 5, 1_800_000_500), rootPublic)
	client.WithSerials(serials)
	if _, _, err := client.GetRestoreEpoch(t.Context()); err != nil {
		t.Fatalf("the same announcement: %v", err)
	}

	stub.mu.Lock()
	stub.response = newResponseIssuedAt(t, 5, 1_800_000_900)
	stub.mu.Unlock()
	if _, _, err := client.GetRestoreEpoch(t.Context()); err != nil {
		t.Fatalf("a later announcement: %v", err)
	}

	if serials.issuedAt != 1_800_000_900 {
		t.Errorf("kept issued_at = %d, want 1800000900", serials.issuedAt)
	}
}

func TestRestoreClient_KeepsNoIssuedAtForAServerThatAnnouncedNothing(t *testing.T) {
	t.Parallel()
	_, rootPublic := newRoot(t, 1)
	serials := &memorySerials{}
	response := newResponseIssuedAt(t, 5, 1_800_000_500)
	response.Epoch = nil
	_, client := serveRestoreStub(t, response, rootPublic)
	client.WithSerials(serials)

	if _, _, err := client.GetRestoreEpoch(t.Context()); err != nil {
		t.Fatal(err)
	}

	if serials.issuedAt != 0 {
		t.Errorf("kept issued_at = %d, want none", serials.issuedAt)
	}
}
