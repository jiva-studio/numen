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

// Clock is the time source of the identity use cases: the time now and a signal after a delay.
type Clock interface {
	machineport.Clock
	After(d time.Duration) <-chan time.Time
}

// Deps groups the collaborators the identity use cases share. Copies of one Deps share one pending restore.
type Deps struct {
	keys    pluginport.KeyStore
	names   DeviceName
	clock   Clock
	random  io.Reader
	pending *PendingStore
	chains  ChainStore
	links   *LinkStore
}

// NewDeps constructs the shared identity use case dependencies. Keys are drawn from crypto/rand.
func NewDeps(keys pluginport.KeyStore, names DeviceName, clock Clock) Deps {
	return Deps{keys: keys, names: names, clock: clock, random: rand.Reader, pending: NewPendingStore(), links: NewLinkStore()}
}

// WithPending returns the deps over another pending restore, so the use cases of a window and of the plugins share one.
func (d Deps) WithPending(pending *PendingStore) Deps {
	if pending != nil {
		d.pending = pending
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

func entryName(name string) string {
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

func readExistingRecord(ctx context.Context, deps Deps) (identitylib.Record, error) {
	record, err := deps.readRecord(ctx)
	if isNoKey(err) {
		return identitylib.Record{}, identitydomain.ErrNoIdentity
	}
	return record, err
}

// WithChains returns the deps over the chain file of this device.
func (d Deps) WithChains(chains ChainStore) Deps {
	d.chains = chains
	return d
}

// WithLinks returns the deps over another link in progress, so the use cases of a window and of the plugins share one.
func (d Deps) WithLinks(links *LinkStore) Deps {
	if links != nil {
		d.links = links
	}
	return d
}

func encodeRecord(record identitylib.Record) string {
	return base64.RawURLEncoding.EncodeToString(record.Bytes())
}
