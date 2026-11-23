package identity

import (
	"context"
	"fmt"
	"time"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

func (d Deps) readState(ctx context.Context) (identitylib.State, error) {
	record, err := readExistingRecord(ctx, d)
	if err != nil {
		return identitylib.State{}, err
	}
	chain, err := d.readChain(ctx)
	if err != nil {
		return identitylib.State{}, err
	}
	state, err := identitylib.NewState(record, chain)
	if err != nil {
		return identitylib.State{}, fmt.Errorf("%w: the record does not fit its chain: %w", identitylib.ErrMalformed, err)
	}
	return state, nil
}

func (d Deps) readChain(ctx context.Context) (identitylib.Chain, error) {
	raw, err := d.chains.Read(ctx)
	if err != nil {
		return identitylib.Chain{}, err
	}
	chain, err := identitylib.ParseChain(raw)
	if err != nil {
		return identitylib.Chain{}, fmt.Errorf("%w: the chain file is not valid: %w", identitylib.ErrMalformed, err)
	}
	return chain, nil
}

func (d Deps) writeChain(ctx context.Context, chain identitylib.Chain) error {
	if err := d.chains.Write(ctx, chain.Bytes()); err != nil {
		return fmt.Errorf("%w: keep the chain: %w", identitydomain.ErrKeyStore, err)
	}
	return nil
}

func (d Deps) writeRecord(ctx context.Context, record identitylib.Record) error {
	if err := d.keys.SetKey(ctx, keyName, encodeRecord(record)); err != nil {
		return fmt.Errorf("%w: keep the identity: %w", identitydomain.ErrKeyStore, err)
	}
	return nil
}

func (d Deps) writeNewState(ctx context.Context, record identitylib.Record, chain identitylib.Chain) error {
	if err := d.writeChain(ctx, chain); err != nil {
		return err
	}
	return d.writeRecord(ctx, record)
}

func (d Deps) writeAdoptedState(ctx context.Context, state identitylib.State) error {
	if err := d.writeRecord(ctx, state.Record); err != nil {
		return err
	}
	return d.writeChain(ctx, state.Chain)
}

func identityOf(state identitylib.State) (identitydomain.Identity, error) {
	own := state.Record.Device.Public()
	devices := append(state.Chain.ListDevices(), state.Chain.ListRevokedDevices()...)
	for _, device := range devices {
		if string(device.Public.Sign) == string(own.Sign) {
			return identitydomain.Identity{
				AccountFingerprint: state.Chain.Root().Fingerprint().String(),
				DeviceFingerprint:  own.Fingerprint().String(),
				DeviceName:         device.Name,
				DeviceAddedAt:      time.Unix(device.AddedAt.Unix(), 0).UTC(),
				IsRevoked:          state.Status == identitylib.StatusRevoked,
				KeyEpoch:           state.KeyEpoch(),
			}, nil
		}
	}
	return identitydomain.Identity{}, fmt.Errorf("%w: the device is not in its chain", identitylib.ErrMalformed)
}
