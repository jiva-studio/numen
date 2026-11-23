package identity_test

import (
	"bytes"
	"encoding/base64"
	"errors"
	"testing"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

const sealContext = "numen-sync/catalog/v1"

type sealKit struct {
	kit
	seal *identity.SealData
	open *identity.OpenData
}

func newSealKit(tb testing.TB, withIdentity bool) sealKit {
	tb.Helper()
	k := newKit("Laptop")
	deps := identity.NewDeps(k.keys, fakeNames{name: "Laptop"}, testsupport.NewClockAt(getNow())).WithChains(k.chains)
	if withIdentity {
		if _, err := k.create.Execute(tb.Context(), getLegalWords()); err != nil {
			tb.Fatal(err)
		}
	}
	return sealKit{kit: k, seal: identity.NewSealData(deps), open: identity.NewOpenData(deps)}
}

func TestOpenData_ReturnsWhatSealDataSealedForThePluginAndTheContext(t *testing.T) {
	t.Parallel()
	k := newSealKit(t, true)

	sealed, err := k.seal.Execute(t.Context(), syncPlugin, []byte(sealContext), []byte("Shopping list"))
	if err != nil {
		t.Fatal(err)
	}
	opened, err := k.open.Execute(t.Context(), syncPlugin, []byte(sealContext), sealed)

	if err != nil || string(opened) != "Shopping list" {
		t.Errorf("opened = %q, error = %v, want the plain text", opened, err)
	}
	if bytes.Contains(sealed, []byte("Shopping")) {
		t.Error("the sealed bytes hold the plain text")
	}
}

func TestSealData_SealsWithTheAccountKeysOfTheRecord(t *testing.T) {
	t.Parallel()
	k := newSealKit(t, true)
	raw, err := base64.RawURLEncoding.DecodeString(k.keys.keys[keyName])
	if err != nil {
		t.Fatal(err)
	}
	record, err := identitylib.ParseRecord(raw)
	if err != nil {
		t.Fatal(err)
	}
	keys, err := record.Device.UnwrapAccountKeys(record.Wrapped)
	if err != nil {
		t.Fatal(err)
	}

	sealed, err := k.seal.Execute(t.Context(), syncPlugin, []byte(sealContext), []byte("milk"))
	if err != nil {
		t.Fatal(err)
	}

	opened, err := keys.OpenForPlugin(syncPlugin, []byte(sealContext), sealed)
	if err != nil || string(opened) != "milk" {
		t.Errorf("the library opened %q with error %v, want milk", opened, err)
	}
}

func TestSealData_ReturnsNewBytesForTheSamePlainTextEachTime(t *testing.T) {
	t.Parallel()
	k := newSealKit(t, true)

	first, err := k.seal.Execute(t.Context(), syncPlugin, []byte(sealContext), []byte("milk"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := k.seal.Execute(t.Context(), syncPlugin, []byte(sealContext), []byte("milk"))
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(first, second) {
		t.Error("two seals of one plain text are equal")
	}
}

func TestOpenData_RefusesAnotherPluginAnotherContextAndChangedBytes(t *testing.T) {
	t.Parallel()
	k := newSealKit(t, true)
	sealed, err := k.seal.Execute(t.Context(), syncPlugin, []byte(sealContext), []byte("milk"))
	if err != nil {
		t.Fatal(err)
	}
	changed := bytes.Clone(sealed)
	changed[len(changed)-1] ^= 1

	for name, call := range map[string]func() ([]byte, error){
		"another plugin": func() ([]byte, error) {
			return k.open.Execute(t.Context(), "md.numen.tools.link", []byte(sealContext), sealed)
		},
		"another context": func() ([]byte, error) { return k.open.Execute(t.Context(), syncPlugin, []byte("other"), sealed) },
		"changed bytes":   func() ([]byte, error) { return k.open.Execute(t.Context(), syncPlugin, []byte(sealContext), changed) },
		"no bytes":        func() ([]byte, error) { return k.open.Execute(t.Context(), syncPlugin, []byte(sealContext), nil) },
	} {
		opened, err := call()
		if !errors.Is(err, identitydomain.ErrCannotOpen) || opened != nil {
			t.Errorf("%s: opened = %q, error = %v, want nil and ErrCannotOpen", name, opened, err)
		}
	}
}

func TestSealDataAndOpenData_ReportNoIdentityWhenTheKeyStoreHasNoRecord(t *testing.T) {
	t.Parallel()
	k := newSealKit(t, false)

	_, sealErr := k.seal.Execute(t.Context(), syncPlugin, []byte(sealContext), []byte("milk"))
	_, openErr := k.open.Execute(t.Context(), syncPlugin, []byte(sealContext), []byte("sealed"))

	if !errors.Is(sealErr, identitydomain.ErrNoIdentity) || !errors.Is(openErr, identitydomain.ErrNoIdentity) {
		t.Errorf("seal = %v, open = %v, want ErrNoIdentity", sealErr, openErr)
	}
}

func TestSealDataAndOpenData_ReportAKeyStoreFailureAndNeverCallItNoIdentity(t *testing.T) {
	t.Parallel()
	k := newSealKit(t, true)
	k.keys.getFails = errBroken

	_, sealErr := k.seal.Execute(t.Context(), syncPlugin, []byte(sealContext), []byte("milk"))
	_, openErr := k.open.Execute(t.Context(), syncPlugin, []byte(sealContext), []byte("sealed"))

	for name, err := range map[string]error{"seal": sealErr, "open": openErr} {
		if !errors.Is(err, errBroken) || errors.Is(err, identitydomain.ErrNoIdentity) {
			t.Errorf("%s: error = %v, want the key store failure and not ErrNoIdentity", name, err)
		}
	}
}

func TestSealDataAndOpenData_ReportARecordThatDoesNotParse(t *testing.T) {
	t.Parallel()
	k := newSealKit(t, false)
	k.keys.keys[keyName] = "not-a-record"

	_, sealErr := k.seal.Execute(t.Context(), syncPlugin, []byte(sealContext), []byte("milk"))
	_, openErr := k.open.Execute(t.Context(), syncPlugin, []byte(sealContext), []byte("sealed"))

	if !errors.Is(sealErr, identitylib.ErrMalformed) || !errors.Is(openErr, identitylib.ErrMalformed) {
		t.Errorf("seal = %v, open = %v, want ErrMalformed", sealErr, openErr)
	}
}

func TestSealDataAndOpenData_DoNothingForAnEmptyPluginID(t *testing.T) {
	t.Parallel()
	k := newSealKit(t, true)
	sealed, err := k.seal.Execute(t.Context(), syncPlugin, []byte(sealContext), []byte("milk"))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := k.seal.Execute(t.Context(), "", []byte(sealContext), []byte("milk")); err == nil {
		t.Error("data was sealed with no plugin id")
	}
	if _, err := k.open.Execute(t.Context(), "", []byte(sealContext), sealed); err == nil {
		t.Error("data was opened with no plugin id")
	}
}

func TestSealData_ChangesNothingInTheKeyStore(t *testing.T) {
	t.Parallel()
	k := newSealKit(t, true)
	before := k.keys.snapshot()
	writes := k.keys.setCalled

	if _, err := k.seal.Execute(t.Context(), syncPlugin, []byte(sealContext), []byte("milk")); err != nil {
		t.Fatal(err)
	}

	if after := k.keys.snapshot(); after[keyName] != before[keyName] || len(after) != len(before) || k.keys.setCalled != writes {
		t.Error("sealing data wrote to the key store")
	}
}
