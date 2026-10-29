package container

import "github.com/jiva-studio/numen/libs/core/usecase/vault"

// ErrClosing is returned for work requested after the window began closing.
var ErrClosing = vault.ErrClosing

// OpenVaults is every vault this window has open.
type OpenVaults = vault.OpenVaults[Config, *Index]
