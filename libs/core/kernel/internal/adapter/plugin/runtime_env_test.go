package plugin

import (
	"context"
	"testing"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func TestADependencyCannotNameAVariableTheHostOwnsOrAMalformedOne(t *testing.T) {
	for _, name := range []string{"NUMEN_PLUGIN_TOKEN", "NUMEN_PLUGIN_PORT", "NUMEN_HOST_PORT", "NUMEN_VAULT_PATH", "A=B", "", "BAD NAME", "X\x00Y"} {
		t.Run(name, func(t *testing.T) {
			host := NewHost(testsupport.GetMachine(), "desktop.editor")
			t.Cleanup(func() {
				if err := host.Teardown.TeardownAll(context.WithoutCancel(t.Context()), 0); err != nil {
					t.Error(err)
				}
			})
			host.Services.SetDependencies(&fakeCatalog{resolved: map[string]plugindomain.ResolvedDependency{
				"model": {ID: "model", EnvVars: map[string]string{name: "/evil"}},
			}})
			requirements := []manifest.DependencyRequirement{{ID: "model", Kind: manifest.DependencyKindModel}}

			vars, err := host.Processes.GetRuntimeVars(t.Context(), requirements)

			if err == nil {
				t.Errorf("getRuntimeVars accepted the variable name %q: %v", name, vars)
			}
		})
	}
}
