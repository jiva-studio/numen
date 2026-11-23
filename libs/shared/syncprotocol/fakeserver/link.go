package fakeserver

import (
	"context"
	"crypto/rand"
	"errors"
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

func (c *linkChannel) wake() {
	close(c.changed)
	c.changed = make(chan struct{})
}

func (s *Server) getOpenChannel(id []byte) (*linkChannel, error) {
	found, ok := s.links[string(id)]
	if !ok || found.isClosed || !s.now().Before(found.expiresAt) {
		return nil, connect.NewError(connect.CodeNotFound, errChannelGone)
	}
	return found, nil
}

func (s *Server) CreateChannel(_ context.Context, _ *connect.Request[syncv1.CreateChannelRequest]) (*connect.Response[syncv1.CreateChannelResponse], error) {
	id := make([]byte, linkIDSize)
	if _, err := rand.Read(id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	expiresAt := s.now().Add(linkLife)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.links[string(id)] = &linkChannel{expiresAt: expiresAt, waiting: map[syncv1.LinkSide]bool{}, changed: make(chan struct{})}
	return connect.NewResponse(&syncv1.CreateChannelResponse{ChannelId: id, ExpiresAt: expiresAt.Unix()}), nil
}

func (s *Server) SendMessage(_ context.Context, req *connect.Request[syncv1.SendMessageRequest]) (*connect.Response[syncv1.SendMessageResponse], error) {
	body := req.Msg.GetBody()
	if len(body) == 0 || len(body) > syncprotocol.MaxLinkMessage {
		return nil, connect.NewError(connect.CodeInvalidArgument, errLinkBody)
	}
	if side := req.Msg.GetTo(); side != syncv1.LinkSide_LINK_SIDE_DISPLAYER && side != syncv1.LinkSide_LINK_SIDE_SCANNER {
		return nil, connect.NewError(connect.CodeInvalidArgument, errLinkSide)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	channel, err := s.getOpenChannel(req.Msg.GetChannelId())
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

func (s *Server) CloseChannel(_ context.Context, req *connect.Request[syncv1.CloseChannelRequest]) (*connect.Response[syncv1.CloseChannelResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	channel, err := s.getOpenChannel(req.Msg.GetChannelId())
	if err != nil {
		return nil, err
	}
	channel.isClosed = true
	channel.wake()
	return connect.NewResponse(&syncv1.CloseChannelResponse{}), nil
}

func (s *Server) WaitMessages(ctx context.Context, req *connect.Request[syncv1.WaitMessagesRequest]) (*connect.Response[syncv1.WaitMessagesResponse], error) {
	reader := req.Msg.GetReader()
	if reader != syncv1.LinkSide_LINK_SIDE_DISPLAYER && reader != syncv1.LinkSide_LINK_SIDE_SCANNER {
		return nil, connect.NewError(connect.CodeInvalidArgument, errLinkSide)
	}
	wait := time.Duration(min(req.Msg.GetWaitSeconds(), syncprotocol.LinkWaitMax)) * time.Second
	timeout := time.After(wait)
	if err := s.takeSide(req.Msg.GetChannelId(), reader); err != nil {
		return nil, err
	}
	defer s.freeSide(req.Msg.GetChannelId(), reader)
	for {
		response, changed, err := s.readMessages(req.Msg)
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

func (s *Server) takeSide(id []byte, side syncv1.LinkSide) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	channel, ok := s.links[string(id)]
	if !ok || !s.now().Before(channel.expiresAt) || (channel.isClosed && channel.isTold) {
		return connect.NewError(connect.CodeNotFound, errChannelGone)
	}
	if channel.waiting[side] {
		return connect.NewError(connect.CodeAborted, errLinkBusy)
	}
	channel.waiting[side] = true
	return nil
}

func (s *Server) freeSide(id []byte, side syncv1.LinkSide) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if channel, ok := s.links[string(id)]; ok {
		channel.waiting[side] = false
	}
}

func (s *Server) readMessages(req *syncv1.WaitMessagesRequest) (*syncv1.WaitMessagesResponse, <-chan struct{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	channel, ok := s.links[string(req.GetChannelId())]
	if !ok || !s.now().Before(channel.expiresAt) {
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
