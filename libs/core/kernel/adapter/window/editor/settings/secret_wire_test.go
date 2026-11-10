package settings

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func TestNewWirePluginInfosIncludesSecretDeclarations(t *testing.T) {
	infos := NewWirePluginInfos([]plugin.Info{{
		ID: "md.numen.sample",
		Secrets: []manifest.SecretDeclaration{
			{Name: "apiKey", Title: "API key", Description: "Key of the service", Group: "openrouter"},
			{Name: "orgId", Title: "Organization"},
		},
	}})

	secrets := infos[0].GetSecrets()
	if len(secrets) != 2 {
		t.Fatalf("traveled as %v", infos[0])
	}
	if secrets[0].GetName() != "apiKey" || secrets[0].GetGroup() != "openrouter" || secrets[0].GetDescription() != "Key of the service" {
		t.Errorf("first declaration = %v", secrets[0])
	}
	if secrets[1].GetName() != "orgId" || secrets[1].GetGroup() != "" {
		t.Errorf("second declaration = %v", secrets[1])
	}
}
