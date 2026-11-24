package keyfile_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/keyfile"
)

type recordingFile struct {
	state       pluginport.KeyFileState
	plaintext   int
	passphrases []string
	unlocked    []string
	fail        error
}

func (f *recordingFile) GetState(context.Context) (pluginport.KeyFileState, error) {
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
	var remembered []pluginport.KeyFileMode
	choose := keyfile.NewChoose(file, func(mode pluginport.KeyFileMode) error {
		remembered = append(remembered, mode)
		return nil
	})

	if err := choose.Execute(t.Context(), pluginport.KeyFilePlaintext, ""); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if file.plaintext != 1 || len(remembered) != 1 || remembered[0] != pluginport.KeyFilePlaintext {
		t.Fatalf("plaintext used %d times, remembered %v", file.plaintext, remembered)
	}
}

func TestChooseAppliesPassphraseAndRemembersIt(t *testing.T) {
	file := &recordingFile{}
	var remembered []pluginport.KeyFileMode
	choose := keyfile.NewChoose(file, func(mode pluginport.KeyFileMode) error {
		remembered = append(remembered, mode)
		return nil
	})

	if err := choose.Execute(t.Context(), pluginport.KeyFilePassphrase, "moon river"); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if len(file.passphrases) != 1 || file.passphrases[0] != "moon river" || len(remembered) != 1 {
		t.Fatalf("passphrases %v, remembered %v", file.passphrases, remembered)
	}
}

func TestChooseRefusesWithoutChangingAnything(t *testing.T) {
	cases := map[string]struct {
		mode       pluginport.KeyFileMode
		passphrase string
		want       error
	}{
		"empty passphrase": {pluginport.KeyFilePassphrase, "", pluginport.ErrEmptyPassphrase},
		"unset":            {pluginport.KeyFileUnset, "", keyfile.ErrInvalidMode},
		"unknown":          {pluginport.KeyFileMode("other"), "", keyfile.ErrInvalidMode},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			file := &recordingFile{}
			choose := keyfile.NewChoose(file, func(pluginport.KeyFileMode) error { t.Fatal("remembered"); return nil })

			err := choose.Execute(t.Context(), c.mode, c.passphrase)

			if !errors.Is(err, c.want) || file.plaintext != 0 || len(file.passphrases) != 0 {
				t.Fatalf("err %v, plaintext %d, passphrases %v", err, file.plaintext, file.passphrases)
			}
		})
	}
}

func TestChooseReportsAChoiceThatIsNotSaved(t *testing.T) {
	choose := keyfile.NewChoose(&recordingFile{}, func(pluginport.KeyFileMode) error { return errors.New("disk full") })

	err := choose.Execute(t.Context(), pluginport.KeyFilePlaintext, "")

	if !errors.Is(err, keyfile.ErrNotSaved) {
		t.Fatalf("err %v", err)
	}
}

func TestChooseKeepsTheWrongPassphraseRefusal(t *testing.T) {
	file := &recordingFile{fail: pluginport.ErrWrongPassphrase}
	choose := keyfile.NewChoose(file, func(pluginport.KeyFileMode) error { t.Fatal("remembered"); return nil })

	err := choose.Execute(t.Context(), pluginport.KeyFilePassphrase, "wrong")

	if !errors.Is(err, pluginport.ErrWrongPassphrase) {
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

	if !errors.Is(err, pluginport.ErrEmptyPassphrase) || len(file.unlocked) != 0 {
		t.Fatalf("err %v, unlocked %v", err, file.unlocked)
	}
}

func TestGetStateReturnsTheFileState(t *testing.T) {
	want := pluginport.KeyFileState{IsInUse: true, Mode: pluginport.KeyFilePassphrase, Warning: "", IsLocked: true}

	got, err := keyfile.NewGetState(&recordingFile{state: want}).Execute(t.Context())

	if err != nil || got != want {
		t.Fatalf("state %+v, err %v", got, err)
	}
}
