package settings_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
)

func TestTheFolderForNewVaultsIsEmptyWhereNothingWasSet(t *testing.T) {
	file := settings.DefaultFile()

	if got := file.GetNewVaultsFolder(); got != "" {
		t.Errorf("the folder for new vaults is %q, want none", got)
	}
}

func TestTheFolderForNewVaultsIsReadFromTheFile(t *testing.T) {
	file := readFileWith(t, `{"app":{"vaults":{"folder":"/srv/vaults"}}}`)

	if got := file.GetNewVaultsFolder(); got != "/srv/vaults" {
		t.Errorf("the folder for new vaults is %q, want /srv/vaults", got)
	}
}

func TestTheFolderForNewVaultsIsKeptWhenTheFileIsWrittenAgain(t *testing.T) {
	file := readFileWith(t, `{"app":{"vaults":{"folder":"/srv/vaults"}}}`)

	written, err := file.WriteJSON()
	if err != nil {
		t.Fatal(err)
	}
	again := readFileWith(t, written)

	if got := again.GetNewVaultsFolder(); got != "/srv/vaults" {
		t.Errorf("the folder for new vaults after a write is %q, want /srv/vaults", got)
	}
}

func TestAFolderForNewVaultsThatIsNotTextIsIgnored(t *testing.T) {
	file := readFileWith(t, `{"app":{"vaults":{"folder":7}}}`)

	if got := file.GetNewVaultsFolder(); got != "" {
		t.Errorf("the folder for new vaults is %q, want none", got)
	}
}
