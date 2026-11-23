package identity

import (
	"errors"
	"fmt"
)

var (
	ErrWordCount          = errors.New("a recovery phrase has 12 words")
	ErrUnknownWord        = errors.New("not a recovery word")
	ErrChecksum           = errors.New("the recovery words do not match their checksum")
	ErrMalformed          = errors.New("identity: malformed data")
	ErrUnsupportedVersion = errors.New("identity: unsupported version")
	ErrBadSignature       = errors.New("identity: bad signature")
	ErrBrokenLink         = errors.New("identity: broken chain link")
	ErrUnknownSigner      = errors.New("identity: signer is not in the chain")
	ErrDuplicateDevice    = errors.New("identity: device is already in the chain")
	ErrDeviceName         = errors.New("identity: invalid device name")
	ErrUnwrap             = errors.New("identity: account keys cannot be unwrapped")
)

var (
	ErrSheetInvalid    = errors.New("the recovery sheets are not valid")
	ErrNotEnoughSheets = errors.New("two recovery sheets are needed")
)

// SheetError names the sheet that makes a set of sheets invalid by its 1-based position in the
// sheets given. It carries no word.
type SheetError struct {
	Position int
	Err      error
}

func (e *SheetError) Error() string { return fmt.Sprintf("sheet %d: %v", e.Position, e.Err) }

func (e *SheetError) Unwrap() error { return e.Err }

var (
	ErrSelfRevoke     = errors.New("identity: a device cannot revoke itself")
	ErrUnknownDevice  = errors.New("identity: device is not active in the chain")
	ErrWrappedSet     = errors.New("identity: the wrapped keys do not name the active devices")
	ErrTooManyDevices = errors.New("identity: too many active devices")
	ErrChainTooLong   = errors.New("identity: the chain is too long")
	ErrNotExtension   = errors.New("identity: the chain does not extend the chain of the state")
	ErrLinkPayload    = errors.New("identity: not a link text")
	ErrLinkAuth       = errors.New("identity: link message not authentic")
	ErrLinkReplay     = errors.New("identity: link message replayed or out of order")
)
