// Package identity creates, reads and deletes the account identity of this device: it derives the
// keys from the recovery words, builds the device chain and keeps the record in the key store.
package identity

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

const (
	keyName       = "identity"
	maxDeviceName = 64
)

// Deps groups the collaborators the identity use cases share.
type Deps struct {
	keys   pluginport.KeyStore
	names  DeviceName
	clock  machineport.Clock
	random io.Reader

	createLock CreateGate
}

// SharedKeys is the key store of an installation with the lock that serializes the creation of an identity
// over it. The windows of the installation share one value.
type SharedKeys struct {
	Store      pluginport.KeyStore
	CreateLock CreateGate
}

// NewDeps constructs the shared identity use case dependencies. Keys are drawn from crypto/rand.
func NewDeps(keys pluginport.KeyStore, names DeviceName, clock machineport.Clock) Deps {
	return Deps{keys: keys, names: names, clock: clock, random: rand.Reader, createLock: NewCreateGate()}
}

// WithCreateLock makes the use cases over one key store create the identity one at a time. Every
// window builds its own deps, so the windows of an installation pass the same lock. A nil lock keeps
// the gate the deps have.
func (d Deps) WithCreateLock(lock CreateGate) Deps {
	if lock != nil {
		d.createLock = lock
	}
	return d
}

func (d Deps) readRecord(ctx context.Context) (identitylib.Record, error) {
	value, err := d.keys.GetKey(ctx, keyName)
	if err != nil {
		return identitylib.Record{}, err
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

func toIdentity(record identitylib.Record) (identitydomain.Identity, error) {
	own := record.Device.Public()
	for _, device := range record.Chain.ListDevices() {
		if string(device.Public.Sign) == string(own.Sign) {
			return identitydomain.Identity{
				AccountFingerprint: record.Chain.Root().Fingerprint().String(),
				DeviceFingerprint:  own.Fingerprint().String(),
				DeviceName:         device.Name,
				DeviceAddedAt:      time.Unix(device.AddedAt.Unix(), 0).UTC(),
			}, nil
		}
	}
	return identitydomain.Identity{}, fmt.Errorf("%w: the device is not in its chain", identitylib.ErrMalformed)
}

func normalizeEntryName(name string) string {
	name = strings.ToValidUTF8(name, "")
	name = strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, name))
	if len(name) <= maxDeviceName {
		return name
	}
	cut := maxDeviceName
	for cut > 0 && !utf8.RuneStart(name[cut]) {
		cut--
	}
	return name[:cut]
}

func isNoKey(err error) bool { return errors.Is(err, plugindomain.ErrNoKey) }
