package plugindependency

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// ErrUndeclaredDependency is a model a plugin asks for that its manifest does not declare.
var ErrUndeclaredDependency = errors.New("the plugin declares no such model")

// DeclaredModels finds the models that the manifest of a plugin declares.
type DeclaredModels interface {
	FindModelDependency(pluginID, id string) (manifest.DependencyRequirement, bool)
}

// FetchDeclaredModel resolves a model that the manifest of the calling plugin declares, fetching it when it
// is not on this machine.
type FetchDeclaredModel struct {
	plugins     DeclaredModels
	getRegistry func() pluginport.DependencyRegistry
}

func NewFetchDeclaredModel(plugins DeclaredModels, registry func() pluginport.DependencyRegistry) *FetchDeclaredModel {
	return &FetchDeclaredModel{plugins: plugins, getRegistry: registry}
}

// Execute returns the path of the model that the manifest of pluginID declares under id. It returns
// ErrUndeclaredDependency, and fetches nothing, for an id the manifest does not declare as a model.
func (f *FetchDeclaredModel) Execute(ctx context.Context, pluginID, id string) (string, error) {
	declared, ok := f.plugins.FindModelDependency(pluginID, id)
	if !ok {
		return "", fmt.Errorf("%w: %s asks for %q", ErrUndeclaredDependency, pluginID, id)
	}
	req, err := manifest.NewDeclaredModelFile(declared)
	if err != nil {
		return "", err
	}
	registry := f.getRegistry()
	if registry == nil {
		return "", errors.New("the installation has no dependency registry")
	}
	file, err := registry.Provision(ctx, req, nil)
	if err != nil {
		return "", err
	}
	return file.PrimaryPath, nil
}
