package identity

import (
	"crypto/ed25519"
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

const entryVersion = 0x01

func (c Chain) Bytes() []byte {
	out := binary.AppendUvarint(nil, uint64(len(c.entries)))
	for _, entry := range c.entries {
		encoded := encodeEntry(entry)
		out = binary.AppendUvarint(out, uint64(len(encoded)))
		out = append(out, encoded...)
	}
	return out
}

func ParseChain(b []byte) (Chain, error) {
	if len(b) > MaxChainBytes {
		return Chain{}, fmt.Errorf("%w: %d bytes, at most %d", ErrChainTooLong, len(b), MaxChainBytes)
	}
	r := reader{b: b}
	count := r.uvarint()
	if r.err != nil || count == 0 || count > uint64(len(b)) {
		return Chain{}, fmt.Errorf("%w: the entry count", ErrMalformed)
	}
	entries := make([]Entry, 0, count)
	for range count {
		raw := r.sized()
		if r.err != nil {
			return Chain{}, r.err
		}
		entry, err := decodeEntry(raw)
		if err != nil {
			return Chain{}, err
		}
		entries = append(entries, entry)
	}
	if !r.done() {
		return Chain{}, fmt.Errorf("%w: bytes after the entries", ErrMalformed)
	}
	chain := Chain{entries: entries}
	if err := chain.Verify(); err != nil {
		return Chain{}, err
	}
	return chain, nil
}

func encodeEntryBody(entry Entry) []byte {
	out := []byte{entryVersion, byte(entry.Kind)}
	out = binary.AppendUvarint(out, entry.Seq)
	out = append(out, entry.Prev[:]...)
	out = binary.BigEndian.AppendUint64(out, uint64(entry.Time.Unix()))
	out = append(out, entry.Signer...)
	switch {
	case entry.Genesis != nil:
		genesis := entry.Genesis
		out = append(out, genesis.Recovery.Sign...)
		out = append(out, genesis.Recovery.Box[:]...)
		out = append(out, genesis.AccountBox[:]...)
		out = binary.AppendUvarint(out, uint64(len(genesis.WrappedForRecovery.sealed)))
		out = append(out, genesis.WrappedForRecovery.sealed...)
	case entry.AddDevice != nil:
		add := entry.AddDevice
		out = append(out, add.Device.Sign...)
		out = append(out, add.Device.Box[:]...)
		out = binary.AppendUvarint(out, uint64(len(add.Name)))
		out = append(out, add.Name...)
	case entry.Revoke != nil:
		out = appendRevoke(out, entry.Revoke)
	}
	return out
}

func encodeEntry(entry Entry) []byte {
	return append(encodeEntryBody(entry), entry.Signature[:]...)
}

func decodeEntry(b []byte) (Entry, error) {
	r := reader{b: b}
	if version := r.byte(); r.err == nil && version != entryVersion {
		return Entry{}, fmt.Errorf("%w: entry version %d", ErrUnsupportedVersion, version)
	}
	var entry Entry
	entry.Kind = EntryKind(r.byte())
	entry.Seq = r.uvarint()
	copy(entry.Prev[:], r.take(32))
	entry.Time = time.Unix(int64(binary.BigEndian.Uint64(r.takeOrZeros(8))), 0).UTC()
	entry.Signer = ed25519.PublicKey(r.take(ed25519.PublicKeySize))
	switch entry.Kind {
	case KindGenesis:
		genesis := &Genesis{}
		genesis.Recovery.Sign = ed25519.PublicKey(r.take(ed25519.PublicKeySize))
		copy(genesis.Recovery.Box[:], r.take(32))
		copy(genesis.AccountBox[:], r.take(32))
		genesis.WrappedForRecovery.sealed = r.sized()
		entry.Genesis = genesis
	case KindAddDevice:
		add := &AddDevice{}
		add.Device.Sign = ed25519.PublicKey(r.take(ed25519.PublicKeySize))
		copy(add.Device.Box[:], r.take(32))
		add.Name = string(r.sized())
		entry.AddDevice = add
	case KindRevokeDevice:
		entry.Revoke = readRevoke(&r)
	default:
		if r.err == nil {
			r.err = fmt.Errorf("%w: entry kind %d", ErrMalformed, entry.Kind)
		}
	}
	copy(entry.Signature[:], r.take(64))
	if r.err != nil {
		return Entry{}, r.err
	}
	if !r.done() {
		return Entry{}, fmt.Errorf("%w: bytes after an entry", ErrMalformed)
	}
	return entry, nil
}

// reader keeps the first error; every read after an error returns zero values.
type reader struct {
	b   []byte
	err error
}

func (r *reader) fail() {
	if r.err == nil {
		r.err = fmt.Errorf("%w: data is cut short", ErrMalformed)
	}
}

func (r *reader) done() bool { return r.err == nil && len(r.b) == 0 }

func (r *reader) take(n int) []byte {
	if r.err != nil || n < 0 || n > len(r.b) {
		r.fail()
		return nil
	}
	out := append([]byte(nil), r.b[:n]...)
	r.b = r.b[n:]
	return out
}

func (r *reader) takeOrZeros(n int) []byte {
	if out := r.take(n); out != nil {
		return out
	}
	return make([]byte, n)
}

func (r *reader) byte() byte {
	if out := r.take(1); out != nil {
		return out[0]
	}
	return 0
}

func (r *reader) uvarint() uint64 {
	if r.err != nil {
		return 0
	}
	value, n := binary.Uvarint(r.b)
	if n <= 0 {
		r.fail()
		return 0
	}
	r.b = r.b[n:]
	return value
}

func (r *reader) sized() []byte {
	size := r.uvarint()
	if size > uint64(len(r.b)) {
		r.fail()
		return nil
	}
	return r.take(int(size))
}

func appendRevoke(out []byte, revoke *Revoke) []byte {
	out = append(out, revoke.Device...)
	out = binary.AppendUvarint(out, uint64(revoke.Epoch))
	out = append(out, revoke.AccountBox[:]...)
	out = binary.AppendUvarint(out, uint64(len(revoke.WrappedForRecovery.sealed)))
	out = append(out, revoke.WrappedForRecovery.sealed...)
	out = binary.AppendUvarint(out, uint64(len(revoke.PreviousKeys)))
	out = append(out, revoke.PreviousKeys...)
	out = binary.AppendUvarint(out, uint64(len(revoke.Wrapped)))
	for _, wrap := range revoke.Wrapped {
		out = append(out, wrap.Device...)
		out = binary.AppendUvarint(out, uint64(len(wrap.Wrapped.sealed)))
		out = append(out, wrap.Wrapped.sealed...)
	}
	return out
}

func readRevoke(r *reader) *Revoke {
	revoke := &Revoke{Device: ed25519.PublicKey(r.take(ed25519.PublicKeySize))}
	epoch := r.uvarint()
	if epoch > math.MaxUint32 {
		r.fail()
	}
	revoke.Epoch = uint32(epoch)
	copy(revoke.AccountBox[:], r.take(32))
	revoke.WrappedForRecovery.sealed = r.sized()
	revoke.PreviousKeys = r.sized()
	count := r.uvarint()
	if count > uint64(len(r.b)) {
		r.fail()
		return revoke
	}
	for range count {
		wrap := DeviceWrap{Device: ed25519.PublicKey(r.take(ed25519.PublicKeySize))}
		wrap.Wrapped.sealed = r.sized()
		revoke.Wrapped = append(revoke.Wrapped, wrap)
	}
	return revoke
}
