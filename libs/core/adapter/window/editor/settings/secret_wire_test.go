package settings

import (
	"testing"

	setting "github.com/jiva-studio/numen/libs/core/usecase/settings"
)

// The group of a declaration travels with the secret a plugin asks for, and the plugin carries no value.
func TestAPluginTravelsWithTheGroupOfItsSecretDeclarations(t *testing.T) {
	infos := NewWirePluginInfos([]setting.PluginInfo{{
		ID: "md.numen.sample",
		Secrets: []setting.PluginSecret{
			{Name: "apiKey", Title: "API key", Description: "Key of the service", Group: "openrouter"},
			{Name: "orgId", Title: "Organization"},
		},
	}})

	secrets := infos[0].GetSecrets()
	if len(secrets) != 2 {
		t.Fatalf("travelled as %v", infos[0])
	}
	if secrets[0].GetName() != "apiKey" || secrets[0].GetGroup() != "openrouter" || secrets[0].GetDescription() != "Key of the service" {
		t.Errorf("first declaration = %v", secrets[0])
	}
	if secrets[1].GetName() != "orgId" || secrets[1].GetGroup() != "" {
		t.Errorf("second declaration = %v", secrets[1])
	}
}
