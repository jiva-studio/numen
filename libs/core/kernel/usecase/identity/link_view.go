package identity

import (
	"fmt"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

func (l *pendingLink) view() identitydomain.Link {
	link := identitydomain.Link{
		Phase:              l.phase,
		Mode:               l.mode,
		Role:               l.role,
		Reason:             l.reason,
		DeviceName:         l.own.name,
		DeviceFingerprint:  l.own.fingerprint,
		AccountFingerprint: l.own.account,
		IsApproved:         l.host.isApproved,
		PeerName:           l.host.join.name,
		TriesLeft:          l.joiner.triesLeft,
	}
	if l.mode == identitydomain.LinkModeShow && !l.isEnded() {
		link.QRText = l.qrText
	}
	if len(l.host.join.device.Sign) > 0 {
		link.PeerFingerprint = l.host.join.device.Fingerprint().String()
	}
	if l.role == identitydomain.LinkRoleHost && l.mode == identitydomain.LinkModeScan && l.phase == identitydomain.LinkPhaseConfirmPeer {
		link.Code = l.host.code
		link.TriesLeft = l.host.triesLeft
	}
	return link
}

func (s *LinkStore) getCurrentLocked() (*pendingLink, error) {
	if s.current == nil || s.current.phase == identitydomain.LinkPhaseCancelled {
		return nil, identitydomain.ErrNoLink
	}
	return s.current, nil
}

func newOwnIdentity(state identitylib.State, name string) ownIdentity {
	own := state.Record.Device.Public()
	return ownIdentity{name: name, fingerprint: own.Fingerprint().String(), account: state.Chain.Root().Fingerprint().String()}
}

func formatCode(value uint32) string { return fmt.Sprintf("%0*d", codeDigits, value) }
