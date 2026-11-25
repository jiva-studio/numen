package keyfile_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/keyfile"
	domainkeyfile "github.com/jiva-studio/numen/libs/core/kernel/domain/keyfile"
	keyfileusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/keyfile"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

type fakeFile struct {
	state       domainkeyfile.State
	err         error
	plaintext   int
	passphrases []string
	unlocked    []string
}

func (f *fakeFile) GetState(context.Context) (domainkeyfile.State, error) { return f.state, nil }

func (f *fakeFile) UsePlaintext(context.Context) error {
	f.plaintext++
	return f.err
}

func (f *fakeFile) UsePassphrase(_ context.Context, passphrase string) error {
	f.passphrases = append(f.passphrases, passphrase)
	return f.err
}
func (f *fakeFile) Unlock(_ context.Context, passphrase string) error {
	f.unlocked = append(f.unlocked, passphrase)
	return f.err
}

func newHandler(file *fakeFile, remembered *[]domainkeyfile.Mode) *keyfile.Handler {
	return keyfile.NewHandler(keyfile.UseCases{
		GetState: keyfileusecase.NewGetState(file),
		Choose: keyfileusecase.NewChoose(file, func(mode domainkeyfile.Mode) error {
			*remembered = append(*remembered, mode)
			return nil
		}),
		Unlock: keyfileusecase.NewUnlock(file),
	})
}

func TestGetKeyFileStateAsksForAChoiceWhileNoneIsMade(t *testing.T) {
	file := &fakeFile{state: domainkeyfile.State{IsInUse: true, Mode: domainkeyfile.Unset, Warning: "no keychain", IsLocked: false}}

	got, err := newHandler(file, new([]domainkeyfile.Mode)).GetKeyFileState(t.Context(), connect.NewRequest(&v1.GetKeyFileStateRequest{}))

	if err != nil || !got.Msg.GetIsChoiceNeeded() || !got.Msg.GetIsInUse() || got.Msg.GetWarning() != "no keychain" {
		t.Fatalf("answer %v, err %v", got.Msg, err)
	}
}

func TestGetKeyFileStateAsksForNothingWhereTheKeychainWorks(t *testing.T) {
	got, err := newHandler(&fakeFile{}, new([]domainkeyfile.Mode)).GetKeyFileState(t.Context(), connect.NewRequest(&v1.GetKeyFileStateRequest{}))

	if err != nil || got.Msg.GetIsChoiceNeeded() || got.Msg.GetIsInUse() || got.Msg.GetIsLocked() {
		t.Fatalf("answer %v, err %v", got.Msg, err)
	}
}

func TestGetKeyFileStateReportsALockedFile(t *testing.T) {
	file := &fakeFile{state: domainkeyfile.State{IsInUse: true, Mode: domainkeyfile.Passphrase, Warning: "", IsLocked: true}}

	got, err := newHandler(file, new([]domainkeyfile.Mode)).GetKeyFileState(t.Context(), connect.NewRequest(&v1.GetKeyFileStateRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	if !got.Msg.GetIsLocked() || got.Msg.GetIsChoiceNeeded() || got.Msg.GetMode() != v1.KeyFileMode_KEY_FILE_MODE_PASSPHRASE {
		t.Fatalf("answer %v", got.Msg)
	}
}

func TestChooseKeyFileKeepsAPassphraseChoice(t *testing.T) {
	file := &fakeFile{}
	var remembered []domainkeyfile.Mode

	got, err := newHandler(file, &remembered).ChooseKeyFile(t.Context(), connect.NewRequest(&v1.ChooseKeyFileRequest{
		Mode: v1.KeyFileMode_KEY_FILE_MODE_PASSPHRASE, Passphrase: "moon river",
	}))

	if err != nil || got.Msg.GetError() != v1.ErrorCode_ERROR_CODE_UNSPECIFIED ||
		len(file.passphrases) != 1 || len(remembered) != 1 {
		t.Fatalf("answer %v, err %v, passphrases %v", got.Msg, err, file.passphrases)
	}
}

func TestChooseKeyFileRefusesAnUnsetMode(t *testing.T) {
	got, err := newHandler(&fakeFile{}, new([]domainkeyfile.Mode)).ChooseKeyFile(t.Context(), connect.NewRequest(&v1.ChooseKeyFileRequest{
		Mode: v1.KeyFileMode_KEY_FILE_MODE_UNSET,
	}))

	if err != nil || got.Msg.GetError() != v1.ErrorCode_ERROR_CODE_INVALID {
		t.Fatalf("answer %v, err %v", got.Msg, err)
	}
}

func TestUnlockKeyFileAnswersAWrongPassphraseAsACode(t *testing.T) {
	got, err := newHandler(&fakeFile{err: domainkeyfile.ErrWrongPassphrase}, new([]domainkeyfile.Mode)).UnlockKeyFile(
		t.Context(), connect.NewRequest(&v1.UnlockKeyFileRequest{Passphrase: "wrong"}))

	if err != nil || got.Msg.GetError() != v1.ErrorCode_ERROR_CODE_WRONG_PASSPHRASE {
		t.Fatalf("answer %v, err %v", got.Msg, err)
	}
}

func TestGetErrorCodeOfKeyFileFailures(t *testing.T) {
	file := &fakeFile{}
	handler := newHandler(file, new([]domainkeyfile.Mode))
	for want, failure := range map[v1.ErrorCode]error{
		v1.ErrorCode_ERROR_CODE_UNSPECIFIED:      nil,
		v1.ErrorCode_ERROR_CODE_WRONG_PASSPHRASE: domainkeyfile.ErrWrongPassphrase,
		v1.ErrorCode_ERROR_CODE_EMPTY_PASSPHRASE: domainkeyfile.ErrEmptyPassphrase,
		v1.ErrorCode_ERROR_CODE_INTERNAL:         keyfileusecase.ErrNotSaved,
		v1.ErrorCode_ERROR_CODE_INVALID:          keyfileusecase.ErrInvalidMode,
	} {
		file.err = failure
		got, err := handler.UnlockKeyFile(t.Context(), connect.NewRequest(&v1.UnlockKeyFileRequest{Passphrase: "x"}))
		if err != nil {
			t.Fatal(err)
		}
		if got.Msg.GetError() != want {
			t.Errorf("%v is written as %v, want %v", failure, got.Msg.GetError(), want)
		}
	}
}
