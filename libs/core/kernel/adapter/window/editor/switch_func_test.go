package editor_test

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
	"github.com/jiva-studio/numen/libs/note/domain"
)

var _ windowsession.VaultSwitch = switchFunc(nil)

// switchFunc is a vault switch written as a function.
type switchFunc func(ctx context.Context, v domain.Vault, open *windowsession.PluginOpen) error

func (f switchFunc) Execute(ctx context.Context, v domain.Vault, open *windowsession.PluginOpen) error {
	return f(ctx, v, open)
}
