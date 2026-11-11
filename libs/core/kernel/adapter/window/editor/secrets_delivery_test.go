package editor_test

import (
	"encoding/json"
	"errors"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
)

func (w *window) awaitIDs(t *testing.T, pluginID, name string, want ...string) {
	t.Helper()
	var got []string
	wiretest.WaitUntil(t, pluginID+" receives "+strings.Join(want, ",")+" for "+name, func() bool {
		got = w.probes[pluginID].ids(t, name)
		return slices.Equal(got, want) || (len(got) == 0 && len(want) == 0)
	})
}

func (w *window) assertIDs(t *testing.T, pluginID, name string, want ...string) {
	t.Helper()
	got := w.probes[pluginID].ids(t, name)
	if !slices.Equal(got, want) && (len(got) != 0 || len(want) != 0) {
		t.Errorf("%s received %v for %s, want %v", pluginID, got, name, want)
	}
}

func (w *window) counts() map[string]int {
	counts := map[string]int{}
	for id, one := range w.probes {
		counts[id] = one.count()
	}
	return counts
}

func TestADeclarationReceivesOnlyTheSecretsAllowedToIt(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	allowed := w.create(t, "openrouter", "Work", "value-of-the-work-key")
	other := w.create(t, "openrouter", "Home", "value-of-the-home-key")

	w.allow(t, allowed, alphaID, "apiKey", "openrouter")

	w.awaitIDs(t, alphaID, "apiKey", allowed)
	entry, _ := w.probes[alphaID].entry(t, "apiKey")
	if got := entry.GetValues()[0]; got.GetValue() != "value-of-the-work-key" || got.GetName() != "Work" || got.GetGroup() != "openrouter" {
		t.Errorf("delivered %v, want the Work secret of openrouter with its value", got)
	}
	if strings.Contains(w.probes[alphaID].serialized(), "value-of-the-home-key") {
		t.Errorf("a secret that is not allowed %s reached the plugin", other)
	}
}

func TestTwoDeclarationsOfOnePluginAreAllowedSeparately(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	one := w.create(t, "openrouter", "Work", "value-of-the-work-key")
	w.allow(t, one, alphaID, "apiKey", "openrouter")
	w.awaitIDs(t, alphaID, "apiKey", one)

	w.setProvider(t, "custom")
	w.awaitIDs(t, alphaID, "apiKey")
	if entry, _ := w.probes[alphaID].entry(t, "apiKey"); entry.GetGroup() != "" {
		t.Errorf("the declaration for custom has group %q, want none", entry.GetGroup())
	}

	w.setProvider(t, "openrouter")
	w.awaitIDs(t, alphaID, "apiKey", one)
	if entry, _ := w.probes[alphaID].entry(t, "apiKey"); entry.GetGroup() != "openrouter" {
		t.Errorf("the declaration for openrouter has group %q", entry.GetGroup())
	}
}

func TestASecretAllowedToAnotherPluginNeverReachesAPlugin(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	one := w.create(t, "openai", "Work", "value-for-beta-only")
	w.allow(t, one, betaID, "apiKey", "openai")
	w.awaitIDs(t, betaID, "apiKey", one)

	w.setProvider(t, "openai")
	w.awaitIDs(t, alphaID, "apiKey")
	for _, id := range []string{alphaID, gammaID} {
		if strings.Contains(w.probes[id].serialized(), "value-for-beta-only") {
			t.Errorf("plugin %s received a secret allowed to another plugin", id)
		}
	}
}

func TestNothingIsAllowedByDefault(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	first := w.create(t, "openrouter", "Default", "value-of-the-first-key")
	second := w.create(t, "mistral", "Default", "value-of-the-second-key")

	for _, id := range []string{first, second} {
		if got := w.find(t, id); got == nil || len(got.GetAllowed()) != 0 {
			t.Errorf("secret %s has allowed %v, want none", id, got.GetAllowed())
		}
	}
	for pluginID, name := range map[string]string{alphaID: "apiKey", betaID: "apiKey", gammaID: "apiKey"} {
		if ids := w.probes[pluginID].ids(t, name); len(ids) != 0 {
			t.Errorf("plugin %s receives %v with nothing allowed", pluginID, ids)
		}
		if strings.Contains(w.probes[pluginID].serialized(), "value-of-the-") {
			t.Errorf("plugin %s received a value with nothing allowed", pluginID)
		}
	}
}

func TestAListStartsWithTheDeclaredGroupAndThenFollowsThePageOrder(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	x := w.create(t, "openrouter", "X", "value-x")
	y := w.create(t, "openrouter", "Y", "value-y")
	z := w.create(t, "openai", "Z", "value-z")
	mine := w.create(t, "work", "W", "value-w")
	for _, id := range []string{x, y, z, mine} {
		w.allow(t, id, alphaID, "apiKey", "openrouter")
	}

	w.awaitIDs(t, alphaID, "apiKey", x, y, z, mine)

	w.setProvider(t, "custom")
	for _, id := range []string{x, y, z, mine} {
		w.allow(t, id, alphaID, "apiKey", "")
	}
	w.awaitIDs(t, alphaID, "apiKey", z, x, y, mine)
}

func TestADeclarationWithNothingAllowedIsDeliveredAsAnEntryWithNoValues(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	entry, found := w.probes[alphaID].entry(t, "orgId")
	if !found || len(entry.GetValues()) != 0 {
		t.Errorf("orgId = %v, %v, want an entry with no values", entry, found)
	}
}

func TestAMoveChangesTheOrderAPluginReceives(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	x := w.create(t, "openrouter", "X", "value-x")
	y := w.create(t, "openrouter", "Y", "value-y")
	z := w.create(t, "openai", "Z", "value-z")
	w.allow(t, x, alphaID, "apiKey", "openrouter")
	w.allow(t, y, alphaID, "apiKey", "openrouter")
	w.awaitIDs(t, alphaID, "apiKey", x, y)

	if _, err := w.client.MoveSecret(t.Context(), connect.NewRequest(&v1.MoveSecretRequest{Id: y, BeforeId: x})); err != nil {
		t.Fatal(err)
	}
	w.awaitIDs(t, alphaID, "apiKey", y, x)
	if got := ids(w.secrets(t)); slices.Index(got, y) > slices.Index(got, x) {
		t.Errorf("the list is %v, want %s before %s", got, y, x)
	}

	if _, err := w.client.MoveSecret(t.Context(), connect.NewRequest(&v1.MoveSecretRequest{Id: y})); err != nil {
		t.Fatal(err)
	}
	w.awaitIDs(t, alphaID, "apiKey", x, y)

	before, listBefore := w.counts(), ids(w.secrets(t))
	if _, err := w.client.MoveSecret(t.Context(), connect.NewRequest(&v1.MoveSecretRequest{Id: y, BeforeId: z})); err == nil {
		t.Error("a move into another group was accepted")
	}
	if after := w.counts(); after[alphaID] != before[alphaID] || !slices.Equal(ids(w.secrets(t)), listBefore) {
		t.Errorf("a refused move changed the list or sent Configure: %v -> %v", before, after)
	}
	w.awaitIDs(t, alphaID, "apiKey", x, y)
}

func TestOnlyTheActiveDeclarationsAreDelivered(t *testing.T) {
	w := openSecretsWindow(t, `{}`)

	var carried []string
	for _, one := range w.probes[alphaID].last(t).GetSecrets() {
		carried = append(carried, one.GetName()+"/"+one.GetGroup())
	}
	slices.Sort(carried)
	if want := []string{"apiKey/openrouter", "orgId/"}; !slices.Equal(carried, want) {
		t.Errorf("provider openrouter delivers %v, want %v", carried, want)
	}

	w.setProvider(t, "custom")
	wiretest.WaitUntil(t, "the declaration for custom is delivered", func() bool {
		entry, found := w.probes[alphaID].entry(t, "apiKey")
		return found && entry.GetGroup() == ""
	})
	carried = nil
	for _, one := range w.probes[alphaID].last(t).GetSecrets() {
		carried = append(carried, one.GetName()+"/"+one.GetGroup())
	}
	slices.Sort(carried)
	if want := []string{"apiKey/", "orgId/"}; !slices.Equal(carried, want) {
		t.Errorf("provider custom delivers %v, want %v", carried, want)
	}
}

func (w *window) editFile(t *testing.T, change func(secrets []map[string]any) []map[string]any) {
	t.Helper()
	var root map[string]any
	if err := json.Unmarshal([]byte(w.readFile(t)), &root); err != nil {
		t.Fatal(err)
	}
	var secrets []map[string]any
	raw, err := json.Marshal(root["secrets"])
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &secrets); err != nil {
		t.Fatal(err)
	}
	root["secrets"] = change(secrets)
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	// An editor saves by replacing the file. A write in place passes through an empty file, which a
	// reader of the settings may take for an installation with no settings.
	draft := w.file + ".edit"
	if err := os.WriteFile(draft, out, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(draft, w.file); err != nil {
		t.Fatal(err)
	}
}

func TestAHandEditOfThePermissionsReachesOnlyThePluginsWhoseListChanged(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	x := w.create(t, "openrouter", "X", "value-x")
	y := w.create(t, "openrouter", "Y", "value-y")
	grant := []any{map[string]any{"plugin": alphaID, "name": "apiKey", "group": "openrouter"}}
	before := w.counts()

	w.editFile(t, func(secrets []map[string]any) []map[string]any {
		for _, one := range secrets {
			one["allowed"] = grant
		}
		return secrets
	})
	w.awaitIDs(t, alphaID, "apiKey", x, y)

	w.editFile(t, func(secrets []map[string]any) []map[string]any {
		slices.Reverse(secrets)
		return secrets
	})
	w.awaitIDs(t, alphaID, "apiKey", y, x)

	w.editFile(t, func(secrets []map[string]any) []map[string]any {
		secrets[0]["allowed"] = []any{}
		return secrets
	})
	w.awaitIDs(t, alphaID, "apiKey", x)

	after := w.counts()
	if after[betaID] != before[betaID] || after[gammaID] != before[gammaID] {
		t.Errorf("plugins whose list did not change were configured: %v -> %v", before, after)
	}
}

func TestAHandEditedEntryOfAPluginThatIsNotInstalledIsDropped(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	x := w.create(t, "openrouter", "X", "value-x")
	w.editFile(t, func(secrets []map[string]any) []map[string]any {
		secrets[0]["allowed"] = []any{
			map[string]any{"plugin": "md.numen.test.absent", "name": "apiKey", "group": "openrouter"},
			map[string]any{"plugin": alphaID, "name": "noSuchDeclaration", "group": ""},
		}
		return secrets
	})
	wiretest.WaitUntil(t, "the invalid permissions are dropped", func() bool {
		return len(w.find(t, x).GetAllowed()) == 0
	})
	if !strings.Contains(w.readFile(t), "md.numen.test.absent") {
		t.Error("the bytes of the file were rewritten, want them kept")
	}
}

func TestOnlyTheCoreWritesTheSecretsSection(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	w.create(t, "openrouter", "X", "value-x")
	before := w.readFile(t)

	for _, target := range [][]string{{"secrets"}, {"secrets", "0", "name"}, {"secrets", "0", "allowed"}} {
		_, err := w.client.WriteSettings(t.Context(), connect.NewRequest(&v1.WriteSettingsRequest{
			Settings: []*v1.Setting{{At: target, Value: `[]`}},
		}))
		if err == nil {
			t.Errorf("a write at %v was accepted", target)
		}
	}
	if after := w.readFile(t); after != before {
		t.Errorf("the file changed:\n%s", after)
	}
}

func TestNoValueIsInTheFileTheWindowRepliesOrTheConfigurationOfAPlugin(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	const value = "invented-value-that-must-stay-secret"
	id := w.create(t, "openrouter", "Work", value)
	w.allow(t, id, alphaID, "apiKey", "openrouter")
	w.awaitIDs(t, alphaID, "apiKey", id)
	if _, err := w.client.WriteSecret(t.Context(), connect.NewRequest(&v1.WriteSecretRequest{Id: id, Value: value + "-2"})); err != nil {
		t.Fatal(err)
	}

	if strings.Contains(w.readFile(t), "invented-value") {
		t.Errorf("the settings file holds a value:\n%s", w.readFile(t))
	}
	if text := protojson.Format(w.get(t)); strings.Contains(text, "invented-value") {
		t.Errorf("GetSettings holds a value:\n%s", text)
	}
	for pluginID, one := range w.probes {
		for _, call := range one.all() {
			if strings.Contains(call.GetConfigurationJson(), "invented-value") {
				t.Errorf("%s: the configuration json holds a value", pluginID)
			}
			for key, text := range call.GetValues() {
				if strings.Contains(text, "invented-value") {
					t.Errorf("%s: value %s holds a secret", pluginID, key)
				}
			}
		}
	}
}

func TestNoErrorTextCarriesAValue(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	const value = "invented-value-in-a-failing-call"
	w.keys.mu.Lock()
	w.keys.failSet = errors.New("the keychain is locked")
	w.keys.mu.Unlock()

	_, err := w.client.CreateSecret(t.Context(), connect.NewRequest(&v1.CreateSecretRequest{Group: "g", Name: "n", Value: value}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Errorf("a failing key store gave %v, want FailedPrecondition", err)
	}
	if err != nil && strings.Contains(err.Error(), value) {
		t.Errorf("the error carries the value: %v", err)
	}
	_, err = w.client.CreateSecret(t.Context(), connect.NewRequest(&v1.CreateSecretRequest{Group: "g", Name: " bad ", Value: value}))
	if err == nil || strings.Contains(err.Error(), value) {
		t.Errorf("a bad name gave %v", err)
	}
	_, err = w.client.WriteSecret(t.Context(), connect.NewRequest(&v1.WriteSecretRequest{Id: "s00000000", Value: value}))
	if err == nil || strings.Contains(err.Error(), value) {
		t.Errorf("an unknown id gave %v", err)
	}
	if len(w.secrets(t)) != 0 {
		t.Errorf("a failed create left %v", ids(w.secrets(t)))
	}
}

func TestASecretWithNoValueOnThisMachineIsReportedAndDeliveredToNoPlugin(t *testing.T) {
	const imported = `{"secrets":[{"id":"s0000abcd","name":"Imported","group":"openrouter",` +
		`"allowed":[{"plugin":"md.numen.test.alpha","name":"apiKey","group":"openrouter"}]}]}`
	w := openSecretsWindow(t, imported)

	got := w.find(t, "s0000abcd")
	if got == nil || w.hasValue(t, "s0000abcd") {
		t.Fatalf("secret = %v, want one that reports no value", got)
	}
	if ids := w.probes[alphaID].ids(t, "apiKey"); len(ids) != 0 {
		t.Errorf("a secret with no value was delivered: %v", ids)
	}
	w.create(t, "openrouter", "Another", "value-another")
	if !strings.Contains(w.readFile(t), "s0000abcd") {
		t.Error("the entry with no value was deleted from the file")
	}
}

func TestOperationsOnOnePluginAreSerialized(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	const count = 8
	created := make([]string, count)
	for i := range created {
		created[i] = w.create(t, "openrouter", string(rune('A'+i)), "value-"+string(rune('a'+i)))
	}

	var group sync.WaitGroup
	for _, id := range created {
		group.Go(func() {
			_, err := w.client.AllowPluginSecret(t.Context(), connect.NewRequest(&v1.AllowPluginSecretRequest{
				SecretId: id, PluginId: alphaID, Name: "apiKey", Group: "openrouter",
			}))
			if err != nil {
				t.Error(err)
			}
		})
	}
	group.Wait()

	wiretest.WaitUntil(t, "the last Configure reflects every operation", func() bool {
		got := w.probes[alphaID].ids(t, "apiKey")
		return len(got) == count
	})
	if got := w.probes[alphaID].ids(t, "apiKey"); !slices.Equal(got, created) {
		t.Errorf("the last Configure delivers %v, want %v", got, created)
	}
}
