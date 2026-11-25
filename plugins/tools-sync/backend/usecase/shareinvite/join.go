package shareinvite

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"time"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	domainshare "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// Phase is how a request to read a folder stands, as the person who made it sees it.
type Phase string

const (
	// PhaseConnecting is a request whose first message the sharer did not answer yet.
	PhaseConnecting Phase = "connecting"
	// PhaseWaiting is a request the sharer has to answer; the code is known.
	PhaseWaiting Phase = "waiting"
	// PhaseReceived is a request the sharer approved and whose folder is on this computer.
	PhaseReceived Phase = "received"
	// PhaseDeclined is a request the sharer declined.
	PhaseDeclined Phase = "declined"
	// PhaseUsed is a link another person used before.
	PhaseUsed Phase = "used"
	// PhaseExpired is a request whose link stopped working before the sharer answered.
	PhaseExpired Phase = "expired"
	// PhaseFailed is a request that ended on an error; Status has it.
	PhaseFailed Phase = "failed"
)

// join is the request of this computer to read a folder.
type join struct {
	id          []byte
	invitation  domainshare.LinkInvitation
	scanner     *identity.LinkScanner
	channel     *identity.LinkChannel
	chain       []byte
	name        string
	phase       Phase
	code        string
	failure     error
	hasFinished bool
}

// Preview is what a link tells before the person accepts it: the folder, the sharer and the moment the link stops
// working. A shared folder is always read-only for the person.
type Preview struct {
	Folder           string
	OwnerFingerprint string
	ExpiresAt        time.Time
	IsReadOnly       bool
}

// ReadLink reads a link without contacting anybody.
type ReadLink struct{ deps Deps }

// NewReadLink returns the use case over the dependencies.
func NewReadLink(deps Deps) *ReadLink { return &ReadLink{deps: deps} }

// Execute refuses a link that is damaged, expired or for another server, and returns what it tells.
func (u *ReadLink) Execute(text string) (Preview, error) {
	invitation, err := checkLink(u.deps, text)
	if err != nil {
		return Preview{}, err
	}
	return Preview{
		Folder: invitation.Folder, OwnerFingerprint: invitation.Fingerprint, ExpiresAt: invitation.ExpiresAt, IsReadOnly: true,
	}, nil
}

func checkLink(deps Deps, text string) (domainshare.LinkInvitation, error) {
	invitation, err := domainshare.ParseLink(text)
	if err != nil {
		return domainshare.LinkInvitation{}, err
	}
	if !syncprotocol.IsSameAddress(invitation.Payload.Server, deps.Server) {
		return domainshare.LinkInvitation{}, domainshare.ErrLinkOtherServer
	}
	if deps.Clock.Now().After(invitation.ExpiresAt) {
		return domainshare.LinkInvitation{}, domainshare.ErrLinkExpired
	}
	return invitation, nil
}

// JoinRequest is a request that was sent: its id is the one Status takes.
type JoinRequest struct {
	ID               string
	Folder           string
	OwnerFingerprint string
}

// Join asks the sharer of a link to read the folder.
type Join struct{ deps Deps }

// NewJoin returns the use case over the dependencies.
func NewJoin(deps Deps) *Join { return &Join{deps: deps} }

// Execute checks the link, sends the first message of the handshake and returns; the rest runs in the courier of the
// plugin. A link of this account itself is ErrOwnIdentity.
func (u *Join) Execute(ctx context.Context, text, deviceName string) (JoinRequest, error) {
	invitation, err := checkLink(u.deps, text)
	if err != nil {
		return JoinRequest{}, err
	}
	own, err := u.deps.Identity.GetPublicIdentity(ctx)
	if err != nil {
		return JoinRequest{}, err
	}
	chain, err := identity.ParseChain(own.Chain)
	if err != nil {
		return JoinRequest{}, err
	}
	if chain.Root().Fingerprint().String() == invitation.Fingerprint {
		return JoinRequest{}, domainshare.ErrOwnIdentity
	}
	scanner, first, err := identity.NewLinkScanner(u.deps.Random, invitation.Payload)
	if err != nil {
		return JoinRequest{}, domainshare.ErrBadLink
	}
	id := bytes.Clone(invitation.Payload.Channel[:])
	if _, err := u.deps.Relay.SendMessage(ctx, id, ports.LinkSideDisplayer, first); err != nil {
		if errors.Is(err, syncprotocol.ErrChannelGone) {
			return JoinRequest{}, domainshare.ErrLinkExpired
		}
		return JoinRequest{}, err
	}
	mine := &join{
		id: id, invitation: invitation, scanner: scanner, channel: nil, chain: own.Chain, name: domainshare.CleanDeviceName(getDeviceName(u.deps, deviceName)),
		phase: PhaseConnecting, code: "", failure: nil, hasFinished: false,
	}
	key := hex.EncodeToString(id)
	u.deps.Hub.mu.Lock()
	u.deps.Hub.joins[key] = mine
	u.deps.Hub.mu.Unlock()
	if err := u.deps.Hub.add(ctx, func(run context.Context) { watchJoin(run, u.deps, mine) }); err != nil {
		return JoinRequest{}, err
	}
	return JoinRequest{ID: key, Folder: invitation.Folder, OwnerFingerprint: invitation.Fingerprint}, nil
}

// Status is how a request stands.
type Status struct {
	Phase            Phase
	Code             string
	Folder           string
	OwnerFingerprint string
	ExpiresAt        time.Time
	Error            string
}

// GetStatus reads a request of this computer.
type GetStatus struct{ deps Deps }

// NewGetStatus returns the use case over the dependencies.
func NewGetStatus(deps Deps) *GetStatus { return &GetStatus{deps: deps} }

// Execute returns the phase of the request; the code shows from PhaseWaiting on. A request that was not answered by the
// time its link expires is PhaseExpired.
func (u *GetStatus) Execute(id string) (Status, error) {
	u.deps.Hub.mu.Lock()
	defer u.deps.Hub.mu.Unlock()
	mine, isFound := u.deps.Hub.joins[id]
	if !isFound {
		return Status{}, domainshare.ErrUnknownRequest
	}
	phase := mine.phase
	if !mine.hasFinished && u.deps.Clock.Now().After(mine.invitation.ExpiresAt) {
		phase = PhaseExpired
	}
	text := ""
	if mine.failure != nil {
		text = mine.failure.Error()
	}
	return Status{
		Phase: phase, Code: mine.code, Folder: mine.invitation.Folder, OwnerFingerprint: mine.invitation.Fingerprint,
		ExpiresAt: mine.invitation.ExpiresAt, Error: text,
	}, nil
}

func watchJoin(ctx context.Context, deps Deps, mine *join) {
	var after uint32
	for ctx.Err() == nil && !isJoinFinished(deps.Hub, mine) {
		if deps.Clock.Now().After(mine.invitation.ExpiresAt) {
			finishJoin(deps.Hub, mine, PhaseExpired, nil)
			return
		}
		answer, err := deps.Relay.WaitMessages(ctx, mine.id, ports.LinkSideScanner, after, waitFor)
		if err != nil {
			if !retryJoin(ctx, deps, mine, err) {
				return
			}
			continue
		}
		for _, one := range answer.Messages {
			after = max(after, one.Seq)
			receiveAtJoin(ctx, deps, mine, one.Body)
		}
		if answer.IsClosed {
			finishJoin(deps.Hub, mine, PhaseExpired, nil)
			return
		}
	}
}

func retryJoin(ctx context.Context, deps Deps, mine *join, failure error) bool {
	switch {
	case errors.Is(failure, syncprotocol.ErrChannelGone):
		finishJoin(deps.Hub, mine, PhaseExpired, nil)
		return false
	case errors.Is(failure, syncprotocol.ErrLinkBusy) && mine.channel == nil:
		finishJoin(deps.Hub, mine, PhaseUsed, domainshare.ErrLinkUsed)
		return false
	}
	return deps.Clock.Wait(ctx, retryAfter) == nil
}

func isJoinFinished(hub *Hub, mine *join) bool {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	return mine.hasFinished
}

func finishJoin(hub *Hub, mine *join, phase Phase, failure error) {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	if mine.hasFinished {
		return
	}
	mine.phase, mine.failure, mine.hasFinished = phase, failure, true
}

func receiveAtJoin(ctx context.Context, deps Deps, mine *join, body []byte) {
	if mine.channel == nil {
		bindJoin(ctx, deps, mine, body)
		return
	}
	plain, err := mine.channel.Open(body)
	if err != nil {
		return
	}
	m, err := domainshare.DecodeLinkMessage(plain)
	if err != nil {
		return
	}
	switch m.Kind {
	case domainshare.MessageApproved:
		accept(ctx, deps, mine, m.Invitation)
	case domainshare.MessageDeclined:
		finishJoin(deps.Hub, mine, PhaseDeclined, domainshare.ErrLinkDeclined)
	}
}

func bindJoin(ctx context.Context, deps Deps, mine *join, body []byte) {
	channel, err := mine.scanner.Accept(body)
	if err != nil {
		return
	}
	request := channel.Seal(domainshare.EncodeLinkMessage(domainshare.LinkMessage{Kind: domainshare.MessageJoin, Chain: mine.chain, Name: mine.name, Invitation: ""}))
	if _, err := deps.Relay.SendMessage(ctx, mine.id, ports.LinkSideDisplayer, request); err != nil {
		return
	}
	deps.Hub.mu.Lock()
	mine.channel, mine.code, mine.phase = channel, channel.Code(), PhaseWaiting
	deps.Hub.mu.Unlock()
}

func accept(ctx context.Context, deps Deps, mine *join, invitation string) {
	owner, err := deps.Receiving.GetInvitationOwner(ctx, invitation)
	if err == nil && owner != mine.invitation.Fingerprint {
		err = domainshare.ErrLinkMismatch
	}
	if err != nil {
		finishJoin(deps.Hub, mine, PhaseFailed, err)
		return
	}
	if err := deps.Receiving.AcceptInvitation(ctx, invitation); err != nil {
		finishJoin(deps.Hub, mine, PhaseFailed, err)
		return
	}
	finishJoin(deps.Hub, mine, PhaseReceived, nil)
}

func getDeviceName(deps Deps, given string) string {
	if given != "" {
		return given
	}
	return deps.Device.GetName()
}
