package identity

import (
	"slices"
	"time"
)

const linkLifetime = 10 * time.Minute

// LinkProgress is what a link in progress holds besides its keys: where it stands, what this device and the other
// device show of themselves, and the confirmation code. TriesLeft counts the codes the host still takes, or on the
// new device the codes it may still enter.
type LinkProgress struct {
	Role               LinkRole
	Mode               LinkMode
	Phase              LinkPhase
	Reason             LinkReason
	StartedAt          time.Time
	QRText             string
	DeviceName         string
	DeviceFingerprint  string
	AccountFingerprint string
	PeerName           string
	PeerFingerprint    string
	Code               string
	TriesLeft          int
	IsApproved         bool
	IsRebuilt          bool
}

// IsEnded reports whether the link failed, linked the device or was canceled.
func (p LinkProgress) IsEnded() bool {
	return slices.Contains([]LinkPhase{LinkPhaseFailed, LinkPhaseLinked, LinkPhaseCanceled}, p.Phase)
}

// IsExpired reports whether a link that has not ended ran for ten minutes at now.
func (p LinkProgress) IsExpired(now time.Time) bool {
	return !p.IsEnded() && !now.Before(p.StartedAt.Add(linkLifetime))
}

// GetLink returns what a window shows: the QR text only while a link in mode show runs, and the code with the tries
// the host takes only on the host in mode scan while the person compares the code.
func (p LinkProgress) GetLink() Link {
	link := Link{
		Phase:              p.Phase,
		Mode:               p.Mode,
		Role:               p.Role,
		Reason:             p.Reason,
		DeviceName:         p.DeviceName,
		DeviceFingerprint:  p.DeviceFingerprint,
		AccountFingerprint: p.AccountFingerprint,
		IsApproved:         p.IsApproved,
		PeerName:           p.PeerName,
		PeerFingerprint:    p.PeerFingerprint,
	}
	if p.Role == LinkRoleJoiner {
		link.TriesLeft = p.TriesLeft
	}
	if p.Mode == LinkModeShow && !p.IsEnded() {
		link.QRText = p.QRText
	}
	if p.Role == LinkRoleHost && p.Mode == LinkModeScan && p.Phase == LinkPhaseConfirmPeer {
		link.Code = p.Code
		link.TriesLeft = p.TriesLeft
	}
	return link
}

// LinkOutbox is what a link keeps of its channel on the relay and of the blobs the courier has not yet carried.
type LinkOutbox struct {
	Server       string
	ChannelID    []byte
	WantsChannel bool
	IsClosed     bool
	Items        []LinkBlob
}

// LinkBlob is a blob for the other side with the version of the link work that queued it.
type LinkBlob struct {
	Version uint64
	Blob    []byte
}

// Acknowledge drops the blobs that the work of the version covers: the courier carried them.
func (o *LinkOutbox) Acknowledge(version uint64) {
	o.Items = slices.DeleteFunc(o.Items, func(item LinkBlob) bool { return item.Version <= version })
}

// GetWork returns the work of the version: the channel to make or use and copies of the blobs to post. The link is
// active while it runs or has blobs left.
func (o LinkOutbox) GetWork(version uint64) LinkWork {
	work := LinkWork{
		Version:      version,
		IsActive:     !o.IsClosed || len(o.Items) > 0,
		WantsChannel: o.WantsChannel,
		ChannelID:    append([]byte(nil), o.ChannelID...),
		Server:       o.Server,
		IsClosed:     o.IsClosed,
	}
	for _, item := range o.Items {
		work.Outbound = append(work.Outbound, append([]byte(nil), item.Blob...))
	}
	return work
}
