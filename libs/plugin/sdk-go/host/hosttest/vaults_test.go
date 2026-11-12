package hosttest_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
)

func TestGetVaultListMarksAMissingVault(t *testing.T) {
	physics, chemistry := hosttest.New("Physics"), hosttest.New("Chemistry")
	physics.Beside(chemistry)
	chemistry.SetMissing(true)

	got, err := physics.GetVaultList(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 2 || got[0].Name != "Chemistry" || !got[0].IsMissing || got[1].IsMissing {
		t.Errorf("got %+v", got)
	}
}

func TestAddVaultJoinsTheList(t *testing.T) {
	physics := hosttest.New("Physics")

	added, err := physics.AddVault(t.Context(), "/vaults/Biology", "")
	if err != nil {
		t.Fatal(err)
	}
	vaults, err := physics.GetVaults(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if added.Name != "Biology" || len(vaults) != 2 {
		t.Errorf("got %+v and %+v", added, vaults)
	}
	if _, err := physics.AddVault(t.Context(), "/vaults/Physics/inner", ""); !errors.Is(err, host.ErrVaultOverlaps) {
		t.Errorf("got %v", err)
	}
}

func TestRenameVaultToATakenNameIsNameTaken(t *testing.T) {
	physics, chemistry := hosttest.New("Physics"), hosttest.New("Chemistry")
	physics.Beside(chemistry)

	if _, err := physics.RenameVault(t.Context(), chemistry.Vault().ID, "physics"); !errors.Is(err, host.ErrVaultNameTaken) {
		t.Errorf("got %v", err)
	}
	renamed, err := physics.RenameVault(t.Context(), chemistry.Vault().ID, "Chem")
	if err != nil || renamed.Name != "Chem" {
		t.Errorf("got %+v, %v", renamed, err)
	}
}

func TestRemoveVaultRejectsTheLastAndTheOpenedVault(t *testing.T) {
	physics, chemistry := hosttest.New("Physics"), hosttest.New("Chemistry")
	physics.Beside(chemistry)
	if err := physics.OpenVault(t.Context(), physics.Vault().ID); err != nil {
		t.Fatal(err)
	}

	if _, err := physics.RemoveVault(t.Context(), physics.Vault().ID); !errors.Is(err, host.ErrVaultActive) {
		t.Errorf("got %v", err)
	}
	folder, err := physics.RemoveVault(t.Context(), chemistry.Vault().ID)
	if err != nil || folder != "/vaults/Chemistry" {
		t.Errorf("got %q, %v", folder, err)
	}
	if _, err := physics.RemoveVault(t.Context(), physics.Vault().ID); !errors.Is(err, host.ErrLastVault) {
		t.Errorf("got %v", err)
	}
	if _, err := chemistry.GetVault(t.Context()); !errors.Is(err, host.ErrUnknownVault) {
		t.Errorf("got %v", err)
	}
}

func TestOpenVaultRecordsTheVault(t *testing.T) {
	physics, chemistry := hosttest.New("Physics"), hosttest.New("Chemistry")
	physics.Beside(chemistry)

	if err := physics.OpenVault(t.Context(), chemistry.Vault().ID); err != nil {
		t.Fatal(err)
	}

	opened, isOpened := physics.GetOpenedVault()
	if !isOpened || opened.ID != chemistry.Vault().ID {
		t.Errorf("got %+v, %v", opened, isOpened)
	}
	if err := physics.OpenVault(t.Context(), "nowhere"); !errors.Is(err, host.ErrUnknownVault) {
		t.Errorf("got %v", err)
	}
}

func TestAddVaultOfFolderContainingVaultReturnsTheCoreText(t *testing.T) {
	physics := hosttest.New("Physics")

	_, err := physics.AddVault(t.Context(), "/vaults", "")

	if !errors.Is(err, host.ErrVaultOverlaps) {
		t.Fatalf("AddVault of /vaults returned %v, want ErrVaultOverlaps", err)
	}
	if want := "/vaults contains the vault Physics; a vault cannot contain another vault"; err.Error() != want {
		t.Errorf("AddVault of /vaults returned %q, want %q", err.Error(), want)
	}
}

func TestAddVaultOfFolderInsideVaultReturnsTheCoreText(t *testing.T) {
	physics := hosttest.New("Physics")

	_, err := physics.AddVault(t.Context(), "/vaults/Physics/optics", "")

	if !errors.Is(err, host.ErrVaultOverlaps) {
		t.Fatalf("AddVault of /vaults/Physics/optics returned %v, want ErrVaultOverlaps", err)
	}
	if want := "/vaults/Physics/optics is inside the vault Physics; a vault cannot be inside another vault"; err.Error() != want {
		t.Errorf("AddVault of /vaults/Physics/optics returned %q, want %q", err.Error(), want)
	}
}
