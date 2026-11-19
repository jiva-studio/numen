package container

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/sourcebridge"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/pluginhealth"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

type sourcePlugins struct {
	pluginport.PluginResolver
	pluginport.ConfigWatcher
}

type sourceBinding struct {
	Activator    bridge.Activator
	Plugins      sourcePlugins
	Descriptions *sourcebridge.Descriptions
	Health       *pluginhealth.Items
	Chunks       *note.ChunkTokens
}

func searchSources(c sourceBinding, ctx context.Context) (indexing, asking searchsourceport.WatchedSource, closer func() error, why error) {
	report := pluginhealth.NewReportSearchSource(c.Health)
	source, closeSource, err := searchOpenSource(c, ctx, func(
		ctx context.Context, activator bridge.Activator, plugins sourcebridge.Plugins, kept *sourcebridge.Descriptions,
	) (*sourcebridge.Source, error) {
		found, err := plugins.ResolveCapability(bridge.CapabilitySearchSource)
		if err != nil {
			return nil, err
		}
		return sourcebridge.OpenWatching(ctx, activator, plugins, kept, func(readErr error) { report.Execute(found.Manifest.ID, readErr) })
	})
	if err != nil {
		return nil, nil, func() error { return nil }, err
	}
	return source, source, closeSource, nil
}

func searchSource(c sourceBinding, ctx context.Context) (searchsourceport.WatchedSource, func() error, error) {
	return searchOpenSource(c, ctx, sourcebridge.Open)
}

// sourceOpener opens the search source over the plugin host, with the descriptions the host keeps.
type sourceOpener func(
	ctx context.Context, activator bridge.Activator, plugins sourcebridge.Plugins, kept *sourcebridge.Descriptions,
) (*sourcebridge.Source, error)

func searchOpenSource(c sourceBinding, ctx context.Context, open sourceOpener) (searchsourceport.WatchedSource, func() error, error) {
	if _, err := c.Plugins.ResolveCapability(bridge.CapabilitySearchSource); errors.Is(err, domain.ErrUnavailable) {
		return nil, func() error { return nil }, nil
	} else if err != nil {
		return nil, nil, err
	}
	return openSource(c, ctx, open)
}

func openSource(c sourceBinding, ctx context.Context, open sourceOpener) (searchsourceport.WatchedSource, func() error, error) {
	opened, err := open(ctx, c.Activator, c.Plugins, c.Descriptions)
	if err != nil {
		return nil, nil, err
	}
	c.Chunks.Follow(opened, opened.OnDescriptionChange)
	return opened, opened.Close, nil
}
