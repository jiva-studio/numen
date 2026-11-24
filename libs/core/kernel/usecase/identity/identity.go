// Package identity creates, reads and deletes the account identity of this device: it derives the
// keys from the recovery words, builds the device chain and keeps the record in the key store.
package identity

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

const (
	keyName       = "identity"
	maxDeviceName = 64
)

// Clock is the time source of the identity use cases: the time now and a signal after a delay.
type Clock interface {
	machineport.Clock
	After(d time.Duration) <-chan time.Time
}

// Deps groups the collaborators the identity use cases are made from; each use case keeps the ones it calls. Copies
// of one Deps share one Progress.
type Deps struct {
	records  records
	names    DeviceName
	clock    Clock
	random   io.Reader
	progress *Progress

	createLock CreateGate
}

// SharedKeys is the key store of an installation with the lock that serializes the creation of an identity
// over it. The windows of the installation share one value.
type SharedKeys struct {
	Store      pluginport.KeyStore
	CreateLock CreateGate
}

// NewDeps constructs the shared identity use case dependencies. Keys are drawn from crypto/rand.
func NewDeps(keys pluginport.KeyStore, names DeviceName, clock Clock) Deps {
	progress := NewProgress()
	return Deps{records: records{keys: keys}, names: names, clock: clock, random: rand.Reader, progress: progress, createLock: progress.gate}
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

// WithProgress returns the deps over another progress, so the use cases of a window and of the plugins share one.
func (d Deps) WithProgress(progress *Progress) Deps {
	if progress != nil {
		d.progress = progress
		if d.records.keys != nil {
			d.records.keys = watchedKeys{KeyStore: d.records.keys, signal: progress.changes}
		}
		d.createLock = progress.gate
	}
	return d
}

// WithChains returns the deps over the chain file of this device.
func (d Deps) WithChains(chains ChainStore) Deps {
	d.records.chains = chains
	return d
}

func (d Deps) getBuilder() stateBuilder {
	return stateBuilder{names: d.names, clock: d.clock, random: d.random}
}

func (d Deps) getHost() linkHost {
	return linkHost{records: d.records, clock: d.clock, random: d.random}
}

func (d Deps) getRevocation() revocation {
	return revocation{records: d.records, clock: d.clock, random: d.random}
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

func encodeRecord(record identitylib.Record) string {
	return base64.RawURLEncoding.EncodeToString(record.Bytes())
}

func (d Deps) getCompletion() restoreCompletion {
	return restoreCompletion{records: d.records, restores: d.progress.restores, builder: d.getBuilder(), gate: d.createLock}
}
