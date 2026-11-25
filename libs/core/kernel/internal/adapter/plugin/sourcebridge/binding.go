package sourcebridge

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// CapabilityResolver finds the enabled plugin that provides a capability.
type CapabilityResolver interface {
	ResolveCapability(capability string) (manifest.DiscoveredPlugin, error)
}

// PluginAccess resolves the source provider and subscribes to its settings.
type PluginAccess struct {
	CapabilityResolver
	ConfigWatcher
}

// Binding opens the installation's source and subscribes to description changes.
type Binding struct {
	Activator    bridge.Activator
	Plugins      PluginAccess
	Descriptions *Descriptions
	ReportRead   func(string, error)
	Follow       func(searchsourceport.Source, func(func()))
}

// Open resolves the configured source and returns its initial description error to the caller.
func (b Binding) Open(ctx context.Context) (*Source, func() error, error) {
	return b.open(func(string) (*Source, error) { return Open(ctx, b.Activator, b.Plugins, b.Descriptions) })
}

// OpenWatching reports description errors as the source's configuration changes.
func (b Binding) OpenWatching(ctx context.Context) (*Source, func() error, error) {
	source, closeSource, err := b.open(func(pluginID string) (*Source, error) {
		return OpenWatching(ctx, b.Activator, b.Plugins, b.Descriptions, func(err error) { b.ReportRead(pluginID, err) })
	})
	if err != nil {
		return nil, func() error { return nil }, err
	}
	return source, closeSource, nil
}

func (b Binding) open(openSource func(string) (*Source, error)) (*Source, func() error, error) {
	found, err := b.Plugins.ResolveCapability(bridge.CapabilitySearchSource)
	if errors.Is(err, domain.ErrUnavailable) {
		return nil, func() error { return nil }, nil
	}
	if err != nil {
		return nil, nil, err
	}
	source, err := openSource(found.Manifest.ID)
	if err != nil {
		return nil, nil, err
	}
	b.Follow(source, source.WatchDescription)
	return source, source.Close, nil
}
