// Package license orchestrates listing, activating, deactivating, verifying and watching the
// licenses that plugins declare.
package license

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"golang.org/x/sync/singleflight"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/license"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/licenseport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
)

// operation holds the collaborators shared across the license use cases.
type operation struct {
	declarations licenseport.Declarations
	keys         pluginport.KeyStore
	validator    licenseport.Validator
	machine      licenseport.MachineIdentity
	clock        machineport.Clock
	changes      licenseport.ChangeHub
	gate         *sync.Mutex
	flights      *singleflight.Group
}

// Deps groups the collaborators the license use cases share.
type Deps struct {
	base operation
}

// NewDeps constructs the shared license use case dependencies.
func NewDeps(
	declarations licenseport.Declarations,
	keys pluginport.KeyStore,
	validator licenseport.Validator,
	machine licenseport.MachineIdentity,
	clock machineport.Clock,
	changes licenseport.ChangeHub,
) Deps {
	return Deps{
		base: operation{
			declarations: declarations,
			keys:         keys,
			validator:    validator,
			machine:      machine,
			clock:        clock,
			changes:      changes,
			gate:         &sync.Mutex{},
			flights:      &singleflight.Group{},
		},
	}
}

func (o operation) listDeclarations() []license.Declaration {
	if o.declarations == nil {
		return nil
	}
	discovered := o.declarations.GetPlugins()
	entries := make([]license.PluginEntry, 0, len(discovered))
	for _, p := range discovered {
		if len(p.Manifest.Contributes.Licenses) == 0 {
			continue
		}
		entries = append(entries, license.PluginEntry{
			ID:       p.Manifest.ID,
			Name:     p.Manifest.Name,
			Licenses: p.Manifest.Contributes.Licenses,
		})
	}
	return license.MergeDeclarations(entries)
}

func (o operation) findDeclaration(licenseID string) (license.Declaration, error) {
	for _, decl := range o.listDeclarations() {
		if decl.ID == licenseID {
			return decl, nil
		}
	}
	return license.Declaration{}, fmt.Errorf("%w: %s", license.ErrNotFound, licenseID)
}

func (o operation) readEnvelope(ctx context.Context, licenseID string) (license.Envelope, bool, error) {
	if o.keys == nil {
		return license.Envelope{}, false, nil
	}
	raw, err := o.keys.GetKey(ctx, license.GetEntryName(licenseID))
	if errors.Is(err, plugindomain.ErrNoKey) {
		return license.Envelope{}, false, nil
	}
	if err != nil {
		return license.Envelope{}, false, fmt.Errorf("%w: read %s: %w", license.ErrKeyStore, licenseID, err)
	}
	env, decodeErr := license.DecodeEnvelope(raw)
	if decodeErr != nil {
		return license.Envelope{}, false, nil //nolint:nilerr // Corrupted envelope is treated as absent so the user can reactivate.
	}
	return env, true, nil
}

func (o operation) writeEnvelope(ctx context.Context, licenseID string, env license.Envelope) error {
	if o.keys == nil {
		return fmt.Errorf("%w: %w", license.ErrKeyStore, plugindomain.ErrNoKeyStore)
	}
	encoded, err := license.EncodeEnvelope(env)
	if err != nil {
		return fmt.Errorf("%w: encode %s: %w", license.ErrKeyStore, licenseID, err)
	}
	if err := o.keys.SetKey(ctx, license.GetEntryName(licenseID), encoded); err != nil {
		return fmt.Errorf("%w: write %s: %w", license.ErrKeyStore, licenseID, err)
	}
	return nil
}

func (o operation) deleteEnvelope(ctx context.Context, licenseID string) error {
	if o.keys == nil {
		return nil
	}
	err := o.keys.DeleteKey(ctx, license.GetEntryName(licenseID))
	if err != nil && !errors.Is(err, plugindomain.ErrNoKey) {
		return fmt.Errorf("%w: delete %s: %w", license.ErrKeyStore, licenseID, err)
	}
	return nil
}

func (o operation) buildInfo(ctx context.Context, decl license.Declaration) (license.Info, bool, error) {
	var options []license.ProviderOption
	if o.validator != nil && len(decl.Providers) > 0 {
		options = o.validator.ListProviderOptions(decl.Providers)
	}
	info := license.Info{
		ID:           decl.ID,
		Title:        decl.Title,
		Contributors: decl.Contributors,
		Providers:    options,
		Status:       license.StatusInactive,
	}
	env, isFound, err := o.readEnvelope(ctx, decl.ID)
	if err != nil || !isFound {
		return info, false, err
	}
	fingerprint := ""
	if o.machine != nil {
		fingerprint = o.machine.GetMachineFingerprint()
	}
	now := o.clock.Now().UTC()
	eval := license.EvaluateEnvelope(env, decl, fingerprint, now)
	if eval.EnvelopeChanged && o.keys != nil {
		if writeErr := o.writeEnvelope(ctx, decl.ID, eval.UpdatedEnvelope); writeErr != nil {
			return info, false, writeErr
		}
	}
	if !eval.HasKey {
		return info, false, nil
	}
	info.Status = eval.Status
	info.IsActive = eval.IsActive
	info.HasKey = true
	info.KeyHint = eval.UpdatedEnvelope.KeyHint
	if info.KeyHint == "" {
		info.KeyHint = license.GetKeyHint(eval.UpdatedEnvelope.Key)
	}
	info.ActiveProvider = eval.UpdatedEnvelope.Provider
	info.InstanceName = eval.UpdatedEnvelope.InstanceName
	info.ExpiresAt = eval.UpdatedEnvelope.ExpiresAt
	info.VerifiedAt = eval.UpdatedEnvelope.VerifiedAt
	return info, eval.NeedsValidation, nil
}

func (o operation) notifyChange(ctx context.Context) {
	if o.changes != nil {
		o.changes.NotifyChange(ctx)
	}
}
