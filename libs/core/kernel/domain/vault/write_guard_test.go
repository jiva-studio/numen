package vault_test

import (
	"errors"
	"testing"

	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
)

const (
	syncPlugin  = "md.numen.tools.sync"
	otherPlugin = "md.numen.tools.link"
)

func TestCheckWrite_AllowsAnyCallerToUnmanagedVault(t *testing.T) {
	t.Parallel()
	for _, caller := range []string{"", syncPlugin, otherPlugin} {
		if err := vaultdomain.CheckWrite("", caller); err != nil {
			t.Errorf("caller %q: error = %v, want none", caller, err)
		}
	}
}

func TestCheckWrite_AllowsOnlyTheManagingPluginToAManagedVault(t *testing.T) {
	t.Parallel()
	if err := vaultdomain.CheckWrite(syncPlugin, syncPlugin); err != nil {
		t.Errorf("the managing plugin: error = %v, want none", err)
	}
	for name, caller := range map[string]string{"the window": "", "another plugin": otherPlugin, "a longer id": syncPlugin + ".x", "another case": "MD.NUMEN.TOOLS.SYNC"} {
		if err := vaultdomain.CheckWrite(syncPlugin, caller); !errors.Is(err, vaultdomain.ErrReadOnly) {
			t.Errorf("%s: error = %v, want ErrReadOnly", name, err)
		}
	}
}

func TestErrReadOnly_ReadsAsTheTextOfTheWindow(t *testing.T) {
	t.Parallel()
	if got := vaultdomain.ErrReadOnly.Error(); got != "this vault is read only" {
		t.Errorf("text = %q", got)
	}
}
