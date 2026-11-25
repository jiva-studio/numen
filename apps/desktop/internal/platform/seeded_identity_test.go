package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/zalando/go-keyring"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/identity"
)

type seedFile struct {
	Fingerprint string `json:"fingerprint"`
	Chain       []byte `json:"chain"`
	Computers   map[string]struct {
		Credentials map[string]string `json:"credentials"`
	} `json:"computers"`
}

type fixedClock struct{}

func (fixedClock) Now() time.Time                            { return time.Date(2026, 11, 25, 0, 0, 0, 0, time.UTC) }
func (fixedClock) StartTimer(time.Duration) <-chan time.Time { return nil }

type namedDevice struct{}

func (namedDevice) GetInstanceName() string { return "Test computer" }

// The acceptance scenarios put the saved test identities where the application reads them at its start; this test reads
// them the way a window does, through the key store and the chain file of the installation.
func TestTheApplicationReadsTheSavedTestIdentities(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "tests", "acceptance", "desktop", "assets", "identities", "identity.json"))
	if err != nil {
		t.Fatalf("the test identities are out of date: run npm run acceptance:identity (%v)", err)
	}
	var seed seedFile
	if err := json.Unmarshal(raw, &seed); err != nil {
		t.Fatal(err)
	}
	keyring.MockInitWithError(fmt.Errorf("the secret service is not running: %w", keyring.ErrUnsupportedPlatform))
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	path, err := settings.GetPath()
	if err != nil {
		t.Fatal(err)
	}
	folder := filepath.Dir(path)
	for _, computer := range []string{"first", "second", "third"} {
		t.Run(computer, func(t *testing.T) {
			writeSeededComputer(t, folder, path, seed, computer)
			store := getKeyStore()
			applyCredentialsFile(store, nil)
			account := identity.NewAccount(func() pluginport.KeyStore { return store }, namedDevice{}, fixedClock{}, nil, getChainStore())
			public, err := account.GetPublicIdentity(context.Background())
			if err != nil {
				t.Fatalf("the application finds no identity: %v", err)
			}
			if public.AccountFingerprint != seed.Fingerprint {
				t.Fatalf("account %q, want %q", public.AccountFingerprint, seed.Fingerprint)
			}
		})
	}
}

func writeSeededComputer(t *testing.T, folder, settingsPath string, seed seedFile, computer string) {
	t.Helper()
	if err := os.RemoveAll(folder); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(folder, "credentials"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(folder, "identity"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settingsPath, []byte(`{"v":1,"app":{"keys":{"credentialsFile":"plaintext"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	credentials, err := json.Marshal(seed.Computers[computer].Credentials)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "credentials", "credentials.json"), credentials, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "identity", "chain.bin"), seed.Chain, 0o600); err != nil {
		t.Fatal(err)
	}
}
