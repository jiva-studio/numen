package accounttest

import (
	"context"
	"errors"
)

// ShowLink makes the host start a link in mode show, as the window of a host does.
func (f *Fake) ShowLink() error { return errors.New("not implemented") }

// WaitLinkChannel blocks until the courier gave the link its channel, and returns the channel.
func (f *Fake) WaitLinkChannel(ctx context.Context) ([]byte, error) {
	return nil, errors.New("not implemented")
}

// QueueLinkOutbound makes the core of the host hand the blob to the courier as a message for the other side.
func (f *Fake) QueueLinkOutbound(blob []byte) error { return errors.New("not implemented") }

// WaitLinkInbound blocks until the courier passed a message of the other side, and returns it.
func (f *Fake) WaitLinkInbound(ctx context.Context) ([]byte, error) {
	return nil, errors.New("not implemented")
}

// CloseLink makes the core end the link, so the next work says the channel is closed.
func (f *Fake) CloseLink() error { return errors.New("not implemented") }

// AdvanceEpoch makes the host revoke a device that never signed in: the chain of the host grows by that
// device and its revoke, and the account keys of the host are the new ones. The server does not hear of it
// until the plugin publishes the chain.
func (f *Fake) AdvanceEpoch() error { return errors.New("not implemented") }
