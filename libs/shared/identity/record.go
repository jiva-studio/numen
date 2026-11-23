package identity

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const recordVersion = 0x01

// Record is what one device keeps in its key store, written and deleted whole.
type Record struct {
	Device  DeviceKey
	Wrapped WrappedAccountKeys
	Chain   Chain
}

func (r Record) Bytes() []byte {
	out := []byte{recordVersion}
	out = append(out, r.Device.seed[:]...)
	out = binary.AppendUvarint(out, uint64(len(r.Wrapped.sealed)))
	out = append(out, r.Wrapped.sealed...)
	return append(out, r.Chain.Bytes()...)
}

func ParseRecord(b []byte) (Record, error) {
	r := reader{b: b}
	if version := r.byte(); r.err == nil && version != recordVersion {
		return Record{}, fmt.Errorf("%w: record version %d", ErrUnsupportedVersion, version)
	}
	var seed [32]byte
	copy(seed[:], r.take(32))
	sealed := r.sized()
	if r.err != nil {
		return Record{}, r.err
	}
	device, err := newDeviceKey(seed)
	if err != nil {
		return Record{}, err
	}
	record := Record{Device: device}
	wrapped, err := ParseWrappedAccountKeys(sealed)
	if err != nil {
		return Record{}, err
	}
	record.Wrapped = wrapped
	chain, err := ParseChain(r.b)
	if err != nil {
		return Record{}, err
	}
	record.Chain = chain
	opened, err := record.Device.UnwrapAccountKeys(wrapped)
	if err != nil {
		return Record{}, fmt.Errorf("%w: the wrapped account keys do not open for the device: %w", ErrMalformed, err)
	}
	if opened.BoxPublicKey() != chain.AccountBoxKey() {
		return Record{}, fmt.Errorf("%w: the wrapped account keys are not the account of the chain", ErrMalformed)
	}
	own := record.Device.SignerKey()
	for _, device := range chain.ListDevices() {
		if bytes.Equal(device.Public.Sign, own) {
			return record, nil
		}
	}
	return Record{}, fmt.Errorf("%w: the device of the record is not in its chain", ErrMalformed)
}
