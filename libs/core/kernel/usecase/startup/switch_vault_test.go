package startup_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/startup"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func TestSwitchVaultPointsThePluginsAtTheVaultOnlyWhereItWasShown(t *testing.T) {
	shown := domain.Vault{Name: "garden", Path: "/vaults/garden"}
	var followed []string
	var told []error
	failure := errors.New("not a vault")
	show := func(_ context.Context, v domain.Vault, _ struct{}) error {
		if v.Name != shown.Name {
			return failure
		}
		return nil
	}
	switching := startup.SwitchVault[struct{}]{
		Show:        show,
		FollowVault: func(path string) { followed = append(followed, path) },
		After:       func(_ domain.Vault, err error) { told = append(told, err) },
	}

	okErr := switching.Execute(t.Context(), shown, struct{}{})
	badErr := switching.Execute(t.Context(), domain.Vault{Name: "missing", Path: "/vaults/missing"}, struct{}{})

	if okErr != nil || !errors.Is(badErr, failure) {
		t.Fatalf("errors %v and %v", okErr, badErr)
	}
	if len(followed) != 1 || followed[0] != "/vaults/garden" || len(told) != 2 || told[0] != nil || !errors.Is(told[1], failure) {
		t.Errorf("followed %v, told %v", followed, told)
	}
}
