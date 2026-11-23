package identity

import (
	"encoding/binary"
	"fmt"
)

const recordVersion = 0x02

// Record is what one device keeps in its key store, written and deleted whole: the device seed and the account keys wrapped for it.
// The chain is kept next to it in a file, see State.
type Record struct {
	Device  DeviceKey
	Wrapped WrappedAccountKeys
}

func (r Record) Bytes() []byte {
	out := []byte{recordVersion}
	out = append(out, r.Device.seed[:]...)
	out = binary.AppendUvarint(out, uint64(len(r.Wrapped.sealed)))
	return append(out, r.Wrapped.sealed...)
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
	if !r.done() {
		return Record{}, fmt.Errorf("%w: bytes after the record", ErrMalformed)
	}
	device, err := newDeviceKey(seed)
	if err != nil {
		return Record{}, err
	}
	wrapped, err := ParseWrappedAccountKeys(sealed)
	if err != nil {
		return Record{}, err
	}
	return Record{Device: device, Wrapped: wrapped}, nil
}
