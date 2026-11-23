// Package api is the JSON messages between the settings view of Numen Sync and its backend.
package api

// Service is the name of the service of the plugin.
const Service = "md.numen.tools.sync"

// The methods of the service.
const (
	MethodListVaults          = "ListVaults"
	MethodListRemoteVaults    = "ListRemoteVaults"
	MethodConnectVault        = "ConnectVault"
	MethodGetStatus           = "GetStatus"
	MethodGetSubscription     = "GetSubscription"
	MethodCreateCheckout      = "CreateCheckout"
	MethodListPlans           = "ListPlans"
	MethodChangePlan          = "ChangePlan"
	MethodListProducts        = "ListProducts"
	MethodCreatePortalSession = "CreatePortalSession"
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

// ListProductsRequest asks for the products the sync server sells that are no plans.
type ListProductsRequest struct{}

// Product is a product of the sync server that is no plan. OneTime is true for a single payment. The price is information.
type Product struct {
	ID         string `json:"id"`
	Slug       string `json:"slug"`
	Name       string `json:"name"`
	PriceCents uint64 `json:"priceCents"`
	Currency   string `json:"currency"`
	OneTime    bool   `json:"oneTime"`
}

// ListProductsResponse holds the products in the order of the server.
type ListProductsResponse struct {
	Products []Product `json:"products"`
}

// CreatePortalSessionRequest asks for the address of the page where the person manages the subscription.
type CreatePortalSessionRequest struct{}

// CreatePortalSessionResponse is the address of the portal.
type CreatePortalSessionResponse struct {
	PortalURL string `json:"portalUrl"`
}
