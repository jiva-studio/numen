package api

// ShareState is how a folder stands to sharing.
type ShareState string

const (
	// ShareNotShared is a folder with no share.
	ShareNotShared ShareState = "notShared"
	// ShareCreating is a share whose files are not all rewrapped yet.
	ShareCreating ShareState = "creating"
	// ShareActive is a share that takes readers.
	ShareActive ShareState = "active"
)

// ReaderState is whether a reader reads the folder.
type ReaderState string

const (
	// ReaderActive is a reader of the folder.
	ReaderActive  ReaderState = "active"
	ReaderRemoved ReaderState = "removed"
)

// GetShareRequest names a folder of a vault.
type GetShareRequest struct {
	VaultID string `json:"vaultId"`
	Folder  string `json:"folder"`
}

type ShareReader struct {
	Label       string      `json:"label"`
	Fingerprint string      `json:"fingerprint"`
	State       ReaderState `json:"state"`
}

// GetShareResponse is the sharing of a folder: its state and its readers, the removed ones too. Error is why a share
// accepts no readers, empty when nothing is wrong.
type GetShareResponse struct {
	State   ShareState    `json:"state"`
	Readers []ShareReader `json:"readers"`
	Error   string        `json:"error,omitempty"`
}

// ListSharesRequest asks for the folders this account shares.
type ListSharesRequest struct{}

// SharedFolder is a folder of a vault of this account that is shared, with its readers, the removed ones too.
type SharedFolder struct {
	VaultID string        `json:"vaultId"`
	Folder  string        `json:"folder"`
	State   ShareState    `json:"state"`
	Readers []ShareReader `json:"readers"`
}

// ListSharesResponse is the folders this account shares, in the order of the vault list. A folder whose share was
// stopped is not listed.
type ListSharesResponse struct {
	Shares []SharedFolder `json:"shares"`
}

// RemoveReaderRequest removes the reader with the account fingerprint from a folder.
type RemoveReaderRequest struct {
	VaultID     string `json:"vaultId"`
	Folder      string `json:"folder"`
	Fingerprint string `json:"fingerprint"`
}

// RemoveReaderResponse is the answer to a removal that went through.
type RemoveReaderResponse struct{}

// StopShareRequest stops sharing a folder.
type StopShareRequest struct {
	VaultID string `json:"vaultId"`
	Folder  string `json:"folder"`
}

type StopShareResponse struct{}

// ReceivedState is how a received share stands.
type ReceivedState string

const (
	// ReceivedReceiving is a share that was not synced yet.
	ReceivedReceiving ReceivedState = "receiving"
	// ReceivedReady is a share synced at least once.
	ReceivedReady ReceivedState = "ready"
	// ReceivedEnded is a share the owner ended.
	ReceivedEnded ReceivedState = "ended"
)

// ListReceivedRequest asks for the shares this computer received.
type ListReceivedRequest struct{}

type ReceivedShare struct {
	VaultID          string        `json:"vaultId"`
	Name             string        `json:"name"`
	OwnerFingerprint string        `json:"ownerFingerprint"`
	State            ReceivedState `json:"state"`
	LastSyncedAt     string        `json:"lastSyncedAt"`
}

// ListReceivedResponse is the shares this computer received.
type ListReceivedResponse struct {
	Shares []ReceivedShare `json:"shares"`
}

// LeaveShareRequest asks to leave the received share held by a vault. The files stay on this device.
type LeaveShareRequest struct {
	VaultID string `json:"vaultId"`
}

// LeaveShareResponse is the answer to a share that was left.
type LeaveShareResponse struct{}
