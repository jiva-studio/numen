//go:build numen_debug

package app

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/identityfile"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	domainkeyfile "github.com/jiva-studio/numen/libs/core/kernel/domain/keyfile"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

func TestDebugHTTPReadsAndPersistsIdentityFilesInTheTemporaryProfile(t *testing.T) {
	makeDebugProfile(t, settings.CredentialsFilePlaintext)
	opts := DebugOptions{HTTPAddr: "127.0.0.1:0"}
	res := container.NewResources()
	if err := opts.apply(func() {}, res); err != nil {
		t.Fatal(err)
	}
	requireDebugIdentity(t, res, "seed record", "seed chain")
	state, err := res.KeyFile.GetState(t.Context())
	if err != nil || !state.IsInUse || state.Mode != domainkeyfile.Plaintext {
		t.Fatalf("key file state = %v, %v, want plaintext", state, err)
	}

	if err := res.Stores.Keys.SetKey(t.Context(), "identity", "replacement record"); err != nil {
		t.Fatal(err)
	}
	if err := res.Stores.Chains.Write(t.Context(), []byte("replacement chain")); err != nil {
		t.Fatal(err)
	}
	restart := container.NewResources()
	if err := opts.apply(func() {}, restart); err != nil {
		t.Fatal(err)
	}
	requireDebugIdentity(t, restart, "replacement record", "replacement chain")
}

func TestDebugHTTPKeepsMemoryStoresWithoutAPlaintextLoopbackProfile(t *testing.T) {
	for _, scenario := range []struct {
		name string
		addr string
		mode string
	}{
		{"ordinary debug profile", "127.0.0.1:0", settings.CredentialsFileUnset},
		{"passphrase profile", "127.0.0.1:0", settings.CredentialsFilePassphrase},
		{"all interfaces", "0.0.0.0:0", settings.CredentialsFilePlaintext},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			makeDebugProfile(t, scenario.mode)
			requireEmptyDebugIdentity(t, scenario.addr)
		})
	}
}

func TestDebugHTTPPersistsAnIdentityCreatedInAnEmptyPlaintextProfile(t *testing.T) {
	folder := makeDebugProfile(t, settings.CredentialsFilePlaintext)
	for _, name := range []string{"credentials", "identity"} {
		if err := os.RemoveAll(filepath.Join(folder, name)); err != nil {
			t.Fatal(err)
		}
	}
	opts := DebugOptions{HTTPAddr: "127.0.0.1:0"}
	res := container.NewResources()
	if err := opts.apply(func() {}, res); err != nil {
		t.Fatal(err)
	}
	if err := res.Stores.Keys.SetKey(t.Context(), "identity", "new record"); err != nil {
		t.Fatal(err)
	}
	if err := res.Stores.Chains.Write(t.Context(), []byte("new chain")); err != nil {
		t.Fatal(err)
	}
	restart := container.NewResources()
	if err := opts.apply(func() {}, restart); err != nil {
		t.Fatal(err)
	}
	requireDebugIdentity(t, restart, "new record", "new chain")
}

func TestDebugHTTPDoesNotOpenIdentityFilesOutsideTheTemporaryProfile(t *testing.T) {
	folder := makeDebugProfile(t, settings.CredentialsFilePlaintext)
	outside := filepath.Join(t.TempDir(), "numen")
	if err := os.Rename(folder, outside); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, folder); err != nil {
		t.Skipf("cannot create a symlink: %v", err)
	}
	requireEmptyDebugIdentity(t, "127.0.0.1:0")
}

func TestDebugHTTPDoesNotFollowAnIdentityFileSymlinkOutsideTheProfile(t *testing.T) {
	folder := makeDebugProfile(t, settings.CredentialsFilePlaintext)
	credentials := filepath.Join(folder, "credentials", "credentials.json")
	outside := filepath.Join(t.TempDir(), "credentials.json")
	if err := os.Rename(credentials, outside); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, credentials); err != nil {
		t.Skipf("cannot create a symlink: %v", err)
	}
	requireEmptyDebugIdentity(t, "127.0.0.1:0")
}

func requireDebugIdentity(t *testing.T, res *container.Resources, record, chain string) {
	t.Helper()
	if got, err := res.Stores.Keys.GetKey(t.Context(), "identity"); err != nil || got != record {
		t.Fatalf("identity record = %q, %v, want %q", got, err, record)
	}
	if got, err := res.Stores.Chains.Read(t.Context()); err != nil || string(got) != chain {
		t.Fatalf("identity chain = %q, %v, want %q", got, err, chain)
	}
}

func requireEmptyDebugIdentity(t *testing.T, addr string) {
	t.Helper()
	opts := DebugOptions{HTTPAddr: addr}
	res := container.NewResources()
	if err := opts.apply(func() {}, res); err != nil {
		t.Fatal(err)
	}
	if _, err := res.Stores.Keys.GetKey(t.Context(), "identity"); !errors.Is(err, plugin.ErrNoKey) {
		t.Fatalf("read identity key: %v, want ErrNoKey", err)
	}
	if _, err := res.Stores.Chains.Read(t.Context()); !errors.Is(err, identitydomain.ErrNoChain) {
		t.Fatalf("read chain: %v, want ErrNoChain", err)
	}
}

func makeDebugProfile(t *testing.T, mode string) string {
	t.Helper()
	root := t.TempDir()
	home := filepath.Join(root, "home")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("AppData", filepath.Join(root, "config"))
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	path, err := settings.GetPath()
	if err != nil {
		t.Fatal(err)
	}
	folder := filepath.Dir(path)
	credentials := filepath.Join(folder, "credentials")
	if err := os.MkdirAll(credentials, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"v":1,"app":{"keys":{"credentialsFile":"`+mode+`"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(credentials, "credentials.json"), []byte(`{"identity":"seed record"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := identityfile.Open(folder).Write(t.Context(), []byte("seed chain")); err != nil {
		t.Fatal(err)
	}
	return folder
}
