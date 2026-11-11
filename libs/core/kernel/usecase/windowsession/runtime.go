package windowsession

import (
	"context"
	"sync/atomic"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// Runtime is what every scenario of one window reads and writes: the vault shown, how far the passes
// behind it got and the model the vault is read under. The zero value is a window that shows no vault.
type Runtime struct {
	vault atomic.Pointer[domain.Vault]

	isReady      atomic.Bool
	failure      reason
	watchFailure reason

	model  atomic.Value
	recipe atomic.Value
}

// runningVault is what ends the passes behind one vault.
type runningVault struct {
	retire func()
	cancel context.CancelFunc
	wait   func()
}

// reason is why the window is degraded, and the empty string while it is not.
type reason struct{ why atomic.Value }

func (r *reason) store(why string) { r.why.Store(why) }

func (r *reason) load() string {
	why, isText := r.why.Load().(string)
	if !isText {
		return ""
	}
	return why
}

// GetCurrentVault is the vault the window has open. A window standing on nothing answers with no
// vault at all.
func (r *Runtime) GetCurrentVault() domain.Vault {
	if v := r.vault.Load(); v != nil {
		return *v
	}
	return domain.Vault{}
}

func (r *Runtime) SetVault(v domain.Vault) { r.vault.Store(&v) }

// IsReady is whether the first read of the vault is done.
func (r *Runtime) IsReady() bool { return r.isReady.Load() }

// SetReady sets whether the first read of the vault is done.
func (r *Runtime) SetReady(isReady bool) { r.isReady.Store(isReady) }

// GetError is what stopped the passes behind the vault, and the empty string when nothing did.
func (r *Runtime) GetError() string { return r.failure.load() }

// SetError records what stopped the passes behind the vault, or the empty string for nothing.
func (r *Runtime) SetError(why string) { r.failure.store(why) }

// GetWatchError is why the vault is not followed, and the empty string while it is.
func (r *Runtime) GetWatchError() string { return r.watchFailure.load() }

// SetWatchError records why the vault is not followed, or the empty string for nothing.
func (r *Runtime) SetWatchError(why string) { r.watchFailure.store(why) }

// RecordModel sets the model the vault is read under and the recipe its vectors are made by.
func (r *Runtime) RecordModel(name, recipe string) {
	r.model.Store(name)
	r.recipe.Store(recipe)
}

// GetModel is the model the vault is read under and its recipe, and nothing before one is recorded.
func (r *Runtime) GetModel() (name, recipe string) {
	return getText(&r.model), getText(&r.recipe)
}

func getText(v *atomic.Value) string {
	text, isText := v.Load().(string)
	if !isText {
		return ""
	}
	return text
}
