package account_test

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/account"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/account/accounttest"
)

func TestGetPublicIdentity_ReportsTheEpochTheChainLengthAndThatTheDeviceIsNotRevoked(t *testing.T) {
	fake := accounttest.NewFake(t, pluginID)

	got, err := newClient(t, fake).GetPublicIdentity(t.Context())

	if err != nil || got.KeyEpoch != 0 || got.ChainLength != 2 || got.IsRevoked {
		t.Errorf("identity = %+v, %v, want epoch 0, 2 entries, not revoked", got, err)
	}
}

func TestGetLinkWork_ReturnsNoWorkWhenNoLinkWasStarted(t *testing.T) {
	fake := accounttest.NewFake(t, pluginID)

	work, err := newClient(t, fake).GetLinkWork(t.Context(), 0, 0)

	if err != nil || work.IsActive || work.WantsChannel || len(work.Outbound) != 0 {
		t.Errorf("work = %+v, %v, want an inactive answer", work, err)
	}
}

func TestGetLinkWork_AsksForAChannelWhenTheHostShowsAQRCode(t *testing.T) {
	fake := accounttest.NewFake(t, pluginID)
	if err := fake.ShowLink(); err != nil {
		t.Fatal(err)
	}
	client := newClient(t, fake)

	work, err := client.GetLinkWork(t.Context(), 0, 0)
	if err != nil || !work.IsActive || !work.WantsChannel {
		t.Fatalf("work = %+v, %v, want an active link that wants a channel", work, err)
	}
	if err := client.SetLinkChannel(t.Context(), []byte("0123456789abcdef")); err != nil {
		t.Fatal(err)
	}
	next, err := client.GetLinkWork(t.Context(), work.Version, 0)
	if err != nil || next.WantsChannel || !bytes.Equal(next.ChannelID, []byte("0123456789abcdef")) {
		t.Errorf("work = %+v, %v, want the channel the courier set", next, err)
	}
}

func TestGetLinkWork_ReturnsAtOnceWhenTheVersionDiffers(t *testing.T) {
	fake := accounttest.NewFake(t, pluginID)
	if err := fake.ShowLink(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	work, err := newClient(t, fake).GetLinkWork(ctx, 12345, 20*time.Second)

	if err != nil || !work.IsActive {
		t.Errorf("work = %+v, %v, want an answer within 5 s", work, err)
	}
}

func TestLinkInbound_RejectsBlobWhenNoLinkIsActive(t *testing.T) {
	fake := accounttest.NewFake(t, pluginID)

	err := newClient(t, fake).PutLinkInbound(t.Context(), []byte("blob"))

	if !errors.Is(err, account.ErrNoLink) {
		t.Errorf("error = %v, want ErrNoLink", err)
	}
}

func TestAdoptChain_RefusesBytesThatAreNotAChain(t *testing.T) {
	fake := accounttest.NewFake(t, pluginID)

	err := newClient(t, fake).AdoptChain(t.Context(), []byte("not a chain"))

	if !errors.Is(err, account.ErrInvalidChain) {
		t.Errorf("error = %v, want ErrInvalidChain", err)
	}
}

func TestGetChainPublish_HasNothingPendingBeforeALinkOrRevoke(t *testing.T) {
	fake := accounttest.NewFake(t, pluginID)

	pending, err := newClient(t, fake).GetChainPublish(t.Context())

	if err != nil || pending.IsPending {
		t.Errorf("pending = %+v, %v, want nothing pending", pending, err)
	}
}
