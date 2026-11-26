package platform

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/zalando/go-keyring"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/identity"
	sharedidentity "github.com/jiva-studio/numen/libs/shared/identity"
)

var seededComputers = []string{"first", "second", "third"}

type seedFile struct {
	Fingerprint string
	Chain       []byte
	Computers   map[string]struct {
		Credentials map[string]string
	}
}

type fixedClock struct{}

func (fixedClock) Now() time.Time                            { return time.Date(2026, 11, 25, 0, 0, 0, 0, time.UTC) }
func (fixedClock) StartTimer(time.Duration) <-chan time.Time { return nil }

type namedDevice struct{}

func (namedDevice) GetInstanceName() string { return "Test computer" }

func makeSeededIdentity(t *testing.T) seedFile {
	t.Helper()
	words, err := sharedidentity.NewRecoveryWords(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	root, err := words.DeriveRecoveryKey()
	if err != nil {
		t.Fatal(err)
	}
	account, err := sharedidentity.NewAccountKeys(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	at := fixedClock{}.Now().Add(-time.Hour)
	chain, err := sharedidentity.NewChain(root, account, at)
	if err != nil {
		t.Fatal(err)
	}
	devices := map[string]sharedidentity.DeviceKey{}
	for i, name := range seededComputers {
		if devices[name], err = sharedidentity.NewDeviceKey(rand.Reader); err != nil {
			t.Fatal(err)
		}
		if chain, err = chain.AppendDevice(root, devices[name].Public(), name, at.Add(time.Duration(i+1)*time.Minute)); err != nil {
			t.Fatal(err)
		}
	}
	seed := seedFile{Fingerprint: chain.Root().Fingerprint().String(), Chain: chain.Bytes(), Computers: map[string]struct{ Credentials map[string]string }{}}
	for _, name := range seededComputers {
		wrapped, err := account.WrapForDevice(devices[name].Public())
		if err != nil {
			t.Fatal(err)
		}
		record := sharedidentity.Record{Device: devices[name], Wrapped: wrapped}
		seed.Computers[name] = struct{ Credentials map[string]string }{Credentials: map[string]string{
			"identity":                         base64.RawURLEncoding.EncodeToString(record.Bytes()),
			"chain-length-" + seed.Fingerprint: strconv.Itoa(chain.Len()),
		}}
	}
	return seed
}

// The acceptance scenarios put the identity of a scenario where the application reads it at its start; this test reads one the way a
// window does, through the key store and the chain file of the installation.
func TestTheApplicationReadsTheIdentityOfAScenario(t *testing.T) {
	seed := makeSeededIdentity(t)
	keyring.MockInitWithError(fmt.Errorf("the secret service is not running: %w", keyring.ErrUnsupportedPlatform))
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	path, err := settings.GetPath()
	if err != nil {
		t.Fatal(err)
	}
	folder := filepath.Dir(path)
	for _, computer := range seededComputers {
		t.Run(computer, func(t *testing.T) {
			writeSeededComputer(t, folder, path, seed, computer)
			store := getKeyStore()
			applyCredentialsFile(store, nil)
			account := identity.NewAccount(func() pluginport.KeyStore { return store }, namedDevice{}, fixedClock{}, nil, getChainStore())
			public, err := account.GetPublicIdentity(t.Context())
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
