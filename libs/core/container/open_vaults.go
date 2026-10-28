package container

import "github.com/jiva-studio/numen/libs/core/usecase/vault"

// ErrClosing is work asked for once the window has begun closing.
var ErrClosing = vault.ErrClosing

// OpenVaults is every vault this window has open.
type OpenVaults = vault.OpenVaults[Config, *Index]
