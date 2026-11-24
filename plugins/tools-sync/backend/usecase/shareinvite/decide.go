package shareinvite

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	domainshare "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/share"
)

const fingerprintTag = 6

// ListRequests lists the people who wait for the sharer to answer.
type ListRequests struct{ deps Deps }

// NewListRequests returns the use case over the dependencies.
func NewListRequests(deps Deps) *ListRequests { return &ListRequests{deps: deps} }

// Execute returns the requests, the oldest id first.
func (u *ListRequests) Execute(_ context.Context) []Request {
	requests := listRequests(u.deps.Hub)
	slices.SortFunc(requests, func(a, b Request) int { return strings.Compare(a.ID, b.ID) })
	return requests
}

// Invitation is a link that works still: whether a person joined it, and when it stops working.
type Invitation struct {
	ID          string
	Folder      string
	ExpiresAt   time.Time
	IsRequested bool
}

// ListInvitations lists the links of the sharer that work still.
type ListInvitations struct{ deps Deps }

// NewListInvitations returns the use case over the dependencies.
func NewListInvitations(deps Deps) *ListInvitations { return &ListInvitations{deps: deps} }

// Execute returns the links that were not used up, revoked or expired, the oldest id first.
func (u *ListInvitations) Execute(_ context.Context) []Invitation {
	hub := u.deps.Hub
	hub.mu.Lock()
	var listed []Invitation
	for key, o := range hub.offers {
		if !o.isDone {
			listed = append(listed, Invitation{ID: key, Folder: o.folder, ExpiresAt: o.expiresAt, IsRequested: o.request != nil})
		}
	}
	hub.mu.Unlock()
	slices.SortFunc(listed, func(a, b Invitation) int { return strings.Compare(a.ID, b.ID) })
	return listed
}

// claim returns the offer and its request, marked busy so that two answers do not run at once.
func claim(deps Deps, id string) (*offer, *pending, error) {
	deps.Hub.mu.Lock()
	defer deps.Hub.mu.Unlock()
	o, isFound := deps.Hub.offers[id]
	if !isFound || o.isDone || o.request == nil || o.request.isBusy {
		return nil, nil, domainshare.ErrUnknownRequest
	}
	if deps.Clock.Now().After(o.expiresAt) {
		return nil, nil, domainshare.ErrLinkExpired
	}
	o.request.isBusy = true
	return o, o.request, nil
}

func release(hub *Hub, p *pending) {
	hub.mu.Lock()
	p.isBusy = false
	hub.mu.Unlock()
}

func end(hub *Hub, o *offer) {
	hub.mu.Lock()
	o.isDone = true
	hub.mu.Unlock()
}

func send(ctx context.Context, deps Deps, o *offer, m message) error {
	deps.Hub.mu.Lock()
	sealed := o.channel.Seal(encodeMessage(m))
	deps.Hub.mu.Unlock()
	_, err := deps.Relay.SendMessage(ctx, o.id, ports.LinkSideScanner, sealed)
	return err
}

// Approve lets a person read a folder.
type Approve struct{ deps Deps }

// NewApprove returns the use case over the dependencies.
func NewApprove(deps Deps) *Approve { return &Approve{deps: deps} }

// Execute shares the folder if it is not shared yet, adds the person as a reader, and sends the person the invitation
// over the channel of the link. The link is used up. A failure before the reader is added leaves the request to
// answer again; a request after its link expired is ErrLinkExpired.
func (u *Approve) Execute(ctx context.Context, id string) error {
	o, p, err := claim(u.deps, id)
	if err != nil {
		return err
	}
	invited, err := addReader(ctx, u.deps, o, p)
	if err != nil {
		release(u.deps.Hub, p)
		return err
	}
	end(u.deps.Hub, o)
	reportWaiting(ctx, u.deps)
	return send(ctx, u.deps, o, message{Kind: kindApproved, Chain: nil, Name: "", Invitation: invited})
}

func addReader(ctx context.Context, deps Deps, o *offer, p *pending) (string, error) {
	input := share.AddReaderInput{VaultID: o.vault, Folder: o.folder, Code: domainshare.NewReaderCode(p.chain), Label: p.name, IsConfirmed: false}
	add := share.NewShareFolder(deps.Sharing)
	invited, err := add.Execute(ctx, input)
	if errors.Is(err, domainshare.ErrChangedIdentity) {
		input.Label = p.name + " (" + p.fingerprint[:min(fingerprintTag, len(p.fingerprint))] + ")"
		invited, err = add.Execute(ctx, input)
	}
	return invited.Text, err
}

// Decline refuses a person.
type Decline struct{ deps Deps }

// NewDecline returns the use case over the dependencies.
func NewDecline(deps Deps) *Decline { return &Decline{deps: deps} }

// Execute tells the person over the channel that the request was declined and uses the link up.
func (u *Decline) Execute(ctx context.Context, id string) error {
	o, p, err := claim(u.deps, id)
	if err != nil {
		return err
	}
	end(u.deps.Hub, o)
	reportWaiting(ctx, u.deps)
	err = send(ctx, u.deps, o, message{Kind: kindDeclined, Chain: nil, Name: "", Invitation: ""})
	release(u.deps.Hub, p)
	return err
}

// Revoke makes a link stop working.
type Revoke struct{ deps Deps }

// NewRevoke returns the use case over the dependencies.
func NewRevoke(deps Deps) *Revoke { return &Revoke{deps: deps} }

// Execute closes the channel of a link that was not used up, and drops a request that waits on it.
func (u *Revoke) Execute(ctx context.Context, id string) error {
	u.deps.Hub.mu.Lock()
	o, isFound := u.deps.Hub.offers[id]
	if !isFound || o.isDone {
		u.deps.Hub.mu.Unlock()
		return domainshare.ErrUnknownRequest
	}
	o.isDone = true
	u.deps.Hub.mu.Unlock()
	reportWaiting(ctx, u.deps)
	err := u.deps.Relay.CloseChannel(ctx, o.id)
	if err != nil && !errors.Is(err, syncprotocol.ErrChannelGone) {
		return err
	}
	return nil
}
