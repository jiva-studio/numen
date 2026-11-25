package cloud_test

import (
	"context"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/cloud"
)

type memoryPins map[string][]byte

func (m memoryPins) ReadPin(_ context.Context, address string) ([]byte, error) {
	return m[address], nil
}

func (m memoryPins) WritePin(_ context.Context, address string, key []byte) error {
	m[address] = key
	return nil
}

func TestSerials_KeepTheHighestSerialPerServerInThePins(t *testing.T) {
	t.Parallel()
	pins := memoryPins{}
	first := cloud.NewSerials(pins, "https://one.example")

	if got, err := first.ReadSerial(t.Context()); err != nil || got != 0 {
		t.Fatalf("ReadSerial() = %d, %v, want 0", got, err)
	}
	if err := first.WriteSerial(t.Context(), 258); err != nil {
		t.Fatal(err)
	}

	if got, err := cloud.NewSerials(pins, "https://one.example").ReadSerial(t.Context()); err != nil || got != 258 {
		t.Errorf("ReadSerial() = %d, %v, want 258", got, err)
	}
	if got, err := cloud.NewSerials(pins, "https://two.example").ReadSerial(t.Context()); err != nil || got != 0 {
		t.Errorf("another server: ReadSerial() = %d, %v, want 0", got, err)
	}
}

func TestSerials_KeepTheLatestAnnouncementTimePerServerInThePins(t *testing.T) {
	t.Parallel()
	pins := memoryPins{}
	first := cloud.NewSerials(pins, "https://one.example")

	if got, err := first.ReadIssuedAt(t.Context()); err != nil || got != 0 {
		t.Fatalf("ReadIssuedAt() = %d, %v, want 0", got, err)
	}
	if err := first.WriteIssuedAt(t.Context(), 1_800_000_500); err != nil {
		t.Fatal(err)
	}

	if got, err := cloud.NewSerials(pins, "https://one.example").ReadIssuedAt(t.Context()); err != nil || got != 1_800_000_500 {
		t.Errorf("ReadIssuedAt() = %d, %v, want 1800000500", got, err)
	}
	if got, err := cloud.NewSerials(pins, "https://one.example").ReadSerial(t.Context()); err != nil || got != 0 {
		t.Errorf("ReadSerial() = %d, %v, want the serial kept apart", got, err)
	}
	if got, err := cloud.NewSerials(pins, "https://two.example").ReadIssuedAt(t.Context()); err != nil || got != 0 {
		t.Errorf("another server: ReadIssuedAt() = %d, %v, want 0", got, err)
	}
}
