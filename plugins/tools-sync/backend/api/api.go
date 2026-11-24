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

// StopSyncingVaultRequest asks to stop syncing a vault on this device. The files and the synced vault on the server stay.
type StopSyncingVaultRequest struct {
	VaultID string `json:"vaultId"`
}

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
