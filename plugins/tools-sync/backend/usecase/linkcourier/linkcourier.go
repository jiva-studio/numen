// Package linkcourier carries the blobs of a link between the core of this device and the relay of the sync server.
package linkcourier

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

const (
	healthKey       = "link-needs-server"
	healthText      = "Linking needs the sync server"
	healthOtherKey  = "link-other-server"
	healthOtherText = "The QR code is for another sync server"
	workWait        = 20 * time.Second
	retryAfter      = time.Second
	inboundWait     = 10 * time.Second
)

// Deps are what the courier needs.
type Deps struct {
	Core   ports.LinkCore
	Relay  ports.Relay
	Health ports.Health
	Clock  ports.Clock
	Server string
}

// Courier runs for as long as the plugin does.
type Courier struct {
	deps     Deps
	isRaised bool
}

// NewCourier returns the courier over the dependencies.
func NewCourier(deps Deps) *Courier { return &Courier{deps: deps, isRaised: false} }

// channelState is the channel the courier works on: whether it made it, which side it is, and the reader of the
// messages of the other side.
type channelState struct {
	id        []byte
	side      ports.LinkSide
	isClosed  bool
	stopWaits context.CancelFunc
}

// Execute carries work until ctx ends. The core holds every key and decides when the link ends; the courier posts what
// the core passes on, passes what the other side posts, and closes the channel when the core says the link ended.
func (u *Courier) Execute(ctx context.Context) error {
	var (
		since   uint64
		created []byte
		current channelState
	)
	defer func() { stopWaits(&current) }()
	for {
		work, err := u.deps.Core.GetLinkWork(ctx, since, workWait)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			if waitErr := u.deps.Clock.Wait(ctx, retryAfter); waitErr != nil {
				return waitErr
			}
			continue
		}
		if work.Version == since && !work.IsActive {
			continue
		}
		if u.carry(ctx, work, &created, &current) {
			since = work.Version
		} else if waitErr := u.deps.Clock.Wait(ctx, retryAfter); waitErr != nil {
			return waitErr
		}
	}
}

func stopWaits(current *channelState) {
	if current.stopWaits != nil {
		current.stopWaits()
	}
}

func (u *Courier) carry(ctx context.Context, work ports.LinkWork, created *[]byte, current *channelState) bool {
	if work.Server != "" && !syncprotocol.IsSameAddress(work.Server, u.deps.Server) {
		u.reportOtherServer(ctx)
		return true
	}
	if work.WantsChannel && len(work.ChannelID) == 0 {
		return u.makeChannel(ctx, created)
	}
	if len(work.ChannelID) == 0 {
		return true
	}
	if !bytes.Equal(current.id, work.ChannelID) {
		stopWaits(current)
		*current = channelState{id: bytes.Clone(work.ChannelID), side: getSide(work.ChannelID, *created), isClosed: false, stopWaits: nil}
	}
	if !u.sendBlobs(ctx, current, work.Outbound) {
		return false
	}
	if work.IsClosed {
		return u.close(ctx, current)
	}
	u.startWaits(ctx, current)
	return true
}

func getSide(id, created []byte) ports.LinkSide {
	if bytes.Equal(id, created) {
		return ports.LinkSideDisplayer
	}
	return ports.LinkSideScanner
}

func getOtherSide(side ports.LinkSide) ports.LinkSide {
	if side == ports.LinkSideDisplayer {
		return ports.LinkSideScanner
	}
	return ports.LinkSideDisplayer
}

func (u *Courier) makeChannel(ctx context.Context, created *[]byte) bool {
	channel, err := u.deps.Relay.CreateChannel(ctx)
	if err != nil {
		u.report(ctx)
		return false
	}
	if err := u.deps.Core.SetLinkChannel(ctx, channel.ID); err != nil {
		return false
	}
	*created = bytes.Clone(channel.ID)
	u.clear(ctx)
	return true
}

func (u *Courier) sendBlobs(ctx context.Context, current *channelState, blobs [][]byte) bool {
	for _, blob := range blobs {
		_, err := u.deps.Relay.SendMessage(ctx, current.id, getOtherSide(current.side), blob)
		if errors.Is(err, syncprotocol.ErrChannelGone) {
			return true
		}
		if err != nil {
			u.report(ctx)
			return false
		}
	}
	return true
}

func (u *Courier) close(ctx context.Context, current *channelState) bool {
	stopWaits(current)
	if current.isClosed {
		return true
	}
	err := u.deps.Relay.CloseChannel(ctx, current.id)
	if err != nil && !errors.Is(err, syncprotocol.ErrChannelGone) {
		u.report(ctx)
		return false
	}
	current.isClosed = true
	return true
}

func (u *Courier) startWaits(ctx context.Context, current *channelState) {
	if current.stopWaits != nil || current.isClosed {
		return
	}
	waitCtx, stop := context.WithCancel(ctx)
	current.stopWaits = stop
	id, side := bytes.Clone(current.id), current.side
	var started sync.WaitGroup
	started.Add(1)
	go func() {
		started.Done()
		u.readInbound(waitCtx, id, side)
	}()
	started.Wait()
}

func (u *Courier) readInbound(ctx context.Context, id []byte, side ports.LinkSide) {
	var after uint32
	for ctx.Err() == nil {
		answer, err := u.deps.Relay.WaitMessages(ctx, id, side, after, inboundWait)
		if errors.Is(err, syncprotocol.ErrChannelGone) {
			return
		}
		if err != nil {
			if u.deps.Clock.Wait(ctx, retryAfter) != nil {
				return
			}
			continue
		}
		for _, message := range answer.Messages {
			after = max(after, message.Seq)
			if err := u.deps.Core.PutLinkInbound(ctx, message.Body); err != nil {
				return
			}
		}
		if answer.IsClosed {
			return
		}
	}
}

func (u *Courier) report(ctx context.Context) {
	item := health.Item{Key: healthKey, Severity: health.Warning, Text: healthText}
	if u.deps.Health.Report(ctx, item) == nil {
		u.isRaised = true
	}
}

func (u *Courier) reportOtherServer(ctx context.Context) {
	_ = u.deps.Health.Report(ctx, health.Item{
		Key: healthOtherKey, Severity: health.Warning, Text: healthOtherText,
		Detail: "Numen does not send a link to any server but the one this device is set to.",
	})
}

func (u *Courier) clear(ctx context.Context) {
	if u.isRaised && u.deps.Health.Clear(ctx, healthKey) == nil {
		u.isRaised = false
	}
}
