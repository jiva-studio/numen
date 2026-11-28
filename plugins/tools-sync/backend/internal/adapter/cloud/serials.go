package cloud

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol/epochtrust"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
)

// Pins keeps bytes by the name of a server.
type Pins interface {
	ReadPin(ctx context.Context, address string) ([]byte, error)
	WritePin(ctx context.Context, address string, key []byte) error
}

type pinSerials struct {
	pins    Pins
	name    string
	address string
}

// NewSerials returns the log of the highest certificate serial of the server at address, kept in the pins.
// A write with no vault to keep it in is skipped.
func NewSerials(pins Pins, address string) epochtrust.Serials {
	return pinSerials{pins: pins, name: "serial:" + address, address: address}
}

func (s pinSerials) ReadSerial(ctx context.Context) (uint64, error) {
	raw, err := s.pins.ReadPin(ctx, s.name)
	if err != nil || len(raw) == 0 {
		return 0, err
	}
	if len(raw) != 8 {
		return 0, fmt.Errorf("the kept certificate serial has %d bytes", len(raw))
	}
	return binary.BigEndian.Uint64(raw), nil
}

func (s pinSerials) WriteSerial(ctx context.Context, serial uint64) error {
	err := s.pins.WritePin(ctx, s.name, binary.BigEndian.AppendUint64(nil, serial))
	if errors.Is(err, vaultlink.ErrNoVaults) {
		return nil
	}
	return err
}

func (s pinSerials) ReadIssuedAt(ctx context.Context) (int64, error) {
	raw, err := s.pins.ReadPin(ctx, "issued:"+s.address)
	if err != nil || len(raw) == 0 {
		return 0, err
	}
	if len(raw) != 8 {
		return 0, fmt.Errorf("the kept announcement time has %d bytes", len(raw))
	}
	return int64(binary.BigEndian.Uint64(raw)), nil
}

func (s pinSerials) WriteIssuedAt(ctx context.Context, issuedAt int64) error {
	err := s.pins.WritePin(ctx, "issued:"+s.address, binary.BigEndian.AppendUint64(nil, uint64(issuedAt)))
	if errors.Is(err, vaultlink.ErrNoVaults) {
		return nil
	}
	return err
}
