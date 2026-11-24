package settings_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
)

func TestCredentialsFileIsUnsetWhereNothingWasChosen(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "numen.json")
	other := filepath.Join(t.TempDir(), "numen.json")
	if err := os.WriteFile(other, []byte(`{"v":1,"app":{"keys":{"credentialsFile":"secret"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	for name, path := range map[string]string{"missing file": missing, "unknown value": other} {
		if got := settings.ReadCredentialsFile(path); got != settings.CredentialsFileUnset {
			t.Errorf("%s: %q", name, got)
		}
	}
}

func TestCredentialsFileChoiceIsKeptForTheNextRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "numen.json")

	for _, value := range []string{settings.CredentialsFilePassphrase, settings.CredentialsFilePlaintext} {
		if err := settings.WriteCredentialsFile(path, value); err != nil {
			t.Fatal(err)
		}
		if got := settings.ReadCredentialsFile(path); got != value {
			t.Errorf("wrote %q, read %q", value, got)
		}
	}
}
