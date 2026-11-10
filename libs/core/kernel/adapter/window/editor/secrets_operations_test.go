package editor_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
)

var secretID = regexp.MustCompile(`^s[0-9a-f]{8}$`)

func permission(plugin, name, group string) string { return plugin + "/" + name + "/" + group }

func allowedOf(one *v1.Secret) []string {
	var list []string
	for _, p := range one.GetAllowed() {
		list = append(list, permission(p.GetPluginId(), p.GetName(), p.GetGroup()))
	}
	slices.Sort(list)
	return list
}

func TestAllowAndDenyChangeThePermissionAndThePluginsList(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	one := w.create(t, "openrouter", "Work", "value-work")
	before := w.counts()

	w.allow(t, one, alphaID, "apiKey", "openrouter")
	w.assertIDs(t, alphaID, "apiKey", one)
	if got := allowedOf(w.find(t, one)); !slices.Equal(got, []string{permission(alphaID, "apiKey", "openrouter")}) {
		t.Errorf("allowed = %v", got)
	}
	if after := w.counts(); after[betaID] != before[betaID] || after[gammaID] != before[gammaID] {
		t.Errorf("plugins whose list did not change were configured: %v -> %v", before, after)
	}

	w.deny(t, one, alphaID, "apiKey", "openrouter")
	w.assertIDs(t, alphaID, "apiKey")
	if got := allowedOf(w.find(t, one)); len(got) != 0 {
		t.Errorf("allowed = %v after a deny", got)
	}
}

func TestAllowAndDenyAreIdempotent(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	one := w.create(t, "openrouter", "Work", "value-work")

	w.allow(t, one, alphaID, "apiKey", "openrouter")
	w.allow(t, one, alphaID, "apiKey", "openrouter")
	if got := allowedOf(w.find(t, one)); len(got) != 1 {
		t.Errorf("allowed = %v after two allows, want one", got)
	}
	w.deny(t, one, alphaID, "apiKey", "openrouter")
	w.deny(t, one, alphaID, "apiKey", "openrouter")
	if got := allowedOf(w.find(t, one)); len(got) != 0 {
		t.Errorf("allowed = %v after two denies", got)
	}
}

func TestAllowRefusesAPluginOrDeclarationThatDoesNotExist(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	one := w.create(t, "openrouter", "Work", "value-work")

	for what, request := range map[string]*v1.AllowPluginSecretRequest{
		"a plugin that is not installed":     {SecretId: one, PluginId: "md.numen.test.absent", Name: "apiKey", Group: "openrouter"},
		"a name the plugin does not declare": {SecretId: one, PluginId: alphaID, Name: "token", Group: "openrouter"},
		"a group the declaration lacks":      {SecretId: one, PluginId: alphaID, Name: "apiKey", Group: "mistral"},
		"the group of another declaration":   {SecretId: one, PluginId: betaID, Name: "apiKey", Group: "openrouter"},
	} {
		if _, err := w.client.AllowPluginSecret(t.Context(), connect.NewRequest(request)); err == nil {
			t.Errorf("an allow for %s was accepted", what)
		}
	}
	if _, err := w.client.AllowPluginSecret(t.Context(), connect.NewRequest(&v1.AllowPluginSecretRequest{
		SecretId: "s00000000", PluginId: alphaID, Name: "apiKey", Group: "openrouter",
	})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("an allow of an unknown secret gave %v, want NotFound", err)
	}
	if got := allowedOf(w.find(t, one)); len(got) != 0 {
		t.Errorf("a refused allow left %v", got)
	}
}

func TestACreatedSecretGoesToTheEndOfItsGroupWithItsValueInTheKeyStore(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	first := w.create(t, "openrouter", "Work", "value-work")
	other := w.create(t, "openai", "Elsewhere", "value-elsewhere")
	last := w.create(t, "openrouter", "Home", "value-home")

	for _, id := range []string{first, other, last} {
		if !secretID.MatchString(id) {
			t.Errorf("id %q is not s followed by 8 hex digits", id)
		}
	}
	got := w.secrets(t)
	if !slices.Equal(ids(got), []string{first, other, last}) && !slices.Equal(ids(got), []string{other, first, last}) {
		t.Errorf("the list is %v", ids(got))
	}
	if slices.Index(ids(got), last) < slices.Index(ids(got), first) {
		t.Errorf("the new secret is before the older one of its group: %v", ids(got))
	}
	if w.keys.snapshot()["secret/"+first] != "value-work" || w.keys.snapshot()["secret/"+last] != "value-home" {
		t.Errorf("the key store holds %v", w.keys.snapshot())
	}
	if !w.hasValue(t, first) {
		t.Error("a secret with a value reports none")
	}
	entries := w.fileSecrets(t)
	if len(entries) != 3 || entries[0]["id"] != first || entries[0]["name"] != "Work" || entries[0]["group"] != "openrouter" {
		t.Errorf("the file holds %v", entries)
	}
}

func TestRenamingASecretChangesOnlyTheFile(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	one := w.create(t, "openrouter", "Work", "value-work")
	w.allow(t, one, alphaID, "apiKey", "openrouter")
	w.assertIDs(t, alphaID, "apiKey", one)
	keysBefore := w.keys.snapshot()

	if _, err := w.client.RenameSecret(t.Context(), connect.NewRequest(&v1.RenameSecretRequest{Id: one, Name: "Office"})); err != nil {
		t.Fatal(err)
	}

	got := w.find(t, one)
	if got.GetName() != "Office" || len(got.GetAllowed()) != 1 || !w.hasValue(t, one) {
		t.Errorf("secret = %v, want the new name with its permission and value", got)
	}
	if after := w.keys.snapshot(); !slices.Equal(keysOf(after), keysOf(keysBefore)) || after["secret/"+one] != "value-work" {
		t.Errorf("the key store changed: %v -> %v", keysBefore, after)
	}
	if w.fileSecrets(t)[0]["name"] != "Office" {
		t.Errorf("the file holds %v", w.fileSecrets(t))
	}
}

func keysOf(items map[string]string) []string {
	var names []string
	for name := range items {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func TestWritingASecretReplacesTheValueAndConfiguresThePluginsThatReadIt(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	one := w.create(t, "openrouter", "Work", "value-old")
	w.allow(t, one, alphaID, "apiKey", "openrouter")
	w.assertIDs(t, alphaID, "apiKey", one)
	before := w.counts()

	if _, err := w.client.WriteSecret(t.Context(), connect.NewRequest(&v1.WriteSecretRequest{Id: one, Value: "value-new"})); err != nil {
		t.Fatal(err)
	}

	if w.keys.snapshot()["secret/"+one] != "value-new" {
		t.Errorf("the key store holds %v", w.keys.snapshot())
	}
	if w.counts()[alphaID] != before[alphaID]+1 {
		t.Errorf("the plugin that reads the secret received %d calls, want 1", w.counts()[alphaID]-before[alphaID])
	}
	entry, _ := w.probes[alphaID].entry(t, "apiKey")
	if entry.GetValues()[0].GetValue() != "value-new" {
		t.Errorf("the plugin received %v", entry.GetValues())
	}
	if after := w.counts(); after[gammaID] != before[gammaID] {
		t.Errorf("a plugin that cannot read the secret was configured: %v -> %v", before, after)
	}
}

func TestABadRequestIsRefusedAndChangesNothing(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	one := w.create(t, "openrouter", "Work", "value-work")
	w.create(t, "openrouter", "Home", "value-home")
	fileBefore, keysBefore, idsBefore := w.readFile(t), w.keys.snapshot(), ids(w.secrets(t))

	type call func(context.Context) error
	create := func(group, name, value string) call {
		return func(ctx context.Context) error {
			_, err := w.client.CreateSecret(ctx, connect.NewRequest(&v1.CreateSecretRequest{Group: group, Name: name, Value: value}))
			return err
		}
	}
	rename := func(id, name string) call {
		return func(ctx context.Context) error {
			_, err := w.client.RenameSecret(ctx, connect.NewRequest(&v1.RenameSecretRequest{Id: id, Name: name}))
			return err
		}
	}
	write := func(id, value string) call {
		return func(ctx context.Context) error {
			_, err := w.client.WriteSecret(ctx, connect.NewRequest(&v1.WriteSecretRequest{Id: id, Value: value}))
			return err
		}
	}
	for what, c := range map[string]struct {
		call call
		code connect.Code
	}{
		"an empty name":             {create("openrouter", "", "value"), connect.CodeInvalidArgument},
		"an untrimmed name":         {create("openrouter", " Work", "value"), connect.CodeInvalidArgument},
		"a name over 64 characters": {create("openrouter", strings.Repeat("n", 65), "value"), connect.CodeInvalidArgument},
		"an empty value":            {create("openrouter", "Fresh", ""), connect.CodeInvalidArgument},
		"a name taken in the group": {create("openrouter", "Work", "value"), connect.CodeInvalidArgument},
		"a rename to a taken name":  {rename(one, "Home"), connect.CodeInvalidArgument},
		"a rename to an empty name": {rename(one, ""), connect.CodeInvalidArgument},
		"a rename of an unknown id": {rename("s00000000", "Fresh"), connect.CodeNotFound},
		"a write of an empty value": {write(one, ""), connect.CodeInvalidArgument},
		"a write to an unknown id":  {write("s00000000", "value"), connect.CodeNotFound},
	} {
		if err := c.call(t.Context()); connect.CodeOf(err) != c.code {
			t.Errorf("%s gave %v, want %v", what, err, c.code)
		}
	}
	if w.readFile(t) != fileBefore || !slices.Equal(ids(w.secrets(t)), idsBefore) {
		t.Errorf("a refused call changed the file:\n%s", w.readFile(t))
	}
	if after := w.keys.snapshot(); !slices.Equal(keysOf(after), keysOf(keysBefore)) || after["secret/"+one] != "value-work" {
		t.Errorf("a refused call changed the key store: %v", after)
	}
}

func TestTheSameNameInAnotherGroupIsAllowed(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	w.create(t, "openrouter", "Default", "value-one")
	if _, err := w.client.CreateSecret(t.Context(), connect.NewRequest(&v1.CreateSecretRequest{
		Group: "openai", Name: "Default", Value: "value-two",
	})); err != nil {
		t.Errorf("the same name in another group: %v", err)
	}
}

func TestAGroupNameThatEqualsAnExistingOneJoinsItAndOneThatDiffersByCaseDoesNot(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	first := w.create(t, "openrouter", "Work", "value-work")
	second := w.create(t, "openrouter", "Home", "value-home")
	cased := w.create(t, "OpenRouter", "Work", "value-cased")

	groups := map[string]string{}
	for _, one := range w.secrets(t) {
		groups[one.GetId()] = one.GetGroup()
	}
	if groups[first] != "openrouter" || groups[second] != "openrouter" {
		t.Errorf("groups = %v, want the second secret in the first group", groups)
	}
	if groups[cased] != "OpenRouter" {
		t.Errorf("groups = %v, want OpenRouter to be a group of its own", groups)
	}
}

func TestDeletingASecretRemovesItsEntryValueAndPermissionsAndConfiguresThePluginsThatReadIt(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	one := w.create(t, "openai", "Shared", "value-shared")
	keep := w.create(t, "openai", "Kept", "value-kept")
	w.allow(t, one, alphaID, "apiKey", "openrouter")
	w.allow(t, one, betaID, "apiKey", "openai")
	w.allow(t, keep, betaID, "apiKey", "openai")
	w.assertIDs(t, alphaID, "apiKey", one)
	w.assertIDs(t, betaID, "apiKey", one, keep)
	before := w.counts()

	if _, err := w.client.DeleteSecret(t.Context(), connect.NewRequest(&v1.DeleteSecretRequest{Id: one})); err != nil {
		t.Fatal(err)
	}

	w.assertIDs(t, alphaID, "apiKey")
	w.assertIDs(t, betaID, "apiKey", keep)
	if w.find(t, one) != nil || len(w.fileSecrets(t)) != 1 {
		t.Errorf("the secret is still listed: %v", ids(w.secrets(t)))
	}
	if _, found := w.keys.snapshot()["secret/"+one]; found {
		t.Error("the value is still in the key store")
	}
	if strings.Contains(w.readFile(t), one) {
		t.Error("the file still names the secret")
	}
	if w.counts()[gammaID] != before[gammaID] {
		t.Error("a plugin that could not read the secret was configured")
	}
}

func TestDeletingAnUnknownSecretIsNotFound(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	if _, err := w.client.DeleteSecret(t.Context(), connect.NewRequest(&v1.DeleteSecretRequest{Id: "s00000000"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("error = %v, want NotFound", err)
	}
}

func TestSecretsOfAGroupNoPluginAsksForRemainWhenItsLastPluginIsUninstalled(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	one := w.create(t, "openai", "Work", "value-work")
	w.allow(t, one, betaID, "apiKey", "openai")

	if err := os.RemoveAll(filepath.Join(w.pluginsd, "beta")); err != nil {
		t.Fatal(err)
	}
	if err := w.host.Discovery.Discover(); err != nil {
		t.Fatal(err)
	}

	got := w.find(t, one)
	if got == nil || got.GetGroup() != "openai" || !w.hasValue(t, one) {
		t.Errorf("secret = %v, want it kept in openai", got)
	}
	if len(w.fileSecrets(t)) != 1 {
		t.Errorf("the file holds %v", w.fileSecrets(t))
	}
}

func TestUninstallingAPluginRemovesItsPermissionsAndDisablingKeepsThem(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	one := w.create(t, "openai", "Work", "value-work")
	w.allow(t, one, alphaID, "apiKey", "openrouter")
	w.allow(t, one, betaID, "apiKey", "openai")

	if err := setEnabled(t.Context(), w.host, betaID, false); err != nil {
		t.Fatal(err)
	}
	if got := allowedOf(w.find(t, one)); len(got) != 2 {
		t.Errorf("allowed = %v after a disable, want both kept", got)
	}

	if err := os.RemoveAll(filepath.Join(w.pluginsd, "beta")); err != nil {
		t.Fatal(err)
	}
	if err := w.host.Discovery.Discover(); err != nil {
		t.Fatal(err)
	}
	if got := allowedOf(w.find(t, one)); !slices.Equal(got, []string{permission(alphaID, "apiKey", "openrouter")}) {
		t.Errorf("allowed = %v after an uninstall, want only alpha", got)
	}
	// A disable writes the plugin's own entry to numen.json, so only the secrets section is checked.
	if strings.Contains(fmt.Sprint(w.fileSecrets(t)), betaID) {
		t.Error("the secrets section still names the uninstalled plugin")
	}
}

func TestAnUpdateThatDropsADeclarationRemovesOnlyThePermissionsOfThatDeclaration(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	one := w.create(t, "openrouter", "Work", "value-work")
	w.allow(t, one, alphaID, "apiKey", "openrouter")
	w.allow(t, one, alphaID, "orgId", "")

	updated := strings.Replace(alphaManifest, `,
		{"name": "orgId", "title": "Organization", "description": "Organization of the account"}`, "", 1)
	if updated == alphaManifest {
		t.Fatal("the fixture manifest has no orgId declaration to drop")
	}
	writeFile(t, filepath.Join(w.pluginsd, "alpha", "manifest.json"), strings.Replace(updated, `"1.0.0"`, `"1.1.0"`, 1))
	if err := w.host.Discovery.Discover(); err != nil {
		t.Fatal(err)
	}

	if got := allowedOf(w.find(t, one)); !slices.Equal(got, []string{permission(alphaID, "apiKey", "openrouter")}) {
		t.Errorf("allowed = %v, want only the declaration that remains", got)
	}
}

func TestRenamingAGroupThePersonMadeMovesAllItsSecrets(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	one := w.create(t, "work", "A", "value-a")
	two := w.create(t, "work", "B", "value-b")
	other := w.create(t, "openai", "C", "value-c")

	if _, err := w.client.RenameSecretGroup(t.Context(), connect.NewRequest(&v1.RenameSecretGroupRequest{Group: "work", Name: "office"})); err != nil {
		t.Fatal(err)
	}

	for _, id := range []string{one, two} {
		if got := w.find(t, id).GetGroup(); got != "office" {
			t.Errorf("secret %s is in %q, want office", id, got)
		}
	}
	if got := w.find(t, other).GetGroup(); got != "openai" {
		t.Errorf("a secret of another group moved to %q", got)
	}
	for _, entry := range w.fileSecrets(t) {
		if entry["group"] == "work" {
			t.Errorf("the file still holds the old group: %v", entry)
		}
	}
}

func TestRenamingAGroupToATakenNameOrRenamingAnAskedForGroupIsRefused(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	w.create(t, "work", "A", "value-a")
	w.create(t, "home", "B", "value-b")
	w.create(t, "openai", "C", "value-c")
	before := w.readFile(t)

	for what, request := range map[string]*v1.RenameSecretGroupRequest{
		"a taken name":                     {Group: "work", Name: "home"},
		"a group a plugin asks for":        {Group: "openai", Name: "renamed"},
		"a name that matches an asked-for": {Group: "work", Name: "openai"},
		"an empty name":                    {Group: "work", Name: ""},
		"an untrimmed name":                {Group: "work", Name: " office"},
	} {
		if _, err := w.client.RenameSecretGroup(t.Context(), connect.NewRequest(request)); err == nil {
			t.Errorf("a rename to %s was accepted", what)
		}
	}
	if w.readFile(t) != before {
		t.Errorf("a refused rename wrote the file:\n%s", w.readFile(t))
	}
}

func TestTheGroupsAPluginAsksForAreDerivedFromTheDeclarations(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	got := w.get(t)
	var asked []string
	for _, info := range got.GetPlugins() {
		for _, secret := range info.GetSecrets() {
			if secret.GetGroup() != "" {
				asked = append(asked, secret.GetGroup())
			}
		}
	}
	slices.Sort(asked)
	if want := []string{"mistral", "openai", "openrouter"}; !slices.Equal(slices.Compact(asked), want) {
		t.Errorf("the groups plugins ask for are %v, want %v", slices.Compact(asked), want)
	}
	if len(w.secrets(t)) != 0 {
		t.Errorf("a new installation lists %v", ids(w.secrets(t)))
	}
}
