package plugin_test

import (
	"context"
	"errors"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/pluginhost"
	"slices"
	"strings"
	"testing"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/process"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// fakeCatalog provisions the requirements whose id it holds and records the requests.
type fakeCatalog struct {
	resolved map[string]plugindomain.ResolvedDependency
	err      error
	calls    []manifest.DependencyRequirement
}

func (f *fakeCatalog) CanProvision(req manifest.DependencyRequirement) bool {
	_, ok := f.resolved[req.ID]
	return ok
}

func (f *fakeCatalog) GetStatus(context.Context, string) (plugindomain.DependencyStatus, error) {
	return plugindomain.DependencyStatus{}, nil
}

func (f *fakeCatalog) List(context.Context) ([]plugindomain.DependencyStatus, error) { return nil, nil }

func (f *fakeCatalog) SetObserver(func(manifest.DependencyRequirement, plugindomain.DependencyStatus)) {
}

func (f *fakeCatalog) Provision(_ context.Context, req manifest.DependencyRequirement, _ func(plugindomain.DependencyStatus)) (plugindomain.ResolvedDependency, error) {
	f.calls = append(f.calls, req)
	return f.resolved[req.ID], f.err
}

func newONNXCatalog() *fakeCatalog {
	return &fakeCatalog{resolved: map[string]plugindomain.ResolvedDependency{
		plugindomain.RuntimeONNX: {ID: plugindomain.RuntimeONNX, EnvVars: map[string]string{process.EnvONNXRuntimeLib: "/models/libonnxruntime.so", process.EnvONNXProvider: "coreml"}},
	}}
}

func newRuntimeRequirements(dir string) []manifest.DependencyRequirement {
	return manifest.Manifest{Dependencies: []manifest.DependencyRequirement{
		{ID: plugindomain.RuntimeONNX, Kind: manifest.DependencyKindRuntime},
	}}.GetDependencyRequirements(dir, nil)
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
	host := pluginhost.New(testsupport.GetMachine(), "desktop.editor")
	t.Cleanup(func() {
		if err := host.Teardown.TeardownAll(context.WithoutCancel(t.Context()), 0); err != nil {
			t.Error(err)
		}
	})
	catalog := newONNXCatalog()
	host.Services.SetDependencies(catalog)

	vars, err := host.Processes.GetLaunchVars(t.Context(), "md.numen.a", newRuntimeRequirements("/plugins/a"), host.Mode.GetVaultPath(), 5, "t")
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.calls) != 1 || catalog.calls[0].PluginDir != "/plugins/a" || catalog.calls[0].Kind != manifest.DependencyKindRuntime {
		t.Errorf("the runtime was provisioned as %+v", catalog.calls)
	}
	for _, want := range []string{"NUMEN_ONNX_RUNTIME_LIB=/models/libonnxruntime.so", "NUMEN_ONNX_PROVIDER=coreml"} {
		if !slices.Contains(vars, want) {
			t.Errorf("environment %v lacks %s", vars, want)
		}
	}
}

func TestADeclaredDependencyInjectsItsEnvironment(t *testing.T) {
	host := pluginhost.New(testsupport.GetMachine(), "desktop.editor")
	t.Cleanup(func() {
		if err := host.Teardown.TeardownAll(context.WithoutCancel(t.Context()), 0); err != nil {
			t.Error(err)
		}
	})
	host.Services.SetDependencies(&fakeCatalog{resolved: map[string]plugindomain.ResolvedDependency{
		"whisper": {ID: "whisper", EnvVars: map[string]string{"NUMEN_DEPENDENCY_WHISPER": "/deps/whisper.bin"}},
	}})
	manifest := manifest.Manifest{Dependencies: []manifest.DependencyRequirement{{ID: "whisper", Kind: manifest.DependencyKindModel}}}

	vars, err := host.Processes.GetLaunchVars(t.Context(), "md.numen.a", manifest.GetDependencyRequirements("/plugins/a", nil), host.Mode.GetVaultPath(), 5, "t")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(vars, "NUMEN_DEPENDENCY_WHISPER=/deps/whisper.bin") {
		t.Errorf("environment %v lacks the dependency variable", vars)
	}
}

func TestAPluginThatDeclaresNoRuntimeProvisionsNothing(t *testing.T) {
	host := pluginhost.New(testsupport.GetMachine(), "desktop.editor")
	t.Cleanup(func() {
		if err := host.Teardown.TeardownAll(context.WithoutCancel(t.Context()), 0); err != nil {
			t.Error(err)
		}
	})
	catalog := newONNXCatalog()
	host.Services.SetDependencies(catalog)

	vars, err := host.Processes.GetLaunchVars(t.Context(), "md.numen.a", nil, host.Mode.GetVaultPath(), 5, "t")
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
	host := pluginhost.New(testsupport.GetMachine(), "desktop.editor")
	t.Cleanup(func() {
		if err := host.Teardown.TeardownAll(context.WithoutCancel(t.Context()), 0); err != nil {
			t.Error(err)
		}
	})
	failure := errors.New("the archive is not the pinned one")
	catalog := newONNXCatalog()
	catalog.err = failure
	host.Services.SetDependencies(catalog)

	vars, err := host.Processes.GetLaunchVars(t.Context(), "md.numen.a", newRuntimeRequirements("/plugins/a"), host.Mode.GetVaultPath(), 5, "t")
	if !errors.Is(err, failure) || !strings.Contains(err.Error(), "md.numen.a") {
		t.Fatalf("getLaunchVars() = %v, %v, want the failure with the plugin named", vars, err)
	}
	if vars != nil {
		t.Errorf("a failed provisioning left the environment %v", vars)
	}
}

func TestAPluginIsNotLaunchedWhenNothingProvisionsItsRuntime(t *testing.T) {
	host := pluginhost.New(testsupport.GetMachine(), "desktop.editor")
	t.Cleanup(func() {
		if err := host.Teardown.TeardownAll(context.WithoutCancel(t.Context()), 0); err != nil {
			t.Error(err)
		}
	})

	if _, err := host.Processes.GetLaunchVars(t.Context(), "md.numen.a", newRuntimeRequirements("/plugins/a"), host.Mode.GetVaultPath(), 5, "t"); !errors.Is(err, supervision.ErrRuntimeUnavailable) {
		t.Fatalf("getLaunchVars() = %v, want supervision.ErrRuntimeUnavailable", err)
	}
}

func TestAPluginIsNotLaunchedWithARuntimeTheHostDoesNotKnow(t *testing.T) {
	host := pluginhost.New(testsupport.GetMachine(), "desktop.editor")
	t.Cleanup(func() {
		if err := host.Teardown.TeardownAll(context.WithoutCancel(t.Context()), 0); err != nil {
			t.Error(err)
		}
	})
	host.Services.SetDependencies(newONNXCatalog())
	unknown := []manifest.DependencyRequirement{{ID: "tensorrt", Kind: manifest.DependencyKindRuntime}}

	if _, err := host.Processes.GetLaunchVars(t.Context(), "md.numen.a", unknown, host.Mode.GetVaultPath(), 5, "t"); !errors.Is(err, supervision.ErrRuntimeUnavailable) {
		t.Fatalf("getLaunchVars() = %v, want supervision.ErrRuntimeUnavailable", err)
	}
}
