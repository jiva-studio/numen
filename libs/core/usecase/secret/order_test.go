package secret_test

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/usecase/secret"
)

func idsOf(list []secret.Secret) []string {
	var ids []string
	for _, one := range list {
		ids = append(ids, one.ID)
	}
	return ids
}

func TestAskedGroupsAreOnceEachInByteOrderAndLeaveOutNoGroup(t *testing.T) {
	got := secret.ListAskedGroups([]secret.Permission{
		{Plugin: "a.b", Name: "key", Group: "openrouter"},
		{Plugin: "a.b", Name: "other"},
		{Plugin: "c.d", Name: "key", Group: "Zeta"},
		{Plugin: "c.d", Name: "key2", Group: "openrouter"},
	})
	if want := []string{"Zeta", "openrouter"}; !slices.Equal(got, want) {
		t.Errorf("groups = %v, want %v", got, want)
	}
}

func TestThePageListsAskedGroupsByNameThenTheGroupsOfThePersonByName(t *testing.T) {
	list := []secret.Secret{
		{ID: "mine-b", Group: "work"},
		{ID: "openrouter-1", Group: "openrouter"},
		{ID: "mine-a", Group: "home"},
		{ID: "openai-1", Group: "openai"},
		{ID: "openrouter-2", Group: "openrouter"},
	}
	got := idsOf(secret.ListInPageOrder(list, []string{"openai", "openrouter"}))
	want := []string{"openai-1", "openrouter-1", "openrouter-2", "mine-a", "mine-b"}
	if !slices.Equal(got, want) {
		t.Errorf("page = %v, want %v", got, want)
	}
}

func TestThePageCompareNamesByByteNotByCase(t *testing.T) {
	list := []secret.Secret{{ID: "lower", Group: "apple"}, {ID: "upper", Group: "Zebra"}}
	if got := idsOf(secret.ListInPageOrder(list, nil)); !slices.Equal(got, []string{"upper", "lower"}) {
		t.Errorf("page = %v, want the uppercase group first", got)
	}
}

func TestADeclarationReceivesOnlyTheSecretsAllowedToItWithItsGroupFirst(t *testing.T) {
	declaration := secret.Permission{Plugin: "a.b", Name: "key", Group: "openrouter"}
	other := secret.Permission{Plugin: "a.b", Name: "key", Group: ""}
	page := []secret.Secret{
		{ID: "z", Group: "openai", Allowed: []secret.Permission{declaration}},
		{ID: "x", Group: "openrouter", Allowed: []secret.Permission{declaration}},
		{ID: "hidden", Group: "openrouter", Allowed: []secret.Permission{other}},
		{ID: "y", Group: "openrouter", Allowed: []secret.Permission{declaration}},
	}
	if got := idsOf(secret.ListAllowed(page, declaration)); !slices.Equal(got, []string{"x", "y", "z"}) {
		t.Errorf("list = %v, want x y z", got)
	}
	if got := idsOf(secret.ListAllowed(page, other)); !slices.Equal(got, []string{"hidden"}) {
		t.Errorf("list for the declaration with no group = %v, want only the one allowed to it", got)
	}
}

func TestADeclarationWithNoGroupReceivesTheAllowedSecretsInTheOrderOfThePage(t *testing.T) {
	declaration := secret.Permission{Plugin: "a.b", Name: "key"}
	page := []secret.Secret{
		{ID: "first", Group: "openai", Allowed: []secret.Permission{declaration}},
		{ID: "second", Group: "openrouter", Allowed: []secret.Permission{declaration}},
	}
	if got := idsOf(secret.ListAllowed(page, declaration)); !slices.Equal(got, []string{"first", "second"}) {
		t.Errorf("list = %v, want the order of the page", got)
	}
}

func TestThePageComparesNamesByTheirUTF8BytesNotByUTF16Units(t *testing.T) {
	list := []secret.Secret{
		{ID: "s1", Group: "\U0001D49C"}, {ID: "s2", Group: "ｱ"}, {ID: "s3", Group: "Z"}, {ID: "s4", Group: "a"},
	}
	if got := idsOf(secret.ListInPageOrder(list, nil)); !slices.Equal(got, []string{"s3", "s4", "s2", "s1"}) {
		t.Errorf("page = %v, want s3 s4 s2 s1", got)
	}
}

func TestThePageKeepsTheOrderOfTheListInsideAGroup(t *testing.T) {
	list := []secret.Secret{{ID: "s1", Group: "openai"}, {ID: "s2", Group: "other"}, {ID: "s3", Group: "openai"}}
	if got := idsOf(secret.ListInPageOrder(list, []string{"openai"})); !slices.Equal(got, []string{"s1", "s3", "s2"}) {
		t.Errorf("page = %v, want s1 s3 s2", got)
	}
}

func TestSectionsKeepAnAskedGroupWithNoSecretAndMarkTheAskedGroups(t *testing.T) {
	list := []secret.Secret{{ID: "s1", Group: "mine"}, {ID: "s2", Group: "openai"}}
	got := secret.ListSections(list, []string{"anthropic", "openai"})
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

func TestAMenuListsTheSecretsOfTheDeclaredGroupFirstThenTheOthersInPageOrder(t *testing.T) {
	list := []secret.Secret{
		{ID: "mine", Group: "home"}, {ID: "a1", Group: "openai"}, {ID: "r1", Group: "openrouter"}, {ID: "r2", Group: "openrouter"},
	}
	page := secret.ListSections(list, []string{"openai", "openrouter"})
	declaration := secret.Permission{Plugin: "a.b", Name: "key", Group: "openrouter"}
	if got := idsOf(secret.ListMenu(page, declaration)); !slices.Equal(got, []string{"r1", "r2", "a1", "mine"}) {
		t.Errorf("menu = %v", got)
	}
	none := secret.Permission{Plugin: "a.b", Name: "key"}
	if got := idsOf(secret.ListMenu(page, none)); !slices.Equal(got, []string{"a1", "r1", "r2", "mine"}) {
		t.Errorf("menu with no group = %v", got)
	}
}
