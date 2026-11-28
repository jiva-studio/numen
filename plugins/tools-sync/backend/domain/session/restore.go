package session

import "errors"

var (
	// ErrCannotOpen means sealed data does not open with the keys of this account.
	ErrCannotOpen = errors.New("the sealed data cannot be opened")
	// ErrNoPendingRestore means no restore waits for an account chain.
	ErrNoPendingRestore = errors.New("no restore is pending on this device")
	// ErrInvalidChain means a chain does not verify or does not belong to the recovery words.
	ErrInvalidChain = errors.New("the account chain is not valid for the recovery words")
)

// PendingRestore is a restore of recovery words that waits for the account chain of the server.
type PendingRestore struct {
	IsPending     bool
	RootSignKey   []byte
	DeviceSignKey []byte
}
