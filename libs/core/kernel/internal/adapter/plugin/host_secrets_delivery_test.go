package plugin_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// brokenKeys fails to read the named entries and reads the others from the store it wraps.
type brokenKeys struct {
	*keys
	broken map[string]bool
}

func (b brokenKeys) GetKey(ctx context.Context, name string) (string, error) {
	if b.broken[name] {
		return "", errLockedKeychain
	}
	return b.keys.GetKey(ctx, name)
}

func getTwoSecrets() plugindomain.SecretSource {
	allowed := []plugindomain.DeclarationKey{{Plugin: secretPluginID, Name: declared}}
	return plugindomain.SecretSource{List: func() ([]plugindomain.Secret, error) {
		return []plugindomain.Secret{
			{ID: "s00000001", Name: "First", Group: "openrouter", Allowed: allowed},
			{ID: "s00000002", Name: "Second", Group: "openrouter", Allowed: allowed},
		}, nil
	}}
}

func TestDeclarationWithFailedReadSendsReadValues(t *testing.T) {
	cases := map[string]struct {
		broken string
		want   []string
	}{
		"the second value fails": {broken: "secret/s00000002", want: []string{"s00000001"}},
		"the first value fails":  {broken: "secret/s00000001", want: nil},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			store := brokenKeys{
				keys:   newKeys(map[string]string{"secret/s00000001": "invented-one", "secret/s00000002": "invented-two"}),
				broken: map[string]bool{tc.broken: true},
			}
			host, probe := newSecretHost(t, store)
			host.Config.SetSecretSource(getTwoSecrets())

			err := host.Config.NotifyPluginConfig(t.Context(), secretPluginID)
			if !errors.Is(err, errLockedKeychain) {
				t.Errorf("error = %v, want the error of the store", err)
			}
			var got []string
			for _, value := range listed(t, probe, declared) {
				got = append(got, value.GetId())
			}
			if len(got) != len(tc.want) || (len(got) == 1 && got[0] != tc.want[0]) {
				t.Errorf("the entry carries %v, want %v", got, tc.want)
			}
		})
	}
}

// A list that was sent while a value could not be read is not recorded as held, so the next refresh
// sends it again and the plugin gets the value once the key store answers.
func TestAListSentWithAFailedReadIsSentAgainByTheNextRefresh(t *testing.T) {
	store := newKeys(map[string]string{"secret/" + secretID: fromKeychain})
	store.failed = errLockedKeychain
	host, probe := newSecretHost(t, store)
	host.Config.SetSecretSource(allowOne(secretPluginID))
	if err := host.Config.NotifyPluginConfig(t.Context(), secretPluginID); err == nil {
		t.Fatal("the read failure was not returned")
	}
	store.failed = nil

	if err := host.Config.Refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	if got := listed(t, probe, declared); len(got) != 1 || got[0].GetValue() != fromKeychain {
		t.Errorf("the plugin holds %v, want the value", got)
	}
}

// countedList counts the reads of the list of secrets.
type countedList struct {
	plugindomain.SecretSource
	reads atomic.Int32
}

func TestOneConfigureReadsTheListOfSecretsOnce(t *testing.T) {
	host, _ := newSecretHost(t, newKeys(map[string]string{"secret/" + secretID: fromKeychain}))
	counted := &countedList{SecretSource: allowOne(secretPluginID)}
	read := counted.List
	counted.List = func() ([]plugindomain.Secret, error) {
		counted.reads.Add(1)
		return read()
	}
	host.Config.SetSecretSource(counted.SecretSource)

	if err := host.Config.NotifyPluginConfig(t.Context(), secretPluginID); err != nil {
		t.Fatal(err)
	}
	if got := counted.reads.Load(); got != 1 {
		t.Errorf("the list was read %d times for one Configure, want 1", got)
	}
}

// A list that cannot be read is sent as an empty entry for each declaration, so the plugin drops its
// secrets, and is not recorded as held.
func TestAListThatCannotBeReadIsSentEmptyAndReturnedAsAnError(t *testing.T) {
	host, probe := newSecretHost(t, newKeys(map[string]string{"secret/" + secretID: fromKeychain}))
	host.Config.SetSecretSource(allowOne(secretPluginID))
	if err := host.Config.NotifyPluginConfig(t.Context(), secretPluginID); err != nil {
		t.Fatal(err)
	}
	errDisk := errors.New("the disk failed")
	host.Config.SetSecretSource(plugindomain.SecretSource{List: func() ([]plugindomain.Secret, error) { return nil, errDisk }})

	if err := host.Config.NotifyPluginConfig(t.Context(), secretPluginID); !errors.Is(err, errDisk) {
		t.Errorf("error = %v, want the error of the list", err)
	}
	if got := listed(t, probe, declared); len(got) != 0 {
		t.Errorf("the entry carries %v, want none", got)
	}
}
