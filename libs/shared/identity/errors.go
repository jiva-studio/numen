package identity

import "errors"

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
