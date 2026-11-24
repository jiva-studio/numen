package identity

import (
	"errors"
	"time"
)

var (
	// ErrLinkActive means a link is already in progress on this device.
	ErrLinkActive = errors.New("a link is already in progress")
	// ErrNoLink means no link is in progress on this device.
	ErrNoLink = errors.New("no link is in progress")
	// ErrNotRevoked means this device was not revoked.
	ErrNotRevoked = errors.New("this device was not revoked")
	// ErrLastDevice means a revoke would leave the account with no device.
	ErrLastDevice = errors.New("the last device of the account cannot be revoked")
	// ErrRevokeRaced means a device was added by the device to revoke while the revoke was made.
	ErrRevokeRaced = errors.New("a device was added while the revoke was made")
	// ErrNoChain means this device holds no chain file.
	ErrNoChain = errors.New("this device has no chain file")
	// ErrDeviceRevoked means the account revoked this device, which no link or revoke can start from.
	ErrDeviceRevoked = errors.New("this device was revoked")
)

type LinkRole string

const (
	LinkRoleHost   LinkRole = "host"
	LinkRoleJoiner LinkRole = "joiner"
)

type LinkMode string

const (
	LinkModeShow LinkMode = "show"
	LinkModeScan LinkMode = "scan"
)

type LinkPhase string

const (
	LinkPhaseWaitingForChannel LinkPhase = "WAITING_FOR_CHANNEL"
	LinkPhaseWaitingForPeer    LinkPhase = "WAITING_FOR_PEER"
	LinkPhaseConfirmPeer       LinkPhase = "CONFIRM_PEER"
	LinkPhaseWaitingForAccept  LinkPhase = "WAITING_FOR_ACCEPT"
	LinkPhaseEnterCode         LinkPhase = "ENTER_CODE"
	LinkPhasePublishing        LinkPhase = "PUBLISHING"
	LinkPhaseConfirmAccount    LinkPhase = "CONFIRM_ACCOUNT"
	LinkPhaseLinked            LinkPhase = "LINKED"
	LinkPhaseFailed            LinkPhase = "FAILED"
	LinkPhaseCanceled          LinkPhase = "CANCELED"
)

type LinkReason string

const (
	LinkReasonDeclined          LinkReason = "declined"
	LinkReasonDeviceHasIdentity LinkReason = "device_has_identity"
	LinkReasonWrongCode         LinkReason = "wrong_code"
	LinkReasonTooManyWrongCodes LinkReason = "too_many_wrong_codes"
	LinkReasonStaleChain        LinkReason = "stale_chain"
	LinkReasonCanceled          LinkReason = "canceled"
	LinkReasonExpired           LinkReason = "expired"
	LinkReasonInvalid           LinkReason = "invalid"
)

// Link is what a window shows of the link in progress. It holds no key; Code is set for the host waiting for an
// answer to a new device, and QRText in mode show only.
type Link struct {
	Phase              LinkPhase
	Mode               LinkMode
	Role               LinkRole
	QRText             string
	Code               string
	DeviceName         string
	DeviceFingerprint  string
	IsApproved         bool
	PeerName           string
	PeerFingerprint    string
	AccountFingerprint string
	TriesLeft          int
	Reason             LinkReason
}

// Device is one device of the account as the list of devices shows it.
type Device struct {
	Name        string
	Fingerprint string
	AddedAt     time.Time
	IsThis      bool
	IsRevoked   bool
}

// LinkWork is what the courier of a link carries: opaque blobs for the other side and the channel.
type LinkWork struct {
	Version      uint64
	IsActive     bool
	WantsChannel bool
	ChannelID    []byte
	Server       string
	Outbound     [][]byte
	IsClosed     bool
}

// ChainPublish is a chain the server must hold before the core goes on.
type ChainPublish struct {
	IsPending bool
	Chain     []byte
}
