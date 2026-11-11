package secret_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/secret"
)

func idsOf(list []plugin.Secret) []string {
	var ids []string
	for _, one := range list {
		ids = append(ids, one.ID)
	}
	return ids
}

// forbiddenStore fails the test on every read of a value.
type forbiddenStore struct {
	keys
	t *testing.T
}

func (s forbiddenStore) GetKey(_ context.Context, name string) (string, error) {
	s.t.Errorf("the key store was read for %s", name)
	return "", errors.New("read")
}

func TestListReadsNoValueFromTheKeyStore(t *testing.T) {
	f := newFixture(plugin.Secret{ID: "s00000001", Name: "Here", Group: "g"})
	f.secrets = newSecrets(f.ledger, forbiddenStore{keys: f.keys, t: t}, f.plugins, nil, func(context.Context) string { return "/credentials.json" }, nil)

	listing, err := f.secrets.List.Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if listing.CredentialsPath != "/credentials.json" || len(listing.Sections) != 2 {
		t.Errorf("listing = %+v", listing)
	}
}

func TestListReturnsSectionsInPageOrderAndMenus(t *testing.T) {
	f := newFixture(
		plugin.Secret{ID: "s00000001", Name: "Mine", Group: "home"},
		plugin.Secret{ID: "s00000002", Name: "Work", Group: "openrouter"},
	)
	listing, err := f.secrets.List.Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var groups []string
	for _, section := range listing.Sections {
		groups = append(groups, section.Group)
	}
	if want := []string{"openrouter", "home"}; !slices.Equal(groups, want) {
		t.Errorf("groups = %v, want %v", groups, want)
	}
	if !listing.Sections[0].IsAsked || listing.Sections[1].IsAsked {
		t.Errorf("sections = %+v", listing.Sections)
	}
	if len(listing.Menus) != 2 {
		t.Fatalf("menus = %+v", listing.Menus)
	}
	if got := idsOf(listing.Menus[0].Secrets); listing.Menus[0].Declaration != getAlphaKey() || !slices.Equal(got, []string{"s00000002", "s00000001"}) {
		t.Errorf("menu of the declaration with a group = %v", got)
	}
	if got := idsOf(listing.Menus[1].Secrets); listing.Menus[1].Declaration != getBetaKey() || !slices.Equal(got, []string{"s00000002", "s00000001"}) {
		t.Errorf("menu of the declaration with no group = %v", got)
	}
}

func TestListStatesReportsValuePresence(t *testing.T) {
	f := newFixture(plugin.Secret{ID: "s00000001", Name: "Here", Group: "g"}, plugin.Secret{ID: "s00000002", Name: "Away", Group: "g"})
	f.keys["secret/s00000001"] = "value-that-must-stay-secret"

	states, err := f.secrets.Access.ListStates.Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	want := []secret.State{{ID: "s00000001", HasValue: true}, {ID: "s00000002"}}
	if !slices.Equal(states, want) {
		t.Errorf("states = %+v, want %+v", states, want)
	}
}

func TestListStatesFailsOnACancelledContext(t *testing.T) {
	f := newFixture(plugin.Secret{ID: "s00000001", Name: "Here", Group: "g"})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := f.secrets.Access.ListStates.Execute(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v", err)
	}
}
