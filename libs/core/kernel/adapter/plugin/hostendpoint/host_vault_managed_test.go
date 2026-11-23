package hostendpoint

import (
	"fmt"
	"slices"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"

	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
)

func TestAPluginWithVaultManageAddsAVaultItManages(t *testing.T) {
	list := newRecordingList()
	client, _, _ := newArrangeClient(t, list, manifest.PermissionVaultManage)

	res, err := client.AddVault(t.Context(), connect.NewRequest(&v1.AddVaultRequest{Path: "/vaults/Seas", Name: "Seas", ManagedBy: "md.numen.a"}))
	if err != nil {
		t.Fatal(err)
	}

	if want := []string{"add /vaults/Seas Seas managed-by md.numen.a"}; !slices.Equal(list.getCalls(), want) || res.Msg.GetError() != 0 {
		t.Errorf("the list received %v and the answer was %v, want %v", list.getCalls(), res.Msg, want)
	}
}

func TestAPluginCannotAddAVaultManagedByAnotherPlugin(t *testing.T) {
	list := newRecordingList()
	client, _, _ := newArrangeClient(t, list, manifest.PermissionVaultManage)

	_, err := client.AddVault(t.Context(), connect.NewRequest(&v1.AddVaultRequest{Path: "/vaults/Seas", ManagedBy: "md.numen.b"}))

	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("error = %v, want invalid argument", err)
	}
	if calls := list.getCalls(); len(calls) != 0 {
		t.Errorf("the list received %v for a vault that names another plugin as manager", calls)
	}
}

func TestAPluginWithoutVaultManageCannotAddAManagedVault(t *testing.T) {
	list := newRecordingList()
	client, _, _ := newArrangeClient(t, list, manifest.PermissionVaultWrite)

	_, err := client.AddVault(t.Context(), connect.NewRequest(&v1.AddVaultRequest{Path: "/vaults/Seas", ManagedBy: "md.numen.a"}))

	if !isDenied(err) || len(list.getCalls()) != 0 {
		t.Errorf("error = %v, calls = %v, want permission denied and no call", err, list.getCalls())
	}
}

func TestTheListOfVaultsReturnsTheManagerOfAManagedVault(t *testing.T) {
	client, _, _ := newArrangeClient(t, newRecordingList(), manifest.PermissionVaultRead)

	res, err := client.ListVaults(t.Context(), connect.NewRequest(&v1.ListVaultsRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	vaults := res.Msg.GetVaults()
	if len(vaults) != 2 || vaults[0].GetManagedBy() != "" || vaults[1].GetManagedBy() != "md.numen.a" {
		t.Errorf("the vaults were %v, want only v2 managed by md.numen.a", vaults)
	}
}

func TestAPluginRemovesOnlyAVaultItManages(t *testing.T) {
	list := newRecordingList()
	client, _, _ := newArrangeClient(t, list, manifest.PermissionVaultManage)

	_, err := client.RemoveVault(t.Context(), connect.NewRequest(&v1.RemoveVaultRequest{Id: "v1"}))

	if !isDenied(err) {
		t.Errorf("error = %v, want permission denied for a vault of the person", err)
	}
	if calls := list.getCalls(); len(calls) != 0 {
		t.Errorf("the list received %v for a vault the plugin does not manage", calls)
	}
}

func TestAPluginRemovesAVaultItManages(t *testing.T) {
	list := newRecordingList()
	client, _, _ := newArrangeClient(t, list, manifest.PermissionVaultManage)

	res, err := client.RemoveVault(t.Context(), connect.NewRequest(&v1.RemoveVaultRequest{Id: "v2"}))
	if err != nil {
		t.Fatal(err)
	}

	if want := []string{"remove v2"}; !slices.Equal(list.getCalls(), want) || res.Msg.GetError() != 0 {
		t.Errorf("the list received %v and the answer was %v, want %v", list.getCalls(), res.Msg, want)
	}
}

func TestAPluginCannotTakeAVaultOfThePersonAsItsOwn(t *testing.T) {
	list := newRecordingList()
	list.err = fmt.Errorf("add: %w", vaultdomain.ErrManagedExisting)
	client, _, _ := newArrangeClient(t, list, manifest.PermissionVaultManage)

	_, err := client.AddVault(t.Context(), connect.NewRequest(&v1.AddVaultRequest{Path: "/vaults/Rivers", ManagedBy: "md.numen.a"}))

	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("error = %v, want invalid argument", err)
	}
}
