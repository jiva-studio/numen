package share

import "errors"

var (
	// ErrNotConnected means the vault is not connected to a synced vault.
	ErrNotConnected = errors.New("the vault is not connected to Numen Sync")
	// ErrReceivedVault means the vault is a shared vault, which is not shared again or synced as an own vault.
	ErrReceivedVault = errors.New("a shared vault cannot be shared again")
	// ErrReceivedNotSynced means a shared vault is connected as a synced vault.
	ErrReceivedNotSynced = errors.New("a shared vault cannot be synced again")
	// ErrOwnIdentity means the reader code is the code of the owner.
	ErrOwnIdentity = errors.New("that is your own identity")
	// ErrAlreadyReader means the person is a reader of the folder already.
	ErrAlreadyReader = errors.New("this person already reads the folder")
	// ErrChangedIdentity means a label is used by a reader with another root and the caller has not confirmed.
	ErrChangedIdentity = errors.New("this person has another identity than before")
	// ErrReaderNotAdded means the server refused to add the reader and gave no reason: the root may have no account.
	ErrReaderNotAdded = errors.New("the reader could not be added; the reader may not have signed in to Numen Sync yet")
	// ErrNotForThisAccount means an invitation names the fingerprint of another reader.
	ErrNotForThisAccount = errors.New("this invitation is for another person")
	// ErrAlreadyReceived means this device holds the share already.
	ErrAlreadyReceived = errors.New("this share is received already")
	// ErrOwnerChanged means an invitation for a known share carries another owner root than the pinned one.
	ErrOwnerChanged = errors.New("the owner identity changed")
	// ErrEnded means the owner removed the reader or stopped the share.
	ErrEnded = errors.New("the owner ended this share")
	// ErrDamagedLog means the share log fails the checks of a reader.
	ErrDamagedLog = errors.New("the share log is damaged")
	// ErrRolledBack means the server served an older state of the share than this reader has seen: a log that lacks its
	// last entry, a log with another entry in its place, a file back at an older revision, or a revision of an epoch the
	// log has not reached.
	ErrRolledBack = errors.New("the share went back to an older state")
	// ErrNotReady means the owner has not finished making the share, so no reader is added yet.
	ErrNotReady = errors.New("the share is not ready for readers")
)

var (
	// ErrUnknownReader means the folder has no reader with the fingerprint.
	ErrUnknownReader = errors.New("this person does not read the folder")
	// ErrUnknownShare means the server has no such share for this account.
	ErrUnknownShare = errors.New("this share does not exist or is not meant for you")
	// ErrLogBusy means other devices of the owner changed the share log first, every time.
	ErrLogBusy = errors.New("the share log changed on another device again and again")
	// ErrNotShared means the folder has no share to change.
	ErrNotShared = errors.New("the folder is not shared")
)
