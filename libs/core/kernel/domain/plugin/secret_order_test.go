package plugin_test

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

func idsOf(list []plugin.Secret) []string {
	var ids []string
	for _, one := range list {
		ids = append(ids, one.ID)
	}
	return ids
}

func TestListDeclaredGroupsSortsUniqueGroups(t *testing.T) {
	got := plugin.ListAskedSecretGroups([]plugin.DeclarationKey{
		{Plugin: "a.b", Name: "key", Group: "openrouter"},
		{Plugin: "a.b", Name: "other"},
		{Plugin: "c.d", Name: "key", Group: "Zeta"},
		{Plugin: "c.d", Name: "key2", Group: "openrouter"},
	})
	if want := []string{"Zeta", "openrouter"}; !slices.Equal(got, want) {
		t.Errorf("groups = %v, want %v", got, want)
	}
}

func TestListInPageOrderOrdersDeclaredGroupsFirst(t *testing.T) {
	list := []plugin.Secret{
		{ID: "mine-b", Group: "work"},
		{ID: "openrouter-1", Group: "openrouter"},
		{ID: "mine-a", Group: "home"},
		{ID: "openai-1", Group: "openai"},
		{ID: "openrouter-2", Group: "openrouter"},
	}
	got := idsOf(plugin.ListSecretsInPageOrder(list, []string{"openai", "openrouter"}))
	want := []string{"openai-1", "openrouter-1", "openrouter-2", "mine-a", "mine-b"}
	if !slices.Equal(got, want) {
		t.Errorf("page = %v, want %v", got, want)
	}
}

func TestThePageCompareNamesByByteNotByCase(t *testing.T) {
	list := []plugin.Secret{{ID: "lower", Group: "apple"}, {ID: "upper", Group: "Zebra"}}
	if got := idsOf(plugin.ListSecretsInPageOrder(list, nil)); !slices.Equal(got, []string{"upper", "lower"}) {
		t.Errorf("page = %v, want the uppercase group first", got)
	}
}

func TestListAllowedOrdersDeclaredGroupFirst(t *testing.T) {
	declaration := plugin.DeclarationKey{Plugin: "a.b", Name: "key", Group: "openrouter"}
	other := plugin.DeclarationKey{Plugin: "a.b", Name: "key", Group: ""}
	page := []plugin.Secret{
		{ID: "z", Group: "openai", Allowed: []plugin.DeclarationKey{declaration}},
		{ID: "x", Group: "openrouter", Allowed: []plugin.DeclarationKey{declaration}},
		{ID: "hidden", Group: "openrouter", Allowed: []plugin.DeclarationKey{other}},
		{ID: "y", Group: "openrouter", Allowed: []plugin.DeclarationKey{declaration}},
	}
	if got := idsOf(plugin.ListAllowedSecrets(page, declaration)); !slices.Equal(got, []string{"x", "y", "z"}) {
		t.Errorf("list = %v, want x y z", got)
	}
	if got := idsOf(plugin.ListAllowedSecrets(page, other)); !slices.Equal(got, []string{"hidden"}) {
		t.Errorf("list for the declaration with no group = %v, want only the one allowed to it", got)
	}
}

func TestListAllowedWithNoGroupKeepsPageOrder(t *testing.T) {
	declaration := plugin.DeclarationKey{Plugin: "a.b", Name: "key"}
	page := []plugin.Secret{
		{ID: "first", Group: "openai", Allowed: []plugin.DeclarationKey{declaration}},
		{ID: "second", Group: "openrouter", Allowed: []plugin.DeclarationKey{declaration}},
	}
	if got := idsOf(plugin.ListAllowedSecrets(page, declaration)); !slices.Equal(got, []string{"first", "second"}) {
		t.Errorf("list = %v, want the order of the page", got)
	}
}

func TestThePageComparesNamesByTheirUTF8BytesNotByUTF16Units(t *testing.T) {
	list := []plugin.Secret{
		{ID: "s1", Group: "\U0001D49C"}, {ID: "s2", Group: "ｱ"}, {ID: "s3", Group: "Z"}, {ID: "s4", Group: "a"},
	}
	if got := idsOf(plugin.ListSecretsInPageOrder(list, nil)); !slices.Equal(got, []string{"s3", "s4", "s2", "s1"}) {
		t.Errorf("page = %v, want s3 s4 s2 s1", got)
	}
}

func TestThePageKeepsTheOrderOfTheListInsideAGroup(t *testing.T) {
	list := []plugin.Secret{{ID: "s1", Group: "openai"}, {ID: "s2", Group: "other"}, {ID: "s3", Group: "openai"}}
	if got := idsOf(plugin.ListSecretsInPageOrder(list, []string{"openai"})); !slices.Equal(got, []string{"s1", "s3", "s2"}) {
		t.Errorf("page = %v, want s1 s3 s2", got)
	}
}

func TestListSectionsIncludesEmptyDeclaredGroups(t *testing.T) {
	list := []plugin.Secret{{ID: "s1", Group: "mine"}, {ID: "s2", Group: "openai"}}
	got := plugin.ListSecretSections(list, []string{"anthropic", "openai"})
	if len(got) != 3 {
		t.Fatalf("sections = %+v", got)
	}
	if got[0].Group != "anthropic" || !got[0].IsAsked || len(got[0].Secrets) != 0 {
		t.Errorf("first section = %+v, want the asked group with no secret", got[0])
	}
	if got[1].Group != "openai" || !got[1].IsAsked || got[2].Group != "mine" || got[2].IsAsked {
		t.Errorf("sections = %+v", got)
	}
}

func TestListMenuOrdersDeclaredGroupFirst(t *testing.T) {
	list := []plugin.Secret{
		{ID: "mine", Group: "home"}, {ID: "a1", Group: "openai"}, {ID: "r1", Group: "openrouter"}, {ID: "r2", Group: "openrouter"},
	}
	page := plugin.ListSecretSections(list, []string{"openai", "openrouter"})
	declaration := plugin.DeclarationKey{Plugin: "a.b", Name: "key", Group: "openrouter"}
	if got := idsOf(plugin.ListSecretMenu(page, declaration)); !slices.Equal(got, []string{"r1", "r2", "a1", "mine"}) {
		t.Errorf("menu = %v", got)
	}
	none := plugin.DeclarationKey{Plugin: "a.b", Name: "key"}
	if got := idsOf(plugin.ListSecretMenu(page, none)); !slices.Equal(got, []string{"a1", "r1", "r2", "mine"}) {
		t.Errorf("menu with no group = %v", got)
	}
}
