package api

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

// ApproveShareRequestRequest lets a person read the folder. Fingerprint is the fingerprint of the person that the sharer confirmed with
// the person; any other fingerprint refuses the request.
type ApproveShareRequestRequest struct {
	RequestID   string `json:"requestId"`
	Fingerprint string `json:"fingerprint"`
}

// ApproveShareRequestResponse is the answer to a request that was approved: the person is a reader of the folder.
type ApproveShareRequestResponse struct{}

// DeclineShareRequestRequest refuses a person.
type DeclineShareRequestRequest struct {
	RequestID string `json:"requestId"`
}

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
