package keyfile_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/keyfile"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	usecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/keyfile"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

type fakeFile struct {
	state       pluginport.KeyFileState
	err         error
	plaintext   int
	passphrases []string
	unlocked    []string
}

func (f *fakeFile) GetState(context.Context) (pluginport.KeyFileState, error) { return f.state, nil }

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

func newHandler(file *fakeFile, remembered *[]pluginport.KeyFileMode) *keyfile.Handler {
	return keyfile.NewHandler(keyfile.UseCases{
		GetState: usecase.NewGetState(file),
		Choose: usecase.NewChoose(file, func(mode pluginport.KeyFileMode) error {
			*remembered = append(*remembered, mode)
			return nil
		}),
		Unlock: usecase.NewUnlock(file),
	})
}

func TestGetKeyFileStateAsksForAChoiceWhileNoneIsMade(t *testing.T) {
	file := &fakeFile{state: pluginport.KeyFileState{IsInUse: true, Mode: pluginport.KeyFileUnset, Warning: "no keychain", IsLocked: false}}

	got, err := newHandler(file, new([]pluginport.KeyFileMode)).GetKeyFileState(t.Context(), connect.NewRequest(&v1.GetKeyFileStateRequest{}))

	if err != nil || !got.Msg.GetIsChoiceNeeded() || !got.Msg.GetIsInUse() || got.Msg.GetWarning() != "no keychain" {
		t.Fatalf("answer %v, err %v", got.Msg, err)
	}
}

func TestGetKeyFileStateAsksForNothingWhereTheKeychainWorks(t *testing.T) {
	got, err := newHandler(&fakeFile{}, new([]pluginport.KeyFileMode)).GetKeyFileState(t.Context(), connect.NewRequest(&v1.GetKeyFileStateRequest{}))

	if err != nil || got.Msg.GetIsChoiceNeeded() || got.Msg.GetIsInUse() || got.Msg.GetIsLocked() {
		t.Fatalf("answer %v, err %v", got.Msg, err)
	}
}

func TestGetKeyFileStateReportsALockedFile(t *testing.T) {
	file := &fakeFile{state: pluginport.KeyFileState{IsInUse: true, Mode: pluginport.KeyFilePassphrase, Warning: "", IsLocked: true}}

	got, _ := newHandler(file, new([]pluginport.KeyFileMode)).GetKeyFileState(t.Context(), connect.NewRequest(&v1.GetKeyFileStateRequest{}))

	if !got.Msg.GetIsLocked() || got.Msg.GetIsChoiceNeeded() || got.Msg.GetMode() != v1.KeyFileMode_KEY_FILE_MODE_PASSPHRASE {
		t.Fatalf("answer %v", got.Msg)
	}
}

func TestChooseKeyFileKeepsAPassphraseChoice(t *testing.T) {
	file := &fakeFile{}
	var remembered []pluginport.KeyFileMode

	got, err := newHandler(file, &remembered).ChooseKeyFile(t.Context(), connect.NewRequest(&v1.ChooseKeyFileRequest{
		Mode: v1.KeyFileMode_KEY_FILE_MODE_PASSPHRASE, Passphrase: "moon river",
	}))

	if err != nil || got.Msg.GetError() != v1.KeyFileErrorCode_KEY_FILE_ERROR_CODE_UNSPECIFIED ||
		len(file.passphrases) != 1 || len(remembered) != 1 {
		t.Fatalf("answer %v, err %v, passphrases %v", got.Msg, err, file.passphrases)
	}
}

func TestChooseKeyFileRefusesAnUnsetMode(t *testing.T) {
	got, err := newHandler(&fakeFile{}, new([]pluginport.KeyFileMode)).ChooseKeyFile(t.Context(), connect.NewRequest(&v1.ChooseKeyFileRequest{
		Mode: v1.KeyFileMode_KEY_FILE_MODE_UNSET,
	}))

	if err != nil || got.Msg.GetError() != v1.KeyFileErrorCode_KEY_FILE_ERROR_CODE_INVALID_MODE {
		t.Fatalf("answer %v, err %v", got.Msg, err)
	}
}

func TestUnlockKeyFileAnswersAWrongPassphraseAsACode(t *testing.T) {
	got, err := newHandler(&fakeFile{err: pluginport.ErrWrongPassphrase}, new([]pluginport.KeyFileMode)).UnlockKeyFile(
		t.Context(), connect.NewRequest(&v1.UnlockKeyFileRequest{Passphrase: "wrong"}))

	if err != nil || got.Msg.GetError() != v1.KeyFileErrorCode_KEY_FILE_ERROR_CODE_WRONG_PASSPHRASE {
		t.Fatalf("answer %v, err %v", got.Msg, err)
	}
}

func TestKeyFileErrorCodesAreProducedFromFailures(t *testing.T) {
	file := &fakeFile{}
	handler := newHandler(file, new([]pluginport.KeyFileMode))
	testsupport.CheckProduced(t, map[v1.KeyFileErrorCode]error{
		v1.KeyFileErrorCode_KEY_FILE_ERROR_CODE_UNSPECIFIED:      nil,
		v1.KeyFileErrorCode_KEY_FILE_ERROR_CODE_WRONG_PASSPHRASE: pluginport.ErrWrongPassphrase,
		v1.KeyFileErrorCode_KEY_FILE_ERROR_CODE_EMPTY_PASSPHRASE: pluginport.ErrEmptyPassphrase,
		v1.KeyFileErrorCode_KEY_FILE_ERROR_CODE_SAVE_FAILED:      usecase.ErrNotSaved,
		v1.KeyFileErrorCode_KEY_FILE_ERROR_CODE_INVALID_MODE:     usecase.ErrInvalidMode,
	}, func(failure error) v1.KeyFileErrorCode {
		file.err = failure
		got, _ := handler.UnlockKeyFile(context.Background(), connect.NewRequest(&v1.UnlockKeyFileRequest{Passphrase: "x"}))
		return got.Msg.GetError()
	})
}
