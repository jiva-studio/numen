package identity

import (
	"bytes"
	"crypto/ed25519"
	"fmt"
	"slices"
)

// roster is the set of devices a chain lists after some number of entries.
type roster struct {
	root    ed25519.PublicKey
	active  []Device
	revoked []Device
	known   map[string]struct{}
	epoch   uint32
}

func newRoster(root ed25519.PublicKey) roster {
	return roster{root: root, known: make(map[string]struct{})}
}

func (c Chain) replay() roster {
	state := newRoster(c.Root().Sign)
	for _, entry := range c.entries[min(1, len(c.entries)):] {
		state.apply(entry)
	}
	return state
}

func (r *roster) indexOf(sign ed25519.PublicKey) int {
	return slices.IndexFunc(r.active, func(d Device) bool { return bytes.Equal(d.Public.Sign, sign) })
}

func (r *roster) isSigner(sign ed25519.PublicKey) bool {
	return bytes.Equal(sign, r.root) || r.indexOf(sign) >= 0
}

func (r *roster) apply(entry Entry) {
	switch {
	case entry.Kind == KindAddDevice && entry.AddDevice != nil:
		r.active = append(r.active, Device{Public: entry.AddDevice.Device, Name: entry.AddDevice.Name, AddedAt: entry.Time})
		r.known[string(entry.AddDevice.Device.Sign)] = struct{}{}
	case entry.Kind == KindRevokeDevice && entry.Revoke != nil:
		if i := r.indexOf(entry.Revoke.Device); i >= 0 {
			removal := Removal{At: entry.Time, By: entry.Signer, Reason: entry.Revoke.Reason, Since: entry.Revoke.Since}
			r.drop(i, removal)
		}
		r.epoch = entry.Revoke.Epoch
	case entry.Kind == KindRotateKeys && entry.Revoke != nil:
		r.epoch = entry.Revoke.Epoch
	}
}

func (r *roster) drop(i int, removal Removal) {
	device := r.active[i]
	device.Removal = removal
	r.revoked = append(r.revoked, device)
	r.active = slices.Delete(slices.Clone(r.active), i, i+1)
}

func (r *roster) verify(entry Entry, i int) error {
	if !r.isSigner(entry.Signer) {
		return fmt.Errorf("%w: entry %d", ErrUnknownSigner, i)
	}
	if !hasValidSignature(entry) {
		return fmt.Errorf("%w: entry %d", ErrBadSignature, i)
	}
	switch entry.Kind {
	case KindAddDevice:
		return r.verifyAdd(entry, i)
	case KindRevokeDevice:
		return r.verifyRevoke(entry, i)
	case KindRotateKeys:
		return r.verifyRotate(entry, i)
	default:
		return fmt.Errorf("%w: entry %d has kind %d", ErrMalformed, i, entry.Kind)
	}
}

func (r *roster) verifyAdd(entry Entry, i int) error {
	if entry.AddDevice == nil {
		return fmt.Errorf("%w: entry %d has no device", ErrMalformed, i)
	}
	if err := checkDeviceName(entry.AddDevice.Name); err != nil {
		return err
	}
	added := entry.AddDevice.Device.Sign
	if _, ok := r.known[string(added)]; ok || bytes.Equal(added, r.root) {
		return fmt.Errorf("%w: entry %d", ErrDuplicateDevice, i)
	}
	if len(r.active) >= MaxActiveDevices {
		return fmt.Errorf("%w: entry %d", ErrTooManyDevices, i)
	}
	return nil
}

func (r *roster) verifyRevoke(entry Entry, i int) error {
	revoke := entry.Revoke
	if revoke == nil {
		return fmt.Errorf("%w: entry %d has no revoke", ErrMalformed, i)
	}
	if bytes.Equal(revoke.Device, entry.Signer) {
		return fmt.Errorf("%w: entry %d", ErrSelfRevoke, i)
	}
	index := r.indexOf(revoke.Device)
	if index < 0 {
		return fmt.Errorf("%w: entry %d", ErrUnknownDevice, i)
	}
	if err := checkRevocation(revoke, entry, i); err != nil {
		return err
	}
	return r.verifyKeys(revoke, slices.Delete(slices.Clone(r.active), index, index+1), i)
}

func checkRevocation(revoke *Revoke, entry Entry, i int) error {
	switch revoke.Reason {
	case ReasonLost:
		if !revoke.Since.IsZero() {
			return fmt.Errorf("%w: entry %d gives a date for a lost device", ErrMalformed, i)
		}
	case ReasonStolen:
		if revoke.Since.IsZero() || revoke.Since.After(entry.Time) {
			return fmt.Errorf("%w: entry %d has no date or a date after the entry", ErrMalformed, i)
		}
	default:
		return fmt.Errorf("%w: entry %d has reason %d", ErrMalformed, i, revoke.Reason)
	}
	return nil
}

func (r *roster) verifyRotate(entry Entry, i int) error {
	revoke := entry.Revoke
	if revoke == nil || len(revoke.Device) != 0 || revoke.Reason != 0 || !revoke.Since.IsZero() {
		return fmt.Errorf("%w: entry %d is not a rotation", ErrMalformed, i)
	}
	if !bytes.Equal(entry.Signer, r.root) {
		return fmt.Errorf("%w: entry %d is not signed by the recovery key", ErrUnknownSigner, i)
	}
	return r.verifyKeys(revoke, r.active, i)
}

// verifyKeys checks the keys an entry starts an epoch with: the epoch follows, the sizes are right and one wrap is made for each
// device that stays, in the order of the roster.
func (r *roster) verifyKeys(revoke *Revoke, remaining []Device, i int) error {
	if revoke.Epoch != r.epoch+1 {
		return fmt.Errorf("%w: entry %d raises the epoch to %d", ErrBrokenLink, i, revoke.Epoch)
	}
	if len(revoke.WrappedForRecovery.sealed) != wrappedSize || len(revoke.PreviousKeys) != previousKeysSize {
		return fmt.Errorf("%w: entry %d has keys of the wrong size", ErrMalformed, i)
	}
	if len(revoke.Wrapped) != len(remaining) {
		return fmt.Errorf("%w: entry %d", ErrWrappedSet, i)
	}
	for j, wrap := range revoke.Wrapped {
		if !bytes.Equal(wrap.Device, remaining[j].Public.Sign) || len(wrap.Wrapped.sealed) != wrappedSize {
			return fmt.Errorf("%w: entry %d", ErrWrappedSet, i)
		}
	}
	return nil
}

// Epoch returns the number of entries that changed the keys: 0 until the first one.
func (c Chain) Epoch() uint32 { return c.replay().epoch }

// ListRevokedDevices returns the revoked devices in the order they were revoked.
func (c Chain) ListRevokedDevices() []Device { return c.replay().revoked }

// IsActiveDevice reports whether the chain lists the signing key as an active device.
func (c Chain) IsActiveDevice(sign ed25519.PublicKey) bool {
	state := c.replay()
	return state.indexOf(sign) >= 0
}

// IsKnownDevice reports whether the key was ever added, revoked or not.
func (c Chain) IsKnownDevice(sign ed25519.PublicKey) bool {
	state := c.replay()
	_, ok := state.known[string(sign)]
	return ok
}

// IsSignedSince reports whether an entry after the prefix was signed by the key.
func (c Chain) IsSignedSince(prefix Chain, sign ed25519.PublicKey) bool {
	for i := len(prefix.entries); i < len(c.entries); i++ {
		if bytes.Equal(c.entries[i].Signer, sign) {
			return true
		}
	}
	return false
}

// SignersSince returns the signing key of each entry after the prefix, in order.
func (c Chain) SignersSince(prefix Chain) []ed25519.PublicKey {
	var signers []ed25519.PublicKey
	for i := len(prefix.entries); i < len(c.entries); i++ {
		signers = append(signers, bytes.Clone(c.entries[i].Signer))
	}
	return signers
}

// SignersSinceFork returns the signing key of each entry of c from the first entry where it differs from other, in order.
// It is empty when one chain is a prefix of the other.
func (c Chain) SignersSinceFork(other Chain) []ed25519.PublicKey {
	i := c.getForkIndex(other)
	if i < 0 {
		return nil
	}
	var signers []ed25519.PublicKey
	for _, entry := range c.entries[i:] {
		signers = append(signers, bytes.Clone(entry.Signer))
	}
	return signers
}
