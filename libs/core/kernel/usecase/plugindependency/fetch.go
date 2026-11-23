// Package plugindependency provisions files requested by a running plugin.
package plugindependency

import (
	"context"
	"errors"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// Fetch resolves model files through the installation's current dependency registry.
type Fetch struct {
	getRegistry func() pluginport.DependencyRegistry
}

func NewFetch(registry func() pluginport.DependencyRegistry) *Fetch {
	return &Fetch{getRegistry: registry}
}

func (f *Fetch) Execute(ctx context.Context, address, digest string) (string, error) {
	req, err := manifest.NewModelFile(address, strings.ToLower(digest))
	if err != nil {
		return "", err
	}
	registry := f.getRegistry()
	if registry == nil {
		return "", errors.New("the installation has no dependency registry")
	}
	file, err := registry.Provision(ctx, req, nil)
	return file.PrimaryPath, err
}
