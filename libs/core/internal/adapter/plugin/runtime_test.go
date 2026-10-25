package plugin

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/process"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// fakeCatalog provisions the requirements whose id it holds and records what it was asked.
type fakeCatalog struct {
	pluginport.DependencyCatalog
	resolved map[string]pluginport.ResolvedDependency
	err      error
	calls    []pluginport.DependencyRequirement
}

func (f *fakeCatalog) CanProvision(req pluginport.DependencyRequirement) bool {
	_, ok := f.resolved[req.ID]
	return ok
}

func (f *fakeCatalog) Register(pluginport.DependencyRequirement) {}

func (f *fakeCatalog) Provision(_ context.Context, req pluginport.DependencyRequirement, _ func(pluginport.DependencyStatus)) (pluginport.ResolvedDependency, error) {
	f.calls = append(f.calls, req)
	return f.resolved[req.ID], f.err
}

func newONNXCatalog() *fakeCatalog {
	return &fakeCatalog{resolved: map[string]pluginport.ResolvedDependency{
		pluginport.RuntimeONNX: {ID: pluginport.RuntimeONNX, EnvVars: map[string]string{process.EnvONNXRuntimeLib: "/models/libonnxruntime.so", process.EnvONNXProvider: "coreml"}},
	}}
}

func newRuntimeRequirements(dir string) []pluginport.DependencyRequirement {
	return pluginport.PluginManifest{Runtimes: []string{pluginport.RuntimeONNX}}.GetDependencyRequirements(dir)
}

func TestThePluginEnvironmentCarriesTheRuntime(t *testing.T) {
	runtimeVars := []string{process.EnvONNXRuntimeLib + "=/models/libonnxruntime.so", process.EnvONNXProvider + "=coreml"}
	with := process.GetPluginVars("md.numen.a", process.LaunchEnvironment{Port: 5, Token: "t", RuntimeVars: runtimeVars})
	for _, want := range runtimeVars {
		if !slices.Contains(with, want) {
			t.Errorf("environment %v lacks %s", with, want)
		}
	}
	defaultVars := process.GetPluginVars("md.numen.a", process.LaunchEnvironment{Port: 5, Token: "t"})
	for _, entry := range defaultVars {
		if strings.HasPrefix(entry, "NUMEN_ONNX_") {
			t.Errorf("environment %v names a runtime the plugin did not declare", defaultVars)
		}
	}
}

func TestALaunchedPluginIsReportedTheRuntimeItDeclares(t *testing.T) {
	host := NewHost("desktop.editor")
	t.Cleanup(func() { _ = host.TeardownAll(context.WithoutCancel(t.Context()), 0) })
	catalog := newONNXCatalog()
	host.SetDependencies(catalog)

	vars, err := host.getLaunchVars(t.Context(), "md.numen.a", newRuntimeRequirements("/plugins/a"), 5, "t")
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.calls) != 1 || catalog.calls[0].PluginDir != "/plugins/a" || catalog.calls[0].Kind != pluginport.DependencyKindRuntime {
		t.Errorf("the runtime was provisioned as %+v", catalog.calls)
	}
	for _, want := range []string{"NUMEN_ONNX_RUNTIME_LIB=/models/libonnxruntime.so", "NUMEN_ONNX_PROVIDER=coreml"} {
		if !slices.Contains(vars, want) {
			t.Errorf("environment %v lacks %s", vars, want)
		}
	}
}

func TestADeclaredDependencyInjectsItsEnvironment(t *testing.T) {
	host := NewHost("desktop.editor")
	t.Cleanup(func() { _ = host.TeardownAll(context.WithoutCancel(t.Context()), 0) })
	host.SetDependencies(&fakeCatalog{resolved: map[string]pluginport.ResolvedDependency{
		"whisper": {ID: "whisper", EnvVars: map[string]string{"NUMEN_DEPENDENCY_WHISPER": "/deps/whisper.bin"}},
	}})
	manifest := pluginport.PluginManifest{Dependencies: []pluginport.DependencyRequirement{{ID: "whisper", Kind: pluginport.DependencyKindModel}}}

	vars, err := host.getLaunchVars(t.Context(), "md.numen.a", manifest.GetDependencyRequirements("/plugins/a"), 5, "t")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(vars, "NUMEN_DEPENDENCY_WHISPER=/deps/whisper.bin") {
		t.Errorf("environment %v lacks the dependency variable", vars)
	}
}

func TestAPluginThatDeclaresNoRuntimeProvisionsNothing(t *testing.T) {
	host := NewHost("desktop.editor")
	t.Cleanup(func() { _ = host.TeardownAll(context.WithoutCancel(t.Context()), 0) })
	catalog := newONNXCatalog()
	host.SetDependencies(catalog)

	vars, err := host.getLaunchVars(t.Context(), "md.numen.a", nil, 5, "t")
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.calls) != 0 {
		t.Errorf("the runtime was provisioned %d times for a plugin that declares none", len(catalog.calls))
	}
	for _, entry := range vars {
		if strings.HasPrefix(entry, "NUMEN_ONNX_") {
			t.Errorf("environment %v names a runtime the plugin did not declare", vars)
		}
	}
}

func TestAPluginIsNotLaunchedWhenItsRuntimeFailsToProvision(t *testing.T) {
	host := NewHost("desktop.editor")
	t.Cleanup(func() { _ = host.TeardownAll(context.WithoutCancel(t.Context()), 0) })
	failure := errors.New("the archive is not the pinned one")
	catalog := newONNXCatalog()
	catalog.err = failure
	host.SetDependencies(catalog)

	vars, err := host.getLaunchVars(t.Context(), "md.numen.a", newRuntimeRequirements("/plugins/a"), 5, "t")
	if !errors.Is(err, failure) || !strings.Contains(err.Error(), "md.numen.a") {
		t.Fatalf("getLaunchVars() = %v, %v, want the failure with the plugin named", vars, err)
	}
	if vars != nil {
		t.Errorf("a failed provisioning left the environment %v", vars)
	}
}

func TestAPluginIsNotLaunchedWhenNothingProvisionsItsRuntime(t *testing.T) {
	host := NewHost("desktop.editor")
	t.Cleanup(func() { _ = host.TeardownAll(context.WithoutCancel(t.Context()), 0) })

	if _, err := host.getLaunchVars(t.Context(), "md.numen.a", newRuntimeRequirements("/plugins/a"), 5, "t"); !errors.Is(err, ErrRuntimeUnavailable) {
		t.Fatalf("getLaunchVars() = %v, want ErrRuntimeUnavailable", err)
	}
}

func TestAPluginIsNotLaunchedWithARuntimeTheHostDoesNotKnow(t *testing.T) {
	host := NewHost("desktop.editor")
	t.Cleanup(func() { _ = host.TeardownAll(context.WithoutCancel(t.Context()), 0) })
	host.SetDependencies(newONNXCatalog())
	unknown := []pluginport.DependencyRequirement{{ID: "tensorrt", Kind: pluginport.DependencyKindRuntime}}

	if _, err := host.getLaunchVars(t.Context(), "md.numen.a", unknown, 5, "t"); !errors.Is(err, ErrRuntimeUnavailable) {
		t.Fatalf("getLaunchVars() = %v, want ErrRuntimeUnavailable", err)
	}
}
