package plugin

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

type recordingProvisioner struct {
	library, provider string
	err               error
	calls             int
	dir               string
}

func (r *recordingProvisioner) Provision(_ context.Context, dir string) (string, string, error) {
	r.calls++
	r.dir = dir
	return r.library, r.provider, r.err
}

func TestThePluginEnvironmentCarriesTheRuntime(t *testing.T) {
	runtimeVars := []string{EnvONNXRuntimeLib + "=/models/libonnxruntime.so", EnvONNXProvider + "=coreml"}
	with := getPluginVars("md.numen.a", launchEnvironment{port: 5, token: "t", runtimeVars: runtimeVars})
	for _, want := range runtimeVars {
		if !slices.Contains(with, want) {
			t.Errorf("environment %v lacks %s", with, want)
		}
	}
	without := getPluginVars("md.numen.a", launchEnvironment{port: 5, token: "t"})
	for _, entry := range without {
		if strings.HasPrefix(entry, "NUMEN_ONNX_") {
			t.Errorf("environment %v names a runtime the plugin did not declare", without)
		}
	}
}

func TestALaunchedPluginIsToldTheRuntimeItDeclares(t *testing.T) {
	host := NewHost("desktop.editor")
	t.Cleanup(func() { _ = host.TeardownAll(context.WithoutCancel(t.Context()), 0) })
	provisioner := &recordingProvisioner{library: "/models/libonnxruntime.so", provider: "coreml"}
	host.SetRuntimeProvisioner(provisioner)

	vars, err := host.getLaunchVars(t.Context(), "md.numen.a", "/plugins/a", []string{port.RuntimeONNX}, 5, "t")
	if err != nil {
		t.Fatal(err)
	}
	if provisioner.dir != "/plugins/a" {
		t.Errorf("the runtime was provisioned for %q", provisioner.dir)
	}
	for _, want := range []string{"NUMEN_ONNX_RUNTIME_LIB=/models/libonnxruntime.so", "NUMEN_ONNX_PROVIDER=coreml"} {
		if !slices.Contains(vars, want) {
			t.Errorf("environment %v lacks %s", vars, want)
		}
	}
}

func TestAPluginThatDeclaresNoRuntimeProvisionsNothing(t *testing.T) {
	host := NewHost("desktop.editor")
	t.Cleanup(func() { _ = host.TeardownAll(context.WithoutCancel(t.Context()), 0) })
	provisioner := &recordingProvisioner{library: "/models/libonnxruntime.so", provider: "cpu"}
	host.SetRuntimeProvisioner(provisioner)

	vars, err := host.getLaunchVars(t.Context(), "md.numen.a", "", nil, 5, "t")
	if err != nil {
		t.Fatal(err)
	}
	if provisioner.calls != 0 {
		t.Errorf("the runtime was provisioned %d times for a plugin that declares none", provisioner.calls)
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
	host.SetRuntimeProvisioner(&recordingProvisioner{err: failure})

	vars, err := host.getLaunchVars(t.Context(), "md.numen.a", "/plugins/a", []string{port.RuntimeONNX}, 5, "t")
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

	if _, err := host.getLaunchVars(t.Context(), "md.numen.a", "/plugins/a", []string{port.RuntimeONNX}, 5, "t"); !errors.Is(err, ErrRuntimeUnavailable) {
		t.Fatalf("getLaunchVars() = %v, want ErrRuntimeUnavailable", err)
	}
}

func TestAPluginIsNotLaunchedWithARuntimeTheHostDoesNotKnow(t *testing.T) {
	host := NewHost("desktop.editor")
	t.Cleanup(func() { _ = host.TeardownAll(context.WithoutCancel(t.Context()), 0) })
	host.SetRuntimeProvisioner(&recordingProvisioner{library: "/l", provider: "cpu"})

	if _, err := host.getLaunchVars(t.Context(), "md.numen.a", "/plugins/a", []string{"tensorrt"}, 5, "t"); !errors.Is(err, ErrRuntimeUnavailable) {
		t.Fatalf("getLaunchVars() = %v, want ErrRuntimeUnavailable", err)
	}
}
