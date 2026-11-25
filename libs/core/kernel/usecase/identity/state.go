package identity

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"time"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

// records keeps the identity of this device: the record in the key store and the device chain in its file.
type records struct {
	keys   pluginport.KeyStore
	chains ChainStore
	clock  Clock
}

func (r records) readRecord(ctx context.Context) (identitylib.Record, error) {
	value, err := r.keys.GetKey(ctx, keyName)
	if isNoKey(err) {
		return identitylib.Record{}, err
	}
	if err != nil {
		return identitylib.Record{}, fmt.Errorf("%w: read the identity: %w", identitydomain.ErrKeyStore, err)
	}
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return identitylib.Record{}, fmt.Errorf("%w: the record is not base64: %w", identitylib.ErrMalformed, err)
	}
	record, err := identitylib.ParseRecord(raw)
	if err != nil {
		return identitylib.Record{}, fmt.Errorf("%w: the record is not valid: %w", identitylib.ErrMalformed, err)
	}
	return record, nil
}

func (r records) checkAbsent(ctx context.Context) error {
	_, err := r.keys.GetKey(ctx, keyName)
	switch {
	case err == nil:
		return r.checkReplaceable(ctx)
	case isNoKey(err):
		return nil
	}
	return fmt.Errorf("%w: read the identity: %w", identitydomain.ErrKeyStore, err)
}

func (r records) checkReplaceable(ctx context.Context) error {
	_, err := r.readState(ctx)
	switch {
	case err == nil:
		return identitydomain.ErrIdentityExists
	case errors.Is(err, identitydomain.ErrNoIdentity):
		return nil
	}
	return err
}

func (r records) readState(ctx context.Context) (identitylib.State, error) {
	state, err := r.readStoredState(ctx)
	if errors.Is(err, identitylib.ErrMalformed) || errors.Is(err, identitylib.ErrUnsupportedVersion) || errors.Is(err, identitydomain.ErrNoChain) {
		return identitylib.State{}, fmt.Errorf("%w: %w", identitydomain.ErrNoIdentity, err)
	}
	return state, err
}

func (r records) readStoredState(ctx context.Context) (identitylib.State, error) {
	record, err := r.readRecord(ctx)
	if isNoKey(err) {
		return identitylib.State{}, identitydomain.ErrNoIdentity
	}
	if err != nil {
		return identitylib.State{}, err
	}
	chain, err := r.readChain(ctx)
	if err != nil {
		return identitylib.State{}, err
	}
	state, err := identitylib.NewState(record, chain)
	if err != nil {
		return identitylib.State{}, fmt.Errorf("%w: the record does not fit its chain: %w", identitylib.ErrMalformed, err)
	}
	return state, nil
}

func (r records) readChain(ctx context.Context) (identitylib.Chain, error) {
	raw, err := r.chains.Read(ctx)
	if err != nil {
		return identitylib.Chain{}, err
	}
	chain, err := identitylib.ParseChain(raw)
	if err != nil {
		return identitylib.Chain{}, fmt.Errorf("%w: the chain file is not valid: %w", identitylib.ErrMalformed, err)
	}
	return chain, nil
}

func (r records) writeChain(ctx context.Context, chain identitylib.Chain) error {
	if err := r.chains.Write(ctx, chain.Bytes()); err != nil {
		return fmt.Errorf("%w: keep the chain: %w", identitydomain.ErrKeyStore, err)
	}
	return nil
}

func (r records) writeRecord(ctx context.Context, record identitylib.Record) error {
	if err := r.keys.SetKey(ctx, keyName, encodeRecord(record)); err != nil {
		return fmt.Errorf("%w: keep the identity: %w", identitydomain.ErrKeyStore, err)
	}
	return nil
}

func (r records) writeNewState(ctx context.Context, record identitylib.Record, chain identitylib.Chain) error {
	if err := raiseHighWater(ctx, r.keys, chain); err != nil {
		return err
	}
	if err := r.writeChain(ctx, chain); err != nil {
		return err
	}
	return r.writeRecord(ctx, record)
}

func (r records) adoptState(ctx context.Context, state identitylib.State, chain identitylib.Chain) (identitylib.State, error) {
	next, err := state.AdoptAt(chain, r.clock.Now())
	if err != nil {
		return identitylib.State{}, fmt.Errorf("%w: %w", identitydomain.ErrInvalidChain, err)
	}
	if bytes.Equal(next.Chain.Bytes(), state.Chain.Bytes()) {
		return next, nil
	}
	if err := raiseHighWater(ctx, r.keys, next.Chain); err != nil {
		return identitylib.State{}, err
	}
	if err := r.writeRecord(ctx, next.Record); err != nil {
		return identitylib.State{}, err
	}
	if err := r.writeChain(ctx, next.Chain); err != nil {
		return identitylib.State{}, err
	}
	return next, nil
}

func (r records) adoptLatest(ctx context.Context, state identitylib.State, latest []byte) (identitylib.State, bool) {
	head, err := identitylib.ParseChain(latest)
	if err != nil {
		return state, false
	}
	adopted, err := r.adoptState(ctx, state, head)
	return adopted, err == nil
}

func toIdentity(state identitylib.State) (identitydomain.Identity, error) {
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

func getHighWaterName(fingerprint string) string { return "chain-length-" + fingerprint }

func checkHighWater(ctx context.Context, keys pluginport.KeyStore, fingerprint string, length int) error {
	value, err := keys.GetKey(ctx, getHighWaterName(fingerprint))
	if isNoKey(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("%w: read the chain length: %w", identitydomain.ErrKeyStore, err)
	}
	highest, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("%w: the chain length is not a number: %w", identitylib.ErrMalformed, err)
	}
	if length < highest {
		return fmt.Errorf("%w: the chain has %d entries and this device has held one with %d", identitydomain.ErrChainOutdated, length, highest)
	}
	return nil
}

func raiseHighWater(ctx context.Context, keys pluginport.KeyStore, chain identitylib.Chain) error {
	if chain.Len() <= newIdentityChainLength {
		return nil
	}
	fingerprint := chain.Root().Fingerprint().String()
	err := checkHighWater(ctx, keys, fingerprint, chain.Len())
	if errors.Is(err, identitydomain.ErrChainOutdated) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := keys.SetKey(ctx, getHighWaterName(fingerprint), strconv.Itoa(chain.Len())); err != nil {
		return fmt.Errorf("%w: keep the chain length: %w", identitydomain.ErrKeyStore, err)
	}
	return nil
}

const newIdentityChainLength = 2
