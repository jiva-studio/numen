package shareinvite

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	domainshare "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// HealthKey is the item the plugin reports while a person waits for the sharer to answer.
const HealthKey = "share-request-waiting"

func watchOffer(ctx context.Context, deps Deps, o *offer) {
	var after uint32
	defer finishOffer(ctx, deps, o)
	for ctx.Err() == nil && !deps.Clock.Now().After(o.expiresAt) && !isOfferDone(deps.Hub, o) {
		answer, err := deps.Relay.WaitMessages(ctx, o.id, ports.LinkSideDisplayer, after, waitFor)
		if errors.Is(err, syncprotocol.ErrChannelGone) {
			return
		}
		if err != nil {
			if deps.Clock.Wait(ctx, retryAfter) != nil {
				return
			}
			continue
		}
		for _, one := range answer.Messages {
			after = max(after, one.Seq)
			receiveAtSharer(ctx, deps, o, one.Body)
		}
		if answer.IsClosed {
			return
		}
	}
}

func isOfferDone(hub *Hub, o *offer) bool {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	return o.isDone
}

func finishOffer(ctx context.Context, deps Deps, o *offer) {
	deps.Hub.mu.Lock()
	o.isDone = true
	o.request = nil
	delete(deps.Hub.offers, hex.EncodeToString(o.id))
	deps.Hub.mu.Unlock()
	reportWaiting(context.WithoutCancel(ctx), deps)
}

func receiveAtSharer(ctx context.Context, deps Deps, o *offer, body []byte) {
	deps.Hub.mu.Lock()
	channel := o.channel
	deps.Hub.mu.Unlock()
	if channel == nil {
		bind(ctx, deps, o, body)
		return
	}
	plain, err := channel.Open(body)
	if err != nil {
		return
	}
	read(ctx, deps, o, plain)
}

func bind(ctx context.Context, deps Deps, o *offer, first []byte) {
	reply, channel, err := o.displayer.Accept(first)
	if err != nil {
		return
	}
	if _, err := deps.Relay.SendMessage(ctx, o.id, ports.LinkSideScanner, reply); err != nil {
		return
	}
	deps.Hub.mu.Lock()
	o.channel = channel
	deps.Hub.mu.Unlock()
}

func read(ctx context.Context, deps Deps, o *offer, plain []byte) {
	m, err := domainshare.DecodeLinkMessage(plain)
	if err != nil || m.Kind != domainshare.MessageJoin {
		return
	}
	chain, err := domainshare.ParseReaderCode(domainshare.NewReaderCode(m.Chain))
	if err != nil {
		return
	}
	deps.Hub.mu.Lock()
	if o.request == nil {
		o.request = &pending{
			chain: m.Chain, name: domainshare.CleanDeviceName(m.Name), fingerprint: chain.Root().Fingerprint().String(),
			code: o.channel.GetCodeFor(o.owner, chain.Root().Fingerprint().String()), isBusy: false,
		}
	}
	deps.Hub.mu.Unlock()
	reportWaiting(ctx, deps)
}

func reportWaiting(ctx context.Context, deps Deps) {
	requests := listRequests(deps.Hub)
	if len(requests) == 0 {
		if err := deps.Health.Clear(ctx, HealthKey); err != nil {
			return
		}
		return
	}
	first := requests[0]
	item := health.Item{
		Key: HealthKey, Severity: health.Warning,
		Text:   fmt.Sprintf("%s wants to read %s", first.DeviceName, first.Label),
		Detail: fmt.Sprintf("Confirmation code %s. Open Numen Sync to allow or decline.", first.Code),
	}
	if err := deps.Health.Report(ctx, item); err != nil {
		return
	}
}

func listRequests(hub *Hub) []Request {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	var requests []Request
	for key, o := range hub.offers {
		if o.request == nil || o.isDone {
			continue
		}
		requests = append(requests, Request{
			ID: key, VaultID: o.vault, Folder: o.folder, Label: o.getLabel(), DeviceName: o.request.name, Fingerprint: o.request.fingerprint, Code: o.request.code,
		})
	}
	return requests
}

func (o *offer) getLabel() string {
	if o.folder == "" {
		return o.name
	}
	return o.folder
}
