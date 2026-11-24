// Package api is the JSON messages between the settings view of Numen Sync and its backend.
package api

// Service is the name of the service of the plugin.
const Service = "md.numen.tools.sync"

// The methods of the service.
const (
	MethodListVaults               = "ListVaults"
	MethodListRemoteVaults         = "ListRemoteVaults"
	MethodConnectVault             = "ConnectVault"
	MethodPreviewConnect           = "PreviewConnect"
	MethodGetStatus                = "GetStatus"
	MethodGetSubscription          = "GetSubscription"
	MethodCreateCheckout           = "CreateCheckout"
	MethodListPlans                = "ListPlans"
	MethodChangePlan               = "ChangePlan"
	MethodCreatePortalSession      = "CreatePortalSession"
	MethodGetAccountState          = "GetAccountState"
	MethodDeleteCloudData          = "DeleteCloudData"
	MethodRestoreAccount           = "RestoreAccount"
	MethodExportAccountData        = "ExportAccountData"
	MethodGetShare                 = "GetShare"
	MethodListShares               = "ListShares"
	MethodRemoveReader             = "RemoveReader"
	MethodStopShare                = "StopShare"
	MethodListReceived             = "ListReceived"
	MethodLeaveShare               = "LeaveShare"
	MethodStopSyncingVault         = "StopSyncingVault"
	MethodStartShareInvitation     = "StartShareInvitation"
	MethodListShareInvitations     = "ListShareInvitations"
	MethodRevokeShareInvitation    = "RevokeShareInvitation"
	MethodListShareRequests        = "ListShareRequests"
	MethodApproveShareRequest      = "ApproveShareRequest"
	MethodDeclineShareRequest      = "DeclineShareRequest"
	MethodPreviewShareInvitation   = "PreviewShareInvitation"
	MethodAcceptShareInvitation    = "AcceptShareInvitation"
	MethodGetShareInvitationStatus = "GetShareInvitationStatus"
	MethodFetchFile                = "FetchFile"
	MethodDropDownload             = "DropDownload"
	MethodGetDownloads             = "GetDownloads"
	MethodSuspend                  = "Suspend"
	MethodResume                   = "Resume"
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
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	Path            string           `json:"path"`
	State           State            `json:"state"`
	LastSyncedAt    string           `json:"lastSyncedAt"`
	ConflictCount   int              `json:"conflictCount"`
	PausedDeletions *PausedDeletions `json:"pausedDeletions"`
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

// PreviewConnectRequest asks how many notes of a vault differ from the notes of a synced vault.
type PreviewConnectRequest struct {
	VaultID  string `json:"vaultId"`
	RemoteID string `json:"remoteId"`
}

// PreviewConnectResponse is the number of notes that have another text on each side.
type PreviewConnectResponse struct {
	DifferCount int `json:"differCount"`
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
// then the server is not asked and Phase is empty. A time is RFC 3339, and empty when the server sent none. Plan is the
// id of the plan, empty while the phase is none; QuotaBytes is 0 then. UsedBytes may be above QuotaBytes.
type GetSubscriptionResponse struct {
	HasIdentity bool   `json:"hasIdentity"`
	Phase       Phase  `json:"phase"`
	PeriodEndAt string `json:"periodEndAt"`
	WritesEndAt string `json:"writesEndAt"`
	DeleteAt    string `json:"deleteAt"`
	Plan        string `json:"plan"`
	QuotaBytes  uint64 `json:"quotaBytes"`
	UsedBytes   uint64 `json:"usedBytes"`
	IsTrial     bool   `json:"isTrial,omitempty"`
}

// CreateCheckoutRequest asks for a payment page for the plan with this id.
type CreateCheckoutRequest struct {
	Plan string `json:"plan"`
}

// CreateCheckoutResponse is the address of the payment page.
type CreateCheckoutResponse struct {
	CheckoutURL string `json:"checkoutUrl"`
}

// ListPlansRequest asks for the plans the sync server sells.
type ListPlansRequest struct{}

// Plan is a plan of the sync server: its id, its name, how many bytes it holds and its price for one period in cents
// of the currency. The price is information; the payment page holds the one that is charged.
type Plan struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	QuotaBytes uint64 `json:"quotaBytes"`
	PriceCents uint64 `json:"priceCents"`
	Currency   string `json:"currency"`
}

// ListPlansResponse holds the plans ascending by quota.
type ListPlansResponse struct {
	Plans []Plan `json:"plans"`
}

// ChangePlanRequest asks to move the active subscription to the plan with this id.
type ChangePlanRequest struct {
	Plan string `json:"plan"`
}

// ChangePlanResponse is the subscription after the change.
type ChangePlanResponse struct {
	Subscription GetSubscriptionResponse `json:"subscription"`
}

// CreatePortalSessionRequest asks for the address of the page where the person manages the subscription.
type CreatePortalSessionRequest struct{}

type CreatePortalSessionResponse struct {
	PortalURL string `json:"portalUrl"`
}

// FetchFileRequest asks to download the absent file at Path of a vault.
type FetchFileRequest struct {
	VaultID string `json:"vaultId"`
	Path    string `json:"path"`
}

// FetchFileResponse is the answer to a download that went through.
type FetchFileResponse struct{}

// DropDownloadRequest asks to remove the download of the file at Path of a vault.
type DropDownloadRequest struct {
	VaultID string `json:"vaultId"`
	Path    string `json:"path"`
}

// DropDownloadResponse is the answer to a removal that went through.
type DropDownloadResponse struct{}

// GetDownloadsRequest asks for the downloads that run.
type GetDownloadsRequest struct{}

// Download is a download that runs: the bytes that came and the bytes of the file.
type Download struct {
	VaultID string `json:"vaultId"`
	Path    string `json:"path"`
	Bytes   int64  `json:"bytes"`
	Total   int64  `json:"total"`
}

type GetDownloadsResponse struct {
	Downloads []Download `json:"downloads"`
}

// SuspendRequest tells that the application left the screen.
type SuspendRequest struct{}

// SuspendResponse is the answer to a suspension that stopped the sync.
type SuspendResponse struct{}

// ResumeRequest tells that the application is on the screen again.
type ResumeRequest struct{}

// ResumeResponse is the answer to a resumption that started the sync.
type ResumeResponse struct{}

// MethodResolveDeletions answers the pause before many removals.
const MethodResolveDeletions = "ResolveDeletions"

// Direction is which way the removals that wait for an answer go.
type Direction string

const (
	// DirectionSend is removals of this device that wait to be sent.
	DirectionSend Direction = "send"
	// DirectionApply is removals of another device that wait to be applied here.
	DirectionApply Direction = "apply"
)

// Choice is the answer to the removals that wait.
type Choice string

const (
	// ChoiceRemove goes on with the removals.
	ChoiceRemove Choice = "remove"
	// ChoiceKeep restores the files.
	ChoiceKeep Choice = "keep"
)

// PausedDeletions is the removals of a vault that wait for an answer. Direction says which way they go.
type PausedDeletions struct {
	Count     int       `json:"count"`
	Tracked   int       `json:"tracked"`
	Direction Direction `json:"direction"`
}

// ResolveDeletionsRequest answers for a vault.
type ResolveDeletionsRequest struct {
	VaultID string `json:"vaultId"`
	Choice  Choice `json:"choice"`
}

// ResolveDeletionsResponse is the answer to a choice that was taken.
type ResolveDeletionsResponse struct{}

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

// StopSyncingVaultRequest asks to stop syncing a vault on this device. The files and the synced vault on the server stay.
type StopSyncingVaultRequest struct {
	VaultID string `json:"vaultId"`
}

// StopSyncingVaultResponse is the answer to a vault that stopped syncing.
type StopSyncingVaultResponse struct{}

// GetAccountStateRequest asks whether the account is being deleted on the server.
type GetAccountStateRequest struct{}

// GetAccountStateResponse is the state of the deletion of the account. HasIdentity is false on a device with no identity.
// Account is the account fingerprint, the name of the account that the person types to confirm a deletion. DeleteAt is an RFC 3339
// time, empty unless IsDeleting.
type GetAccountStateResponse struct {
	HasIdentity bool   `json:"hasIdentity"`
	Account     string `json:"account"`
	IsDeleting  bool   `json:"isDeleting"`
	DeleteAt    string `json:"deleteAt"`
}

// DeleteCloudDataRequest asks the server to delete the account after its grace period.
type DeleteCloudDataRequest struct{}

// DeleteCloudDataResponse says when the server deletes the account for good, an RFC 3339 time.
type DeleteCloudDataResponse struct {
	DeleteAt string `json:"deleteAt"`
}

// RestoreAccountRequest cancels the deletion of an account that is being deleted.
type RestoreAccountRequest struct{}

type RestoreAccountResponse struct{}

// ExportAccountDataRequest asks for what the server holds of the account.
type ExportAccountDataRequest struct{}

// ExportAccountDataResponse is the JSON document of what the server holds of the account, as text.
type ExportAccountDataResponse struct {
	Document string `json:"document"`
}

// StartShareInvitationRequest makes the link a folder is shared by.
type StartShareInvitationRequest struct {
	VaultID string `json:"vaultId"`
	Folder  string `json:"folder"`
}

// StartShareInvitationResponse is the link to copy and send, and when it stops working (RFC 3339). The link works once,
// for 24 hours at most. ID names the link in ListShareInvitations and RevokeShareInvitation.
type StartShareInvitationResponse struct {
	ID        string `json:"id"`
	Link      string `json:"link"`
	ExpiresAt string `json:"expiresAt"`
}

// ListShareInvitationsRequest asks for the links of this computer that work still.
type ListShareInvitationsRequest struct{}

// ShareInvitation is a link that works still. IsRequested says a person joined it and waits for an answer, which
// ListShareRequests lists.
type ShareInvitation struct {
	ID          string `json:"id"`
	Folder      string `json:"folder"`
	ExpiresAt   string `json:"expiresAt"`
	IsRequested bool   `json:"isRequested"`
}

// ListShareInvitationsResponse is the links that were not used up, revoked or expired.
type ListShareInvitationsResponse struct {
	Invitations []ShareInvitation `json:"invitations"`
}

// RevokeShareInvitationRequest makes a link stop working.
type RevokeShareInvitationRequest struct {
	ID string `json:"id"`
}

// RevokeShareInvitationResponse is the answer to a link that was revoked.
type RevokeShareInvitationResponse struct{}

// ListShareRequestsRequest asks for the people who wait for this person to answer.
type ListShareRequestsRequest struct{}

// ShareRequest is a device that wants to read a folder, with the name it announced and the fingerprint of its account. Code is the confirmation code the person sees too; the owner
// allows the request only when the two match.
type ShareRequest struct {
	RequestID   string `json:"requestId"`
	VaultID     string `json:"vaultId"`
	Folder      string `json:"folder"`
	DeviceName  string `json:"deviceName"`
	Fingerprint string `json:"fingerprint"`
	Code        string `json:"code"`
}

// ListShareRequestsResponse is the requests that wait. The plugin reports the health item share.requestWaiting while
// there is one.
type ListShareRequestsResponse struct {
	Requests []ShareRequest `json:"requests"`
}

// ApproveShareRequestRequest lets a person read the folder.
type ApproveShareRequestRequest struct {
	RequestID string `json:"requestId"`
}

// ApproveShareRequestResponse is the answer to a request that was approved: the person is a reader of the folder.
type ApproveShareRequestResponse struct{}

// DeclineShareRequestRequest refuses a person.
type DeclineShareRequestRequest struct {
	RequestID string `json:"requestId"`
}

// DeclineShareRequestResponse is the answer to a request that was declined.
type DeclineShareRequestResponse struct{}

// PreviewShareInvitationRequest holds a link a person pasted.
type PreviewShareInvitationRequest struct {
	Link string `json:"link"`
}

// PreviewShareInvitationResponse is what a link tells before it is accepted: the folder, the fingerprint of the owner
// account and the moment the link stops working. IsReadOnly is always true: a shared folder is read-only for the person.
type PreviewShareInvitationResponse struct {
	Folder           string `json:"folder"`
	OwnerFingerprint string `json:"ownerFingerprint"`
	ExpiresAt        string `json:"expiresAt"`
	IsReadOnly       bool   `json:"isReadOnly"`
}

// AcceptShareInvitationRequest sends the owner of a link a request to read the folder. DeviceName is the name of this computer the owner sees; empty makes the backend use the host name.
type AcceptShareInvitationRequest struct {
	Link       string `json:"link"`
	DeviceName string `json:"deviceName"`
}

// AcceptShareInvitationResponse is the request that was sent; GetShareInvitationStatus follows it.
type AcceptShareInvitationResponse struct {
	RequestID        string `json:"requestId"`
	Folder           string `json:"folder"`
	OwnerFingerprint string `json:"ownerFingerprint"`
}

// ShareRequestPhase is how a request of this computer stands.
type ShareRequestPhase string

const (
	// ShareRequestConnecting has no answer from the owner yet; no code shows.
	ShareRequestConnecting ShareRequestPhase = "connecting"
	// ShareRequestWaiting shows the code and waits for the owner to allow.
	ShareRequestWaiting ShareRequestPhase = "waiting"
	// ShareRequestReceived is allowed; the folder is in ListReceived.
	ShareRequestReceived ShareRequestPhase = "received"
	// ShareRequestDeclined was refused by the owner.
	ShareRequestDeclined ShareRequestPhase = "declined"
	// ShareRequestUsed is a link another person used before.
	ShareRequestUsed ShareRequestPhase = "used"
	// ShareRequestExpired is a link that stopped working before the owner answered.
	ShareRequestExpired ShareRequestPhase = "expired"
	// ShareRequestFailed ended on an error, in Error.
	ShareRequestFailed ShareRequestPhase = "failed"
)

// GetShareInvitationStatusRequest names a request AcceptShareInvitation sent.
type GetShareInvitationStatusRequest struct {
	RequestID string `json:"requestId"`
}

// GetShareInvitationStatusResponse is how a request stands. Code is set from the phase waiting on.
type GetShareInvitationStatusResponse struct {
	Phase            ShareRequestPhase `json:"phase"`
	Code             string            `json:"code,omitempty"`
	Folder           string            `json:"folder"`
	OwnerFingerprint string            `json:"ownerFingerprint"`
	ExpiresAt        string            `json:"expiresAt"`
	Error            string            `json:"error,omitempty"`
}
