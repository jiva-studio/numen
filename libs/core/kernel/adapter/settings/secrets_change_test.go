package settings_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

func getAlphaPermission() plugin.DeclarationKey {
	return plugin.DeclarationKey{Plugin: "md.numen.alpha", Name: "apiKey", Group: "openrouter"}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestChangeSecretsPreservesOtherSections(t *testing.T) {
	const before = "{\n  \"app\":   {\"appearance\": {\"mode\": \"dark\"}},\n  \"md.numen.alpha\": {\"model\": \"small\"}\n}\n"
	path := write(t, before)

	err := settings.NewFiles().ChangeSecrets(path, func(list []plugin.Secret) ([]plugin.Secret, error) {
		return append(list, plugin.Secret{ID: "s0a1b2c3d", Name: "Work", Group: "openrouter", Allowed: []plugin.DeclarationKey{getAlphaPermission()}}), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	after := readFile(t, path)
	if !strings.Contains(after, `"id":"s0a1b2c3d"`) || !strings.Contains(after, `"plugin":"md.numen.alpha"`) {
		t.Errorf("the section is missing from:\n%s", after)
	}
	if !strings.Contains(after, "\"app\":   {\"appearance\": {\"mode\": \"dark\"}}") || !strings.Contains(after, `"md.numen.alpha": {"model": "small"}`) {
		t.Errorf("the rest of the file changed:\n%s", after)
	}
	cfg, err := openAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Secrets) != 1 || cfg.Secrets[0].Allowed[0] != getAlphaPermission() {
		t.Errorf("read back %+v", cfg.Secrets)
	}
}

func TestChangeSecretsSkipsWriteOnFailureOrUnchangedList(t *testing.T) {
	path := write(t, `{"secrets":[{"id":"s0a1b2c3d","name":"Work","group":"g","allowed":[]}]}`)
	before := readFile(t, path)
	errRefused := errors.New("refused")

	if err := settings.NewFiles().ChangeSecrets(path, func([]plugin.Secret) ([]plugin.Secret, error) {
		return nil, errRefused
	}); !errors.Is(err, errRefused) {
		t.Errorf("error = %v", err)
	}
	if err := settings.NewFiles().ChangeSecrets(path, func(list []plugin.Secret) ([]plugin.Secret, error) {
		return list, nil
	}); err != nil {
		t.Fatal(err)
	}
	if after := readFile(t, path); after != before {
		t.Errorf("the file changed:\n%s", after)
	}
}

func TestChangeSecretsCreatesFileAndRejectsInvalidJSON(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "numen.json")
	add := func(list []plugin.Secret) ([]plugin.Secret, error) {
		return append(list, plugin.Secret{ID: "s0a1b2c3d", Name: "Work", Group: "g"}), nil
	}
	if err := settings.NewFiles().ChangeSecrets(missing, add); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(readFile(t, missing), "s0a1b2c3d") {
		t.Error("the file was not created with the secret")
	}

	broken := write(t, `{"secrets": [`)
	if err := settings.NewFiles().ChangeSecrets(broken, add); err == nil {
		t.Error("a file that is not JSON was written")
	}
	if readFile(t, broken) != `{"secrets": [` {
		t.Error("a file that is not JSON was changed")
	}
}

func TestChangeSecretsFromManyCallersLosesNoChange(t *testing.T) {
	path := write(t, `{}`)
	files := settings.NewFiles()
	const count = 12
	var group sync.WaitGroup
	for i := range count {
		group.Go(func() {
			id := fmt.Sprintf("s%08x", i)
			err := files.ChangeSecrets(path, func(list []plugin.Secret) ([]plugin.Secret, error) {
				return append(list, plugin.Secret{ID: id, Name: id, Group: "g"}), nil
			})
			if err != nil {
				t.Error(err)
			}
		})
	}
	group.Wait()
	cfg, err := openAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Secrets) != count {
		t.Errorf("%d secrets in the file, want %d", len(cfg.Secrets), count)
	}
}

func TestDropInvalidSecretsFiltersEntriesAndRecordsNotices(t *testing.T) {
	path := write(t, `{"secrets":[
		{"id":"s0a1b2c3d","name":"Work","group":"openrouter","allowed":[
			{"plugin":"md.numen.alpha","name":"apiKey","group":"openrouter"},
			{"plugin":"md.numen.alpha","name":"apiKey","group":"openrouter"},
			{"plugin":"md.numen.ghost","name":"apiKey","group":""}]},
		{"id":"s0a1b2c3d","name":"Twin","group":"openrouter","allowed":[]},
		{"id":"sBAD","name":"Short id","group":"openrouter","allowed":[]},
		{"id":"s0000aaaa","name":"Work","group":"openrouter","allowed":[]},
		{"id":"s0000bbbb","name":" untrimmed","group":"openrouter","allowed":[]},
		{"id":"s0000cccc","name":"Other group","group":"","allowed":[]},
		{"id":"s0000dddd","name":"Work","group":"home","allowed":[]}]}`)
	cfg, err := openAt(path)
	if err != nil {
		t.Fatal(err)
	}

	cfg.DropInvalidSecrets(func(one plugin.DeclarationKey) bool { return one == getAlphaPermission() })

	var ids []string
	for _, one := range cfg.Secrets {
		ids = append(ids, one.ID)
	}
	if want := []string{"s0a1b2c3d", "s0000dddd"}; !slices.Equal(ids, want) {
		t.Errorf("kept %v, want %v", ids, want)
	}
	if got := cfg.Secrets[0].Allowed; len(got) != 1 || got[0] != getAlphaPermission() {
		t.Errorf("permissions = %v, want the declared one once", got)
	}
	if !hasNotice(cfg.Notices, "md.numen.ghost") || !hasNotice(cfg.Notices, "listed twice") || len(cfg.Notices) != 7 {
		t.Errorf("notices = %q", cfg.Notices)
	}
	if got := readFile(t, path); !strings.Contains(got, "md.numen.ghost") || !strings.Contains(got, "sBAD") {
		t.Error("the bytes of the file were changed")
	}
}

func TestWriteJSONIncludesSecretsSectionWithoutValues(t *testing.T) {
	cfg, err := openAt(write(t, `{`+validSecretsSection+`}`))
	if err != nil {
		t.Fatal(err)
	}
	content, err := cfg.WriteJSON()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(content, `"secrets":[{"id":"s0a1b2c3d","name":"Default","group":"openrouter","allowed":[`) {
		t.Errorf("settings = %s", content)
	}
	section, err := cfg.WriteSecretsJSON()
	if err != nil || !strings.HasPrefix(section, `[{"id":"s0a1b2c3d"`) {
		t.Errorf("section = %s, %v", section, err)
	}
}
