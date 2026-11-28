// Package keyfile holds the words of the file a computer without a system keychain keeps its keys in.
package keyfile

import "errors"

// ErrWrongPassphrase is the passphrase that does not open the key file.
var ErrWrongPassphrase = errors.New("the passphrase does not open the key file")

// ErrEmptyPassphrase is a passphrase of no characters.
var ErrEmptyPassphrase = errors.New("the passphrase is empty")

// MinPassphraseLength is the fewest characters a new passphrase of the key file has.
const MinPassphraseLength = 10

// ErrShortPassphrase is a new passphrase of fewer than MinPassphraseLength characters.
var ErrShortPassphrase = errors.New("the passphrase is too short")

// Mode is how the key file keeps the keys.
type Mode string

const (
	// Unset is a file the person has not chosen a mode for.
	Unset Mode = "unset"
	// Plaintext keeps the keys as readable text.
	Plaintext Mode = "plaintext"
	// Passphrase keeps the keys encrypted with a passphrase.
	Passphrase Mode = "passphrase"
)

// State is how the key file is kept now.
type State struct {
	IsInUse  bool
	Mode     Mode
	Warning  string
	IsLocked bool
}
