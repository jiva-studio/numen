// Package identityseed makes the test identities of the acceptance scenarios: one person with recovery words and three
// computers that hold the keys of the account, and the saved state of the fake sync server that knows that account.
// The words and the keys come from fixed seeds, so the account and the devices are always the same; the sealed parts hold
// random nonces. The keys are for tests only.
package identityseed

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/tools/syncserver/fakeserver"
)

const (
	// IdentityFile holds the words, the chain and the keys of the computers.
	IdentityFile = "identity.json"
	// ServerFile is the saved state of the fake sync server, in the layout the server reads with -state-dir.
	ServerFile = "server.json"
	// Purpose is written into IdentityFile and marks the keys as keys of tests.
	Purpose = "TEST ONLY: keys of the acceptance scenarios, never used for a real identity. Regenerate with npm run acceptance:identity."

	keyName        = "identity"
	highWaterName  = "chain-length-"
	firstEntryTime = "2026-01-01T00:00:00Z"
	serverFormat   = 2
)

// ComputerNames are the computers of the person, in the order they join the chain.
var ComputerNames = []string{"first", "second", "third"}

var deviceTitles = map[string]string{"first": "First computer", "second": "Second computer", "third": "Third computer"}

// Computer is what one computer keeps of the identity: the entries of its key store.
type Computer struct {
	DeviceFingerprint string            `json:"deviceFingerprint"`
	Credentials       map[string]string `json:"credentials"`
}

// File is the content of IdentityFile.
type File struct {
	Purpose     string              `json:"purpose"`
	Words       []string            `json:"words"`
	Fingerprint string              `json:"fingerprint"`
	Chain       []byte              `json:"chain"`
	Computers   map[string]Computer `json:"computers"`
}

// Seed is everything the scenarios load: the identity file and the saved state of the server.
type Seed struct {
	Identity File
	Server   []byte
}

type stream struct {
	label   string
	counter uint64
	block   []byte
}

func newStream(label string) *stream { return &stream{label: label} }

func (s *stream) Read(p []byte) (int, error) {
	for n := 0; n < len(p); {
		if len(s.block) == 0 {
			var counter [8]byte
			binary.BigEndian.PutUint64(counter[:], s.counter)
			sum := sha256.Sum256(append([]byte("numen acceptance identity "+s.label), counter[:]...))
			s.block, s.counter = sum[:], s.counter+1
		}
		copied := copy(p[n:], s.block)
		s.block = s.block[copied:]
		n += copied
	}
	return len(p), nil
}

// Generate makes the seed with the code of the identity library as it is now.
func Generate() (Seed, error) {
	words, err := identity.NewRecoveryWords(newStream("words"))
	if err != nil {
		return Seed{}, err
	}
	root, err := words.DeriveRecoveryKey()
	if err != nil {
		return Seed{}, err
	}
	account, err := identity.NewAccountKeys(newStream("account"))
	if err != nil {
		return Seed{}, err
	}
	at, err := time.Parse(time.RFC3339, firstEntryTime)
	if err != nil {
		return Seed{}, err
	}
	chain, err := identity.NewChain(root, account, at)
	if err != nil {
		return Seed{}, err
	}
	devices := map[string]identity.DeviceKey{}
	for i, name := range ComputerNames {
		device, err := identity.NewDeviceKey(newStream("device " + name))
		if err != nil {
			return Seed{}, err
		}
		devices[name] = device
		chain, err = chain.AppendDevice(root, device.Public(), deviceTitles[name], at.Add(time.Duration(i+1)*time.Minute))
		if err != nil {
			return Seed{}, err
		}
	}
	file := File{Purpose: Purpose, Words: words.List(), Fingerprint: chain.Root().Fingerprint().String(), Chain: chain.Bytes(), Computers: map[string]Computer{}}
	for _, name := range ComputerNames {
		wrapped, err := account.WrapForDevice(devices[name].Public())
		if err != nil {
			return Seed{}, err
		}
		record := identity.Record{Device: devices[name], Wrapped: wrapped}
		file.Computers[name] = Computer{
			DeviceFingerprint: devices[name].Public().Fingerprint().String(),
			Credentials: map[string]string{
				keyName:                          base64.RawURLEncoding.EncodeToString(record.Bytes()),
				highWaterName + file.Fingerprint: strconv.Itoa(chain.Len()),
			},
		}
	}
	server, err := marshalServer(file.Fingerprint, chain.Bytes())
	if err != nil {
		return Seed{}, err
	}
	return Seed{Identity: file, Server: server}, nil
}

func marshalServer(fingerprint string, chain []byte) ([]byte, error) {
	return json.Marshal(map[string]any{
		"format": serverFormat, "chains": map[string][]byte{fingerprint: chain}, "catalogs": map[string]any{}, "vaults": map[string]any{},
		"blobs": map[string]any{}, "used": map[string]any{}, "subscriptions": map[string]any{}, "plans": map[string]any{},
		"trials": map[string]any{}, "partLog": []any{}, "shares": map[string]any{},
	})
}

func Write(folder string, seed Seed) error {
	raw, err := json.MarshalIndent(seed.Identity, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(folder, 0o750); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(folder, IdentityFile), append(raw, '\n'), 0o600); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(folder, ServerFile), seed.Server, 0o600)
}

// ErrOutOfDate says that the saved seed is not what the code makes now.
var ErrOutOfDate = errors.New("the test identities are out of date: run npm run acceptance:identity")

// Check loads the seed in the folder the way the application and the fake server do: the words give the account of the
// chain, every record opens for its device in the chain, and the server state restores. The sealed parts hold random
// nonces, so two runs of Generate differ in bytes and only a load can tell a stale file.
func Check(folder string) error {
	saved, err := load(folder)
	if err != nil {
		return fmt.Errorf("%w (%w)", ErrOutOfDate, err)
	}
	if err := verify(saved); err != nil {
		return fmt.Errorf("%w (%w)", ErrOutOfDate, err)
	}
	return nil
}

func load(folder string) (Seed, error) {
	var seed Seed
	raw, err := os.ReadFile(filepath.Join(folder, IdentityFile))
	if err != nil {
		return seed, err
	}
	if err := json.Unmarshal(raw, &seed.Identity); err != nil {
		return seed, err
	}
	seed.Server, err = os.ReadFile(filepath.Join(folder, ServerFile))
	return seed, err
}

func verify(seed Seed) error {
	chain, err := identity.ParseChain(seed.Identity.Chain)
	if err != nil {
		return err
	}
	words, err := identity.ParseRecoveryWords(seed.Identity.Words)
	if err != nil {
		return err
	}
	root, err := words.DeriveRecoveryKey()
	if err != nil {
		return err
	}
	if root.Public().Fingerprint().String() != seed.Identity.Fingerprint || chain.Root().Fingerprint().String() != seed.Identity.Fingerprint {
		return errors.New("the words and the chain are of different accounts")
	}
	if err := chain.Verify(); err != nil {
		return err
	}
	for _, name := range ComputerNames {
		entry, found := seed.Identity.Computers[name]
		if !found {
			return fmt.Errorf("no computer %q", name)
		}
		raw, err := base64.RawURLEncoding.DecodeString(entry.Credentials[keyName])
		if err != nil {
			return err
		}
		record, err := identity.ParseRecord(raw)
		if err != nil {
			return err
		}
		if _, err := identity.NewState(record, chain); err != nil {
			return err
		}
	}
	return fakeserver.New(fakeserver.Config{}).Restore(seed.Server)
}
