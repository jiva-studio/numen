package pluginport

import (
	"context"
	"errors"
)

// ErrWrongPassphrase is the passphrase that does not open the key file.
var ErrWrongPassphrase = errors.New("the passphrase does not open the key file")

// ErrEmptyPassphrase is a passphrase of no characters.
var ErrEmptyPassphrase = errors.New("the passphrase is empty")

// KeyFileMode is how the key file keeps the keys.
type KeyFileMode string

const (
	// KeyFileUnset is a file the person has not chosen a mode for.
	KeyFileUnset KeyFileMode = "unset"
	// KeyFilePlaintext keeps the keys as readable text.
	KeyFilePlaintext KeyFileMode = "plaintext"
	// KeyFilePassphrase keeps the keys encrypted with a passphrase.
	KeyFilePassphrase KeyFileMode = "passphrase"
)

// KeyFileState is how the key file is kept now.
type KeyFileState struct {
	IsInUse  bool
	Mode     KeyFileMode
	Warning  string
	IsLocked bool
}

// KeyFile is the file a computer without a system keychain keeps its keys in.
type KeyFile interface {
	// GetState returns how the file is kept; IsInUse is false where the system keychain works.
	GetState(ctx context.Context) (KeyFileState, error)
	// UsePlaintext keeps the keys of the file as plaintext from now on. It returns what keeps the file
	// from being written again, such as a passphrase file that is still locked.
	UsePlaintext(ctx context.Context) error
	// UsePassphrase encrypts the keys of the file with the passphrase from now on. It returns
	// ErrWrongPassphrase for a passphrase that does not open a file that is already encrypted.
	UsePassphrase(ctx context.Context, passphrase string) error
	// Unlock gives the passphrase of the encrypted file for this run. It returns ErrWrongPassphrase for a
	// passphrase that does not open the file.
	Unlock(ctx context.Context, passphrase string) error
}
