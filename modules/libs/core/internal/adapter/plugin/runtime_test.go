package plugin

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// fakeCatalog provisions the requirements whose id it holds and records what it was asked.
type fakeCatalog struct {
	port.DependencyCatalog
	resolved map[string]port.ResolvedDependency
	err      error
	calls    []port.DependencyRequirement
}

func (f *fakeCatalog) CanProvision(req port.DependencyRequirement) bool {
	_, ok := f.resolved[req.ID]
	return ok
}

func (f *fakeCatalog) Register(port.DependencyRequirement) {}

func (f *fakeCatalog) Provision(_ context.Context, req port.DependencyRequirement, _ func(port.DependencyStatus)) (port.ResolvedDependency, error) {
	f.calls = append(f.calls, req)
	return f.resolved[req.ID], f.err
}

func newONNXCatalog() *fakeCatalog {
	return &fakeCatalog{resolved: map[string]port.ResolvedDependency{
		port.RuntimeONNX: {ID: port.RuntimeONNX, EnvVars: map[string]string{EnvONNXRuntimeLib: "/models/libonnxruntime.so", EnvONNXProvider: "coreml"}},
	}}
}

func newRuntimeRequirements(dir string) []port.DependencyRequirement {
	return port.PluginManifest{Runtimes: []string{port.RuntimeONNX}}.GetDependencyRequirements(dir)
}

func TestThePluginEnvironmentCarriesTheRuntime(t *testing.T) {
	runtimeVars := []string{EnvONNXRuntimeLib + "=/models/libonnxruntime.so", EnvONNXProvider + "=coreml"}
	with := getPluginVars("md.numen.a", launchEnvironment{port: 5, token: "t", runtimeVars: runtimeVars})
	for _, want := range runtimeVars {
		if !slices.Contains(with, want) {
			t.Errorf("environment %v lacks %s", with, want)
		}
	}
	defaultVars := getPluginVars("md.numen.a", launchEnvironment{port: 5, token: "t"})
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
	if len(catalog.calls) != 1 || catalog.calls[0].PluginDir != "/plugins/a" || catalog.calls[0].Kind != port.DependencyKindRuntime {
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
	host.SetDependencies(&fakeCatalog{resolved: map[string]port.ResolvedDependency{
		"whisper": {ID: "whisper", EnvVars: map[string]string{"NUMEN_DEPENDENCY_WHISPER": "/deps/whisper.bin"}},
	}})
	manifest := port.PluginManifest{Dependencies: []port.DependencyRequirement{{ID: "whisper", Kind: port.DependencyKindModel}}}

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
	unknown := []port.DependencyRequirement{{ID: "tensorrt", Kind: port.DependencyKindRuntime}}

	if _, err := host.getLaunchVars(t.Context(), "md.numen.a", unknown, 5, "t"); !errors.Is(err, ErrRuntimeUnavailable) {
		t.Fatalf("getLaunchVars() = %v, want ErrRuntimeUnavailable", err)
	}
}
