// Package linkrelay is the relay of the link service: it holds the sealed messages of both sides of a channel and reads none of them.
package linkrelay

import (
	"context"
	"crypto/rand"
	"errors"
	"sync"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

const (
	linkLife        = 10 * time.Minute
	linkIDSize      = 16
	linkMaxMessages = 24
)

var (
	errChannelGone = errors.New("the link channel is not there")
	errLinkBusy    = errors.New("another wait runs on this side of the channel")
	errLinkSide    = errors.New("a link call names the side it posts to or reads for")
	errLinkBody    = errors.New("a link message has 1 to 262144 bytes")
)

type linkMessage struct {
	seq  uint32
	to   syncv1.LinkSide
	body []byte
}

// linkChannel is a channel of the relay: it holds the sealed messages of both sides and reads none of them.
type linkChannel struct {
	expiresAt  time.Time
	isClosed   bool
	isTold     bool
	messages   []linkMessage
	waiting    map[syncv1.LinkSide]bool
	changed    chan struct{}
	lastNumber uint32
}

// Relay serves the channels of the link service.
type Relay struct {
	now      func() time.Time
	mu       sync.Mutex
	channels map[string]*linkChannel
}

// New returns a relay that reads the time from now.
func New(now func() time.Time) *Relay {
	return &Relay{now: now, channels: map[string]*linkChannel{}}
}

func (c *linkChannel) wake() {
	close(c.changed)
	c.changed = make(chan struct{})
}

func (r *Relay) getOpenChannel(id []byte) (*linkChannel, error) {
	found, ok := r.channels[string(id)]
	if !ok || found.isClosed || !r.now().Before(found.expiresAt) {
		return nil, connect.NewError(connect.CodeNotFound, errChannelGone)
	}
	return found, nil
}

// CreateChannel serves the call of the same name of the link service.
func (r *Relay) CreateChannel(_ context.Context, _ *connect.Request[syncv1.CreateChannelRequest]) (*connect.Response[syncv1.CreateChannelResponse], error) {
	id := make([]byte, linkIDSize)
	if _, err := rand.Read(id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	expiresAt := r.now().Add(linkLife)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.channels[string(id)] = &linkChannel{expiresAt: expiresAt, waiting: map[syncv1.LinkSide]bool{}, changed: make(chan struct{})}
	return connect.NewResponse(&syncv1.CreateChannelResponse{ChannelId: id, ExpiresAt: expiresAt.Unix()}), nil
}

// SendMessage serves the call of the same name of the link service.
func (r *Relay) SendMessage(_ context.Context, req *connect.Request[syncv1.SendMessageRequest]) (*connect.Response[syncv1.SendMessageResponse], error) {
	body := req.Msg.GetBody()
	if len(body) == 0 || len(body) > syncprotocol.MaxLinkMessage {
		return nil, connect.NewError(connect.CodeInvalidArgument, errLinkBody)
	}
	if side := req.Msg.GetTo(); side != syncv1.LinkSide_LINK_SIDE_DISPLAYER && side != syncv1.LinkSide_LINK_SIDE_SCANNER {
		return nil, connect.NewError(connect.CodeInvalidArgument, errLinkSide)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	channel, err := r.getOpenChannel(req.Msg.GetChannelId())
	if err != nil {
		return nil, err
	}
	channel.lastNumber++
	channel.messages = append(channel.messages, linkMessage{seq: channel.lastNumber, to: req.Msg.GetTo(), body: append([]byte(nil), body...)})
	if channel.lastNumber >= linkMaxMessages {
		channel.isClosed = true
	}
	channel.wake()
	return connect.NewResponse(&syncv1.SendMessageResponse{Seq: channel.lastNumber}), nil
}

// CloseChannel serves the call of the same name of the link service.
func (r *Relay) CloseChannel(_ context.Context, req *connect.Request[syncv1.CloseChannelRequest]) (*connect.Response[syncv1.CloseChannelResponse], error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	channel, err := r.getOpenChannel(req.Msg.GetChannelId())
	if err != nil {
		return nil, err
	}
	channel.isClosed = true
	channel.wake()
	return connect.NewResponse(&syncv1.CloseChannelResponse{}), nil
}

// WaitMessages serves the call of the same name of the link service.
func (r *Relay) WaitMessages(ctx context.Context, req *connect.Request[syncv1.WaitMessagesRequest]) (*connect.Response[syncv1.WaitMessagesResponse], error) {
	reader := req.Msg.GetReader()
	if reader != syncv1.LinkSide_LINK_SIDE_DISPLAYER && reader != syncv1.LinkSide_LINK_SIDE_SCANNER {
		return nil, connect.NewError(connect.CodeInvalidArgument, errLinkSide)
	}
	wait := time.Duration(min(req.Msg.GetWaitSeconds(), syncprotocol.LinkWaitMax)) * time.Second
	timeout := time.After(wait)
	if err := r.takeSide(req.Msg.GetChannelId(), reader); err != nil {
		return nil, err
	}
	defer r.freeSide(req.Msg.GetChannelId(), reader)
	for {
		response, changed, err := r.readMessages(req.Msg)
		if err != nil || len(response.GetMessages()) > 0 || response.GetIsClosed() || wait <= 0 {
			return connect.NewResponse(response), err
		}
		select {
		case <-changed:
		case <-timeout:
			return connect.NewResponse(response), nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func (r *Relay) takeSide(id []byte, side syncv1.LinkSide) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	channel, ok := r.channels[string(id)]
	if !ok || !r.now().Before(channel.expiresAt) || (channel.isClosed && channel.isTold) {
		return connect.NewError(connect.CodeNotFound, errChannelGone)
	}
	if channel.waiting[side] {
		return connect.NewError(connect.CodeAborted, errLinkBusy)
	}
	channel.waiting[side] = true
	return nil
}

func (r *Relay) freeSide(id []byte, side syncv1.LinkSide) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if channel, ok := r.channels[string(id)]; ok {
		channel.waiting[side] = false
	}
}

func (r *Relay) readMessages(req *syncv1.WaitMessagesRequest) (*syncv1.WaitMessagesResponse, <-chan struct{}, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	channel, ok := r.channels[string(req.GetChannelId())]
	if !ok || !r.now().Before(channel.expiresAt) {
		return nil, nil, connect.NewError(connect.CodeNotFound, errChannelGone)
	}
	response := &syncv1.WaitMessagesResponse{}
	for _, message := range channel.messages {
		if message.to == req.GetReader() && message.seq > req.GetAfterSeq() {
			response.Messages = append(response.Messages, &syncv1.LinkMessage{Seq: message.seq, Body: message.body})
		}
	}
	if channel.isClosed {
		response.IsClosed = true
		channel.isTold = true
	}
	return response, channel.changed, nil
}
