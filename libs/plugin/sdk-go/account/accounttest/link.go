package accounttest

import (
	"context"
	"crypto/rand"
	"errors"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/shared/identity"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

const (
	channelSize = 16
	maxWait     = 20 * time.Second
)

type fakeBlob struct {
	version uint64
	body    []byte
}

// fakeLink is the link the host of the fake has in progress, as the courier sees it.
type fakeLink struct {
	version      uint64
	wantsChannel bool
	channel      []byte
	isClosed     bool
	outbound     []fakeBlob
	inbound      chan []byte
	changed      chan struct{}
	pending      *identity.Chain
}

func newFakeLink() *fakeLink {
	return &fakeLink{wantsChannel: true, inbound: make(chan []byte, 64), changed: make(chan struct{})}
}

func (l *fakeLink) bump() uint64 {
	l.version++
	close(l.changed)
	l.changed = make(chan struct{})
	return l.version
}

// ShowLink makes the host start a link in mode show, as the window of a host does.
func (f *Fake) ShowLink() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.link != nil && f.link.isClosed {
		f.link = nil
	}
	if f.link != nil {
		return errors.New("a link is already in progress")
	}
	f.link = newFakeLink()
	f.link.bump()
	return nil
}

// WaitLinkChannel blocks until the courier gave the link its channel, and returns the channel.
func (f *Fake) WaitLinkChannel(ctx context.Context) ([]byte, error) {
	for {
		f.mu.Lock()
		link := f.link
		if link == nil {
			f.mu.Unlock()
			return nil, errors.New("no link is in progress")
		}
		channel, changed := link.channel, link.changed
		f.mu.Unlock()
		if channel != nil {
			return channel, nil
		}
		select {
		case <-changed:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// QueueLinkOutbound makes the core of the host hand the blob to the courier as a message for the other side.
func (f *Fake) QueueLinkOutbound(blob []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.link == nil {
		return errors.New("no link is in progress")
	}
	f.link.outbound = append(f.link.outbound, fakeBlob{version: f.link.bump(), body: append([]byte(nil), blob...)})
	return nil
}

// WaitLinkInbound blocks until the courier passed a message of the other side, and returns it.
func (f *Fake) WaitLinkInbound(ctx context.Context) ([]byte, error) {
	f.mu.Lock()
	link := f.link
	f.mu.Unlock()
	if link == nil {
		return nil, errors.New("no link is in progress")
	}
	select {
	case blob := <-link.inbound:
		return blob, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// CloseLink makes the core end the link, so the next work says the channel is closed.
func (f *Fake) CloseLink() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.link == nil {
		return errors.New("no link is in progress")
	}
	f.link.isClosed = true
	f.link.bump()
	return nil
}

// AdvanceEpoch makes the host revoke a device that never signed in: the chain of the host grows by that
// device and its revoke, and the account keys of the host are the new ones. The server is not notified of it
// until the plugin publishes the chain.
func (f *Fake) AdvanceEpoch() error {
	stranger, err := identity.NewDeviceKey(rand.Reader)
	if err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.record == nil {
		return errors.New("the host has no identity")
	}
	now := clock.System{}.Now()
	added, err := f.record.Chain.AppendDevice(f.record.root, stranger.Public(), "Stranger", now)
	if err != nil {
		return err
	}
	f.record.Chain = added
	return f.revokeLocked(stranger.Public().Sign, now)
}

// RevokeDevice makes the host revoke the device with the signing key, which its chain lists since the plugin adopted the chain of
// the server. The account keys of the host are the new ones, and the server is not notified of it until the plugin publishes the chain.
func (f *Fake) RevokeDevice(deviceSignKey []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.record == nil {
		return errors.New("the host has no identity")
	}
	return f.revokeLocked(deviceSignKey, clock.System{}.Now())
}

func (f *Fake) revokeLocked(deviceSignKey []byte, now time.Time) error {
	revoked, keys, err := f.record.Chain.RevokeDevice(f.record.Device, f.record.keys, deviceSignKey, rand.Reader, now)
	if err != nil {
		return err
	}
	wrapped, ok := revoked.GetWrappedFor(f.record.Device.SignerKey())
	if !ok {
		return errors.New("the revoke holds no keys for the host")
	}
	f.record.Wrapped, f.record.Chain, f.record.keys = wrapped, revoked, keys
	if f.link == nil {
		f.link = newFakeLink()
		f.link.wantsChannel = false
		f.link.isClosed = true
	}
	f.link.pending = &revoked
	f.link.bump()
	return nil
}

func (s *service) GetLinkWork(ctx context.Context, req *connect.Request[v1.GetLinkWorkRequest]) (*connect.Response[v1.GetLinkWorkResponse], error) {
	wait := min(time.Duration(req.Msg.GetWaitSeconds())*time.Second, maxWait)
	timeout := time.After(wait)
	for {
		response, changed := s.fake.readWork(req.Msg.GetSinceVersion())
		if response.GetVersion() != req.Msg.GetSinceVersion() || wait <= 0 {
			return connect.NewResponse(response), nil
		}
		select {
		case <-changed:
		case <-timeout:
			response, _ = s.fake.readWork(req.Msg.GetSinceVersion())
			return connect.NewResponse(response), nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func (f *Fake) readWork(since uint64) (*v1.GetLinkWorkResponse, <-chan struct{}) {
	f.mu.Lock()
	defer f.mu.Unlock()
	link := f.link
	if link == nil {
		return &v1.GetLinkWorkResponse{}, make(chan struct{})
	}
	kept := link.outbound[:0]
	for _, blob := range link.outbound {
		if blob.version > since {
			kept = append(kept, blob)
		}
	}
	link.outbound = kept
	response := &v1.GetLinkWorkResponse{
		Version: link.version, IsActive: !link.isClosed || len(kept) > 0, WantsChannel: link.wantsChannel,
		ChannelId: link.channel, IsClosed: link.isClosed,
	}
	for _, blob := range kept {
		response.Outbound = append(response.Outbound, blob.body)
	}
	return response, link.changed
}

func (s *service) SetLinkChannel(_ context.Context, req *connect.Request[v1.SetLinkChannelRequest]) (*connect.Response[v1.SetLinkChannelResponse], error) {
	f := s.fake
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.link == nil || !f.link.wantsChannel || len(req.Msg.GetChannelId()) != channelSize {
		return nil, refuse(connect.CodeFailedPrecondition, "NO_LINK")
	}
	f.link.channel = append([]byte(nil), req.Msg.GetChannelId()...)
	f.link.wantsChannel = false
	f.link.bump()
	return connect.NewResponse(&v1.SetLinkChannelResponse{}), nil
}

func (s *service) PutLinkInbound(_ context.Context, req *connect.Request[v1.PutLinkInboundRequest]) (*connect.Response[v1.PutLinkInboundResponse], error) {
	f := s.fake
	f.mu.Lock()
	link := f.link
	f.mu.Unlock()
	if link == nil || link.isClosed {
		return nil, refuse(connect.CodeFailedPrecondition, "NO_LINK")
	}
	select {
	case link.inbound <- append([]byte(nil), req.Msg.GetBlob()...):
	default:
	}
	return connect.NewResponse(&v1.PutLinkInboundResponse{}), nil
}

func (s *service) GetChainPublish(context.Context, *connect.Request[v1.GetChainPublishRequest]) (*connect.Response[v1.GetChainPublishResponse], error) {
	f := s.fake
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.link == nil || f.link.pending == nil {
		return connect.NewResponse(&v1.GetChainPublishResponse{}), nil
	}
	return connect.NewResponse(&v1.GetChainPublishResponse{IsPending: true, Chain: f.link.pending.Bytes()}), nil
}

func (s *service) PutChainPublished(_ context.Context, req *connect.Request[v1.PutChainPublishedRequest]) (*connect.Response[v1.PutChainPublishedResponse], error) {
	f := s.fake
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.link == nil || f.link.pending == nil {
		return nil, refuse(connect.CodeFailedPrecondition, "NO_LINK")
	}
	f.link.pending = nil
	f.link.bump()
	return connect.NewResponse(&v1.PutChainPublishedResponse{}), nil
}

func (s *service) AdoptChain(_ context.Context, req *connect.Request[v1.AdoptChainRequest]) (*connect.Response[v1.AdoptChainResponse], error) {
	f := s.fake
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.record == nil {
		return nil, refuse(connect.CodeFailedPrecondition, "NO_IDENTITY")
	}
	chain, err := identity.ParseChain(req.Msg.GetChain())
	if err != nil || !chain.IsExtensionOf(f.record.Chain) {
		return nil, refuse(connect.CodeInvalidArgument, "INVALID_CHAIN")
	}
	f.record.Chain = chain
	return connect.NewResponse(&v1.AdoptChainResponse{}), nil
}
