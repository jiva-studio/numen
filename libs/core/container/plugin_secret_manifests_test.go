package container

import (
	"slices"
	"testing"
)

func secretGroups(t *testing.T, pluginID string) []string {
	t.Helper()
	for _, one := range getPluginManifests(t) {
		if one.ID != pluginID {
			continue
		}
		var groups []string
		for _, secret := range one.Secrets {
			if secret.Name == "apiKey" {
				groups = append(groups, secret.Group)
			}
		}
		return groups
	}
	t.Fatalf("no plugin %s in the plugins directory", pluginID)
	return nil
}

func TestThePluginsThatCallARemoteServiceDeclareOneKeyForEachProviderGroup(t *testing.T) {
	for _, id := range []string{"md.numen.engines.embeddings", "md.numen.tools.proofreading"} {
		t.Run(id, func(t *testing.T) {
			groups := secretGroups(t, id)
			slices.Sort(groups)
			if want := []string{"", "openai", "openrouter"}; !slices.Equal(groups, want) {
				t.Errorf("apiKey groups = %q, want %q", groups, want)
			}
		})
	}
}

func TestTheNumenAgentDeclaresItsKeyInTheOpenRouterGroup(t *testing.T) {
	if groups := secretGroups(t, "md.numen.agents.numen"); !slices.Equal(groups, []string{"openrouter"}) {
		t.Errorf("apiKey groups = %q, want [openrouter]", groups)
	}
}

func TestEachDeclarationOfAGroupIsActiveOnlyForItsProvider(t *testing.T) {
	for _, one := range getPluginManifests(t) {
		for _, secret := range one.Secrets {
			if secret.Name != "apiKey" || one.ID == "md.numen.agents.numen" {
				continue
			}
			if secret.When == nil || secret.When.Setting != "provider" {
				t.Errorf("%s: apiKey of group %q has no condition on provider", one.ID, secret.Group)
				continue
			}
			want := secret.Group
			if want == "" {
				want = "custom"
			}
			if !slices.Equal(secret.When.Is, []string{want}) {
				t.Errorf("%s: apiKey of group %q is active for %q, want [%s]", one.ID, secret.Group, secret.When.Is, want)
			}
		}
	}
}
