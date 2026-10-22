package plugin

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port/pluginport"
)

func TestADependencyCannotNameAVariableTheHostOwnsOrAMalformedOne(t *testing.T) {
	for _, name := range []string{"NUMEN_PLUGIN_TOKEN", "NUMEN_PLUGIN_PORT", "NUMEN_HOST_PORT", "NUMEN_VAULT_PATH", "A=B", "", "BAD NAME", "X\x00Y"} {
		t.Run(name, func(t *testing.T) {
			host := NewHost("desktop.editor")
			t.Cleanup(func() { _ = host.TeardownAll(t.Context(), 0) })
			host.SetDependencies(&fakeCatalog{resolved: map[string]pluginport.ResolvedDependency{
				"model": {ID: "model", EnvVars: map[string]string{name: "/evil"}},
			}})
			requirements := []pluginport.DependencyRequirement{{ID: "model", Kind: pluginport.DependencyKindModel}}

			vars, err := host.getRuntimeVars(t.Context(), requirements)

			if err == nil {
				t.Errorf("getRuntimeVars accepted the variable name %q: %v", name, vars)
			}
		})
	}
}
