// Package api is the JSON messages between the settings view of Numen Sync and its backend.
package api

// Service is the name of the service of the plugin.
const Service = "md.numen.tools.sync"

// The methods of the service.
const (
	MethodListVaults       = "ListVaults"
	MethodListRemoteVaults = "ListRemoteVaults"
	MethodConnectVault     = "ConnectVault"
	MethodGetStatus        = "GetStatus"
	MethodGetSubscription  = "GetSubscription"
	MethodCreateCheckout   = "CreateCheckout"
	MethodGetReaderCode    = "GetReaderCode"
	MethodReadReaderCode   = "ReadReaderCode"
	MethodGetShare         = "GetShare"
	MethodAddReader        = "AddReader"
	MethodRemoveReader     = "RemoveReader"
	MethodStopShare        = "StopShare"
	MethodReadInvitation   = "ReadInvitation"
	MethodAcceptInvitation = "AcceptInvitation"
	MethodListReceived     = "ListReceived"
)

// State is how a vault of this device stands to the synced vaults.
type State string

const (
	// StateNotConnected is a vault that is not tied to a synced vault.
	StateNotConnected State = "notConnected"
	// StateConnected is a vault tied to a synced vault.
	StateConnected State = "connected"
	// StateCopy is a vault whose folder is a copy of a connected vault.
	StateCopy State = "copy"
	// StateReceived is a vault that holds a folder another person shares.
	StateReceived State = "received"
)

// Mode is how a vault is connected.
type Mode string

const (
	// ModeNew makes a synced vault of the vault.
	ModeNew Mode = "new"
	// ModeExisting ties the vault to a synced vault of the account.
	ModeExisting Mode = "existing"
)

// Vault is a vault of this installation.
type Vault struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Path          string `json:"path"`
	State         State  `json:"state"`
	LastSyncedAt  string `json:"lastSyncedAt"`
	ConflictCount int    `json:"conflictCount"`
}

// ListVaultsRequest asks for every vault of the installation.
type ListVaultsRequest struct{}

type ListVaultsResponse struct {
	Vaults []Vault `json:"vaults"`
}

// RemoteVault is a synced vault of the account.
type RemoteVault struct {
	RemoteID    string `json:"remoteId"`
	Name        string `json:"name"`
	IsConnected bool   `json:"isConnected"`
}

// ListRemoteVaultsRequest asks for the synced vaults of the account.
type ListRemoteVaultsRequest struct{}

// ListRemoteVaultsResponse is the synced vaults of the account.
type ListRemoteVaultsResponse struct {
	Vaults []RemoteVault `json:"vaults"`
}

// ConnectVaultRequest connects a vault. RemoteID names the synced vault for ModeExisting.
type ConnectVaultRequest struct {
	VaultID  string `json:"vaultId"`
	Mode     Mode   `json:"mode"`
	RemoteID string `json:"remoteId,omitempty"`
}

// ConnectVaultResponse is the answer to a connection that went through.
type ConnectVaultResponse struct{}

// GetStatusRequest asks how sync stands.
type GetStatusRequest struct{}

// StatusItem is a warning of sync.
type StatusItem struct {
	Key  string `json:"key"`
	Text string `json:"text"`
}

// GetStatusResponse holds whether this device has an identity, whether the server answers and what warns.
type GetStatusResponse struct {
	HasIdentity bool         `json:"hasIdentity"`
	IsReachable bool         `json:"isReachable"`
	Items       []StatusItem `json:"items"`
}

// Phase is where the subscription of the account stands, as the sync server tells it.
type Phase string

const (
	// PhaseNone is an account that never had a subscription.
	PhaseNone Phase = "none"
	// PhaseActive is a paid period that runs.
	PhaseActive Phase = "active"
	// PhaseGrace is a period that ended and whose changes are still sent.
	PhaseGrace Phase = "grace"
	// PhaseReadOnly is an account whose vaults in the cloud can only be read.
	PhaseReadOnly Phase = "readOnly"
	// PhaseDeleted is an account whose vaults the cloud deleted.
	PhaseDeleted Phase = "deleted"
)

// GetSubscriptionRequest asks for the subscription of the account.
type GetSubscriptionRequest struct{}

// GetSubscriptionResponse is the subscription of the account. HasIdentity is false when this device has none;
// then the server is not asked and Phase is empty. A time is RFC 3339, and empty when the server sent none.
type GetSubscriptionResponse struct {
	HasIdentity bool   `json:"hasIdentity"`
	Phase       Phase  `json:"phase"`
	PeriodEndAt string `json:"periodEndAt"`
	WritesEndAt string `json:"writesEndAt"`
	DeleteAt    string `json:"deleteAt"`
}

// CreateCheckoutRequest asks for a payment page.
type CreateCheckoutRequest struct{}

// CreateCheckoutResponse is the address of the payment page.
type CreateCheckoutResponse struct {
	CheckoutURL string `json:"checkoutUrl"`
}

// GetReaderCodeRequest asks for the code this account gives the owner of a share.
type GetReaderCodeRequest struct{}

// GetReaderCodeResponse is the reader code and the fingerprint of this account, which the two people compare.
type GetReaderCodeResponse struct {
	Code        string `json:"code"`
	Fingerprint string `json:"fingerprint"`
}

// ReadReaderCodeRequest holds a reader code a person pasted.
type ReadReaderCodeRequest struct {
	Code string `json:"code"`
}

// ReadReaderCodeResponse is the account fingerprint of the reader the code names. Label is the name the owner gave
// that reader before, empty for a person not shared with.
type ReadReaderCodeResponse struct {
	Fingerprint string `json:"fingerprint"`
	Label       string `json:"label,omitempty"`
}

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
	ReaderActive ReaderState = "active"
	// ReaderRemoved is a reader the owner removed.
	ReaderRemoved ReaderState = "removed"
)

// GetShareRequest names a folder of a vault.
type GetShareRequest struct {
	VaultID string `json:"vaultId"`
	Folder  string `json:"folder"`
}

// ShareReader is a reader of a shared folder.
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

// AddReaderRequest shares the folder with the person of the code, and makes the share first when the folder has none.
// ConfirmChange says the owner accepts a label that another person had before.
type AddReaderRequest struct {
	VaultID       string `json:"vaultId"`
	Folder        string `json:"folder"`
	Code          string `json:"code"`
	Label         string `json:"label"`
	ConfirmChange bool   `json:"confirmChange,omitempty"`
}

// AddReaderResponse is the invitation to send the reader and the account fingerprint of the reader.
type AddReaderResponse struct {
	Invitation  string `json:"invitation"`
	Fingerprint string `json:"fingerprint"`
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

// StopShareResponse is the answer to a share that stopped.
type StopShareResponse struct{}

// ReadInvitationRequest holds an invitation a person pasted.
type ReadInvitationRequest struct {
	Text string `json:"text"`
}

// ReadInvitationResponse is what an invitation offers: the name of the vault it makes and the fingerprint of the owner.
type ReadInvitationResponse struct {
	ShareName        string `json:"shareName"`
	OwnerFingerprint string `json:"ownerFingerprint"`
	IsForThisAccount bool   `json:"isForThisAccount"`
	AlreadyReceived  bool   `json:"alreadyReceived"`
}

// AcceptInvitationRequest holds the invitation to receive.
type AcceptInvitationRequest struct {
	Text string `json:"text"`
}

// AcceptInvitationResponse is the answer to an invitation that was received; the vault is in ListReceived.
type AcceptInvitationResponse struct{}

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

// ReceivedShare is a share this computer received.
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
