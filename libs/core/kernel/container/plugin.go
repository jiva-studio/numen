package container

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// configRefresher sends the configuration again to the plugins that were not given it yet.
type configRefresher interface {
	// Refresh configures the running plugins that were not given their configuration yet, and the
	// plugins named in forced.
	Refresh(ctx context.Context, forced ...string) error
}

// pluginBinding gives the host the mode and the vault path the plugins run under.
type pluginBinding interface {
	SetSafeMode(enabled bool)
	SetVaultPath(path string)
}

type pluginEventEmitter interface {
	EmitEvent(event string)
	// EmitFilesChanged emits the onFilesChanged event of the vault to the plugins that take a file of one of
	// the types.
	EmitFilesChanged(vaultID string, types []domain.FileType)
}

// pluginRetrier starts a plugin once, as any use of it does.
type pluginRetrier interface {
	// RetryPlugin answers why the start failed where it did.
	RetryPlugin(ctx context.Context, pluginID string) (*plugin.Failure, error)
}

type pluginSettingDeclarer interface {
	// GetPluginSettingDeclarations returns the settings the manifest of the plugin declares, and
	// false for a plugin the host does not know.
	GetPluginSettingDeclarations(pluginID string) ([]manifest.Setting, bool)
}

// pluginLifecycle stops the plugins when the host shuts down and starts the ones that stay running.
type pluginLifecycle interface {
	TeardownAll(ctx context.Context, gracePeriod time.Duration) error
	KeepAll(ctx context.Context)
}
