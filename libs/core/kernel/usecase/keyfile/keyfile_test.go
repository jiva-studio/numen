package keyfile_test

import (
	"context"
	"errors"
	"testing"

	domainkeyfile "github.com/jiva-studio/numen/libs/core/kernel/domain/keyfile"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/keyfile"
)

type recordingFile struct {
	state       domainkeyfile.State
	plaintext   int
	passphrases []string
	unlocked    []string
	fail        error
}

func (f *recordingFile) GetState(context.Context) (domainkeyfile.State, error) {
	return f.state, f.fail
}

func (f *recordingFile) UsePlaintext(context.Context) error {
	f.plaintext++
	return f.fail
}

func (f *recordingFile) UsePassphrase(_ context.Context, passphrase string) error {
	f.passphrases = append(f.passphrases, passphrase)
	return f.fail
}

func (f *recordingFile) Unlock(_ context.Context, passphrase string) error {
	f.unlocked = append(f.unlocked, passphrase)
	return f.fail
}

func TestChooseAppliesPlaintextAndRemembersIt(t *testing.T) {
	file := &recordingFile{}
	var remembered []domainkeyfile.Mode
	choose := keyfile.NewChoose(file, func(mode domainkeyfile.Mode) error {
		remembered = append(remembered, mode)
		return nil
	})

	if err := choose.Execute(t.Context(), domainkeyfile.Plaintext, ""); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if file.plaintext != 1 || len(remembered) != 1 || remembered[0] != domainkeyfile.Plaintext {
		t.Fatalf("plaintext used %d times, remembered %v", file.plaintext, remembered)
	}
}

func TestChooseAppliesPassphraseAndRemembersIt(t *testing.T) {
	file := &recordingFile{}
	var remembered []domainkeyfile.Mode
	choose := keyfile.NewChoose(file, func(mode domainkeyfile.Mode) error {
		remembered = append(remembered, mode)
		return nil
	})

	if err := choose.Execute(t.Context(), domainkeyfile.Passphrase, "moon river"); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if len(file.passphrases) != 1 || file.passphrases[0] != "moon river" || len(remembered) != 1 {
		t.Fatalf("passphrases %v, remembered %v", file.passphrases, remembered)
	}
}

func TestChooseRefusesWithoutChangingAnything(t *testing.T) {
	cases := map[string]struct {
		mode       domainkeyfile.Mode
		passphrase string
		want       error
	}{
		"empty passphrase": {domainkeyfile.Passphrase, "", domainkeyfile.ErrEmptyPassphrase},
		"unset":            {domainkeyfile.Unset, "", keyfile.ErrInvalidMode},
		"unknown":          {domainkeyfile.Mode("other"), "", keyfile.ErrInvalidMode},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			file := &recordingFile{}
			choose := keyfile.NewChoose(file, func(domainkeyfile.Mode) error { t.Fatal("remembered"); return nil })

			err := choose.Execute(t.Context(), c.mode, c.passphrase)

			if !errors.Is(err, c.want) || file.plaintext != 0 || len(file.passphrases) != 0 {
				t.Fatalf("err %v, plaintext %d, passphrases %v", err, file.plaintext, file.passphrases)
			}
		})
	}
}

func TestChooseReportsAChoiceThatIsNotSaved(t *testing.T) {
	choose := keyfile.NewChoose(&recordingFile{}, func(domainkeyfile.Mode) error { return errors.New("disk full") })

	err := choose.Execute(t.Context(), domainkeyfile.Plaintext, "")

	if !errors.Is(err, keyfile.ErrNotSaved) {
		t.Fatalf("err %v", err)
	}
}

func TestChooseRefusesAWrongPassphrase(t *testing.T) {
	file := &recordingFile{fail: domainkeyfile.ErrWrongPassphrase}
	choose := keyfile.NewChoose(file, func(domainkeyfile.Mode) error { t.Fatal("remembered"); return nil })

	err := choose.Execute(t.Context(), domainkeyfile.Passphrase, "wrong")

	if !errors.Is(err, domainkeyfile.ErrWrongPassphrase) {
		t.Fatalf("err %v", err)
	}
}

func TestUnlockPassesThePassphraseOn(t *testing.T) {
	file := &recordingFile{}

	if err := keyfile.NewUnlock(file).Execute(t.Context(), "moon river"); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if len(file.unlocked) != 1 || file.unlocked[0] != "moon river" {
		t.Fatalf("unlocked %v", file.unlocked)
	}
}

func TestUnlockRefusesAnEmptyPassphrase(t *testing.T) {
	file := &recordingFile{}

	err := keyfile.NewUnlock(file).Execute(t.Context(), "")

	if !errors.Is(err, domainkeyfile.ErrEmptyPassphrase) || len(file.unlocked) != 0 {
		t.Fatalf("err %v, unlocked %v", err, file.unlocked)
	}
}

func TestGetStateReturnsTheFileState(t *testing.T) {
	want := domainkeyfile.State{IsInUse: true, Mode: domainkeyfile.Passphrase, Warning: "", IsLocked: true}

	got, err := keyfile.NewGetState(&recordingFile{state: want}).Execute(t.Context())

	if err != nil || got != want {
		t.Fatalf("state %+v, err %v", got, err)
	}
}
