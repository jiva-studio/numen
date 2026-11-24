package keychain

import (
	"context"
	"errors"
	"fmt"
	"os"
)

// ErrFileBackendNotConfirmed is returned when a key would go to the credentials file and the app has chosen neither a passphrase nor a plaintext file.
var ErrFileBackendNotConfirmed = errors.New("this machine has no keychain: choose a passphrase or a plaintext credentials file first")

var errEmptyPassphrase = errors.New("the passphrase is empty")

// PassphraseSource asks the user for the passphrase of the credentials file.
type PassphraseSource interface {
	GetPassphrase(ctx context.Context) (string, error)
}

// FileMode is how the credentials file is kept where the machine has no keychain.
type FileMode int

const (
	FileModeUnset FileMode = iota
	FileModePlaintext
	FileModePassphrase
)

// FileState tells the window whether the credentials file is in use and how it is kept.
// Warning is empty unless the file is in use and holds its keys as plaintext or no mode is chosen yet.
// IsLocked is a passphrase file whose passphrase this run has not been given yet.
type FileState struct {
	IsInUse  bool
	Mode     FileMode
	Warning  string
	IsLocked bool
}

const (
	warningUnset     = "This machine has no system keychain. Numen will not store keys until you choose a passphrase or a plaintext file."
	warningPlaintext = "This machine has no system keychain. Keys are stored as plaintext in the credentials file; anyone who can read your files can read them."
)

// UsePlaintextFile records the explicit choice of a plaintext credentials file.
func (s *Store) UsePlaintextFile() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fileMode, s.source, s.passphrase = FileModePlaintext, nil, ""
}

// UsePassphraseFile records the choice of a credentials file encrypted with a passphrase that the source supplies.
func (s *Store) UsePassphraseFile(source PassphraseSource) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fileMode, s.source, s.passphrase = FileModePassphrase, source, ""
}

// GetFileState returns how this store keeps keys where the keychain cannot be reached.
func (s *Store) GetFileState(ctx context.Context) (FileState, error) {
	isUp, err := s.isKeychainReachable(ctx)
	if err != nil || isUp {
		return FileState{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	state := FileState{IsInUse: true, Mode: s.fileMode, Warning: "", IsLocked: false}
	switch s.fileMode {
	case FileModeUnset:
		state.Warning = warningUnset
	case FileModePlaintext:
		state.Warning = warningPlaintext
	case FileModePassphrase:
		state.IsLocked = s.passphrase == ""
	}
	return state, nil
}

// ChangeFileMode reads the keys the file holds with the mode in force, calls apply to change the mode, and
// writes the keys again in the new mode. A file that cannot be read keeps its mode.
func (s *Store) ChangeFileMode(ctx context.Context, apply func()) error {
	return s.changeEntries(ctx, func() error {
		entries, err := s.readEntries(ctx)
		if err != nil {
			return err
		}
		apply()
		if len(entries) == 0 {
			return nil
		}
		return s.writeEntries(ctx, entries, s.getFileMode())
	})
}

// CheckFile reads the credentials file with the passphrase the source gives, so that a wrong passphrase
// fails here and a right one is kept for the keys that follow.
func (s *Store) CheckFile(ctx context.Context) error {
	_, err := s.readEntries(ctx)
	return err
}

func (s *Store) getFileMode() FileMode {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.fileMode
}

func (s *Store) warnPlaintext() {
	s.mu.Lock()
	hasWarned := s.hasWarned
	s.hasWarned = true
	warn := s.warn
	s.mu.Unlock()
	if hasWarned {
		return
	}
	if warn == nil {
		warn = func(text string) { fmt.Fprintln(os.Stderr, "numen: "+text) }
	}
	warn(warningPlaintext)
}

func (s *Store) getPassphrase(ctx context.Context) (string, error) {
	s.mu.Lock()
	passphrase, source := s.passphrase, s.source
	s.mu.Unlock()
	if passphrase != "" {
		return passphrase, nil
	}
	if source == nil {
		return "", ErrFileBackendNotConfirmed
	}
	passphrase, err := source.GetPassphrase(ctx)
	if err != nil {
		return "", err
	}
	if passphrase == "" {
		return "", errEmptyPassphrase
	}
	s.mu.Lock()
	s.passphrase = passphrase
	s.mu.Unlock()
	return passphrase, nil
}

func (s *Store) forgetPassphrase() {
	s.mu.Lock()
	s.passphrase = ""
	s.mu.Unlock()
}

func (s *Store) getKDFParams() kdfParams {
	if s.kdf == (kdfParams{}) {
		return getDefaultKDFParams()
	}
	return s.kdf
}
