package identity_test

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/identity"
)

const (
	sealPluginID = "md.numen.tools.sync"
	sealContext  = "numen-sync/catalog/v1"
)

type failingReader struct{ err error }

func (r failingReader) Read([]byte) (int, error) { return 0, r.err }

func allByteValues() []byte {
	values := make([]byte, 256)
	for i := range values {
		values[i] = byte(i)
	}
	return values
}

func newFixedAccount(tb testing.TB, fixed []byte) identity.AccountKeys {
	tb.Helper()
	account, err := identity.NewAccountKeys(bytes.NewReader(fixed))
	if err != nil {
		tb.Fatal(err)
	}
	return account
}

func readSealVector(tb testing.TB, kek, salt []byte, epoch uint32, pluginID, context string, plain []byte) []byte {
	tb.Helper()
	pluginKey, err := hkdf.Key(sha256.New, kek, nil, "numen/v1/plugin/"+pluginID, 32)
	if err != nil {
		tb.Fatal(err)
	}
	subkey, err := hkdf.Key(sha256.New, pluginKey, salt, "numen/v1/seal", 32)
	if err != nil {
		tb.Fatal(err)
	}
	block, err := aes.NewCipher(subkey)
	if err != nil {
		tb.Fatal(err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		tb.Fatal(err)
	}
	associated := binary.BigEndian.AppendUint32([]byte("numen/v1/plugin-seal/"+pluginID+"\x00"), epoch)
	associated = append(associated, context...)
	out := binary.BigEndian.AppendUint32([]byte{0x02}, epoch)
	out = append(out, salt...)
	return aead.Seal(out, make([]byte, 12), plain, associated)
}

func TestSealForPlugin_ProducesTheFrozenFormatForAFixedRandomReader(t *testing.T) {
	t.Parallel()
	fixed := make([]byte, 0, 96)
	for i := range 96 {
		fixed = append(fixed, byte(i+1))
	}
	account := newFixedAccount(t, fixed[:64])
	kek, salt := fixed[32:64], fixed[64:80]
	plain := []byte("milk, bread, eggs")

	sealed, err := account.SealForPlugin(bytes.NewReader(fixed[64:]), sealPluginID, []byte(sealContext), plain)
	if err != nil {
		t.Fatal(err)
	}

	want := readSealVector(t, kek, salt, 0, sealPluginID, sealContext, plain)
	if !bytes.Equal(sealed, want) {
		t.Errorf("sealed = %x\nwant     %x", sealed, want)
	}
	if len(sealed) != 1+4+16+len(plain)+16 {
		t.Errorf("sealed has %d bytes, want %d", len(sealed), 1+4+16+len(plain)+16)
	}
}

func TestOpenForPlugin_ReturnsWhatSealForPluginSealed(t *testing.T) {
	t.Parallel()
	account := newAccount(t)

	for name, plain := range map[string][]byte{
		"empty":     {},
		"text":      []byte("milk"),
		"32 KiB":    bytes.Repeat([]byte{7}, 32<<10),
		"all bytes": allByteValues(),
	} {
		sealed, err := account.SealForPlugin(rand.Reader, sealPluginID, []byte(sealContext), plain)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		opened, err := account.OpenForPlugin(sealPluginID, []byte(sealContext), sealed)
		if err != nil || !bytes.Equal(opened, plain) {
			t.Errorf("%s: opened = %q, error = %v, want the plain text", name, opened, err)
		}
	}
}

func TestSealForPlugin_ReturnsDifferentBytesForTheSamePlainEachTime(t *testing.T) {
	t.Parallel()
	account := newAccount(t)

	first, err := account.SealForPlugin(rand.Reader, sealPluginID, []byte(sealContext), []byte("milk"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := account.SealForPlugin(rand.Reader, sealPluginID, []byte(sealContext), []byte("milk"))
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(first, second) {
		t.Error("two seals of one plain text are equal")
	}
	if bytes.Equal(first[5:21], second[5:21]) {
		t.Error("two seals use the same salt")
	}
}

func TestOpenForPlugin_RefusesAnotherPluginAnotherContextAndAnotherAccount(t *testing.T) {
	t.Parallel()
	account := newAccount(t)
	sealed, err := account.SealForPlugin(rand.Reader, sealPluginID, []byte(sealContext), []byte("milk"))
	if err != nil {
		t.Fatal(err)
	}

	cases := map[string]func() ([]byte, error){
		"another plugin": func() ([]byte, error) {
			return account.OpenForPlugin("md.numen.tools.link", []byte(sealContext), sealed)
		},
		"another context": func() ([]byte, error) {
			return account.OpenForPlugin(sealPluginID, []byte("numen-sync/catalog/v2"), sealed)
		},
		"empty context":   func() ([]byte, error) { return account.OpenForPlugin(sealPluginID, nil, sealed) },
		"another account": func() ([]byte, error) { return newAccount(t).OpenForPlugin(sealPluginID, []byte(sealContext), sealed) },
		"plugin id moved into the context": func() ([]byte, error) {
			return account.OpenForPlugin(sealPluginID+"\x00"+sealContext, nil, sealed)
		},
	}
	for name, open := range cases {
		opened, err := open()
		if !errors.Is(err, identity.ErrUnwrap) || opened != nil {
			t.Errorf("%s: opened = %q, error = %v, want nil and ErrUnwrap", name, opened, err)
		}
	}
}

func TestOpenForPlugin_RefusesAnyChangedByte(t *testing.T) {
	t.Parallel()
	account := newAccount(t)
	sealed, err := account.SealForPlugin(rand.Reader, sealPluginID, []byte(sealContext), []byte("milk and bread"))
	if err != nil {
		t.Fatal(err)
	}

	for i := range sealed {
		changed := bytes.Clone(sealed)
		changed[i] ^= 0x01
		if _, err := account.OpenForPlugin(sealPluginID, []byte(sealContext), changed); !errors.Is(err, identity.ErrUnwrap) {
			t.Fatalf("byte %d changed: error = %v, want ErrUnwrap", i, err)
		}
	}
}

func TestOpenForPlugin_RefusesInputThatIsTooShortOrTruncatedOrLonger(t *testing.T) {
	t.Parallel()
	account := newAccount(t)
	sealed, err := account.SealForPlugin(rand.Reader, sealPluginID, []byte(sealContext), []byte("milk"))
	if err != nil {
		t.Fatal(err)
	}

	for _, size := range []int{0, 1, 4, 5, 20, 21, 32, len(sealed) - 1} {
		if _, err := account.OpenForPlugin(sealPluginID, []byte(sealContext), sealed[:size]); !errors.Is(err, identity.ErrUnwrap) {
			t.Errorf("%d bytes: error = %v, want ErrUnwrap", size, err)
		}
	}
	if _, err := account.OpenForPlugin(sealPluginID, []byte(sealContext), append(bytes.Clone(sealed), 0)); !errors.Is(err, identity.ErrUnwrap) {
		t.Errorf("one byte appended: error = %v, want ErrUnwrap", err)
	}
	if _, err := account.OpenForPlugin(sealPluginID, []byte(sealContext), nil); !errors.Is(err, identity.ErrUnwrap) {
		t.Errorf("nil input: error = %v, want ErrUnwrap", err)
	}
}

func TestOpenForPlugin_RefusesTheFormatsThatAreNotTheCurrentOne(t *testing.T) {
	t.Parallel()
	account := newAccount(t)
	for _, version := range []byte{0x00, 0x01, 0x03} {
		sealed, err := account.SealForPlugin(rand.Reader, sealPluginID, []byte(sealContext), []byte("milk"))
		if err != nil {
			t.Fatal(err)
		}
		sealed[0] = version
		if _, err := account.OpenForPlugin(sealPluginID, []byte(sealContext), sealed); !errors.Is(err, identity.ErrUnwrap) {
			t.Errorf("version %d: error = %v, want ErrUnwrap", version, err)
		}
	}
}

func TestSealForPlugin_NamesTheEpochOfTheKeysItUsed(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	chain, keys := m.revokeSecond(t)
	sealed, err := keys.SealForPlugin(rand.Reader, sealPluginID, []byte(sealContext), []byte("milk"))
	if err != nil {
		t.Fatal(err)
	}
	if got := binary.BigEndian.Uint32(sealed[1:5]); got != chain.Epoch() {
		t.Errorf("the seal names epoch %d, want %d", got, chain.Epoch())
	}
}

func TestOpenForPlugin_RefusesDataOfALaterEpochWithKeysOfAnEarlierOne(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	_, keys := m.revokeSecond(t)
	sealed, err := keys.SealForPlugin(rand.Reader, sealPluginID, []byte(sealContext), []byte("milk"))
	if err != nil {
		t.Fatal(err)
	}
	if opened, err := m.firstKeys.OpenForPlugin(sealPluginID, []byte(sealContext), sealed); !errors.Is(err, identity.ErrUnwrap) || opened != nil {
		t.Errorf("a revoked device opened = %q, error = %v, want nil and ErrUnwrap", opened, err)
	}
}

func TestOpenForPlugin_RefusesAnEpochThatWasChangedInTheSeal(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	_, keys := m.revokeSecond(t)
	sealed, err := keys.SealForPlugin(rand.Reader, sealPluginID, []byte(sealContext), []byte("milk"))
	if err != nil {
		t.Fatal(err)
	}
	binary.BigEndian.PutUint32(sealed[1:5], 0)
	if _, err := m.firstKeys.OpenForPlugin(sealPluginID, []byte(sealContext), sealed); !errors.Is(err, identity.ErrUnwrap) {
		t.Errorf("error = %v, want ErrUnwrap", err)
	}
}

func TestSealForPlugin_ReportsAReaderThatFailsAndSealsNothing(t *testing.T) {
	t.Parallel()
	account := newAccount(t)
	broken := errors.New("no entropy")

	sealed, err := account.SealForPlugin(failingReader{err: broken}, sealPluginID, []byte(sealContext), []byte("milk"))

	if !errors.Is(err, broken) || sealed != nil {
		t.Errorf("sealed = %x, error = %v, want nil and the reader failure", sealed, err)
	}
}

func TestSealForPlugin_OpensOnAnotherDeviceOfTheSameAccount(t *testing.T) {
	t.Parallel()
	account := newAccount(t)
	first, second := newDevice(t), newDevice(t)
	wrappedFirst, err := account.WrapForDevice(first.Public())
	if err != nil {
		t.Fatal(err)
	}
	wrappedSecond, err := account.WrapForDevice(second.Public())
	if err != nil {
		t.Fatal(err)
	}
	keysFirst, err := first.UnwrapAccountKeys(wrappedFirst)
	if err != nil {
		t.Fatal(err)
	}
	keysSecond, err := second.UnwrapAccountKeys(wrappedSecond)
	if err != nil {
		t.Fatal(err)
	}

	sealed, err := keysFirst.SealForPlugin(rand.Reader, sealPluginID, []byte(sealContext), []byte("milk"))
	if err != nil {
		t.Fatal(err)
	}
	opened, err := keysSecond.OpenForPlugin(sealPluginID, []byte(sealContext), sealed)

	if err != nil || string(opened) != "milk" {
		t.Errorf("opened = %q, error = %v, want milk", opened, err)
	}
}
