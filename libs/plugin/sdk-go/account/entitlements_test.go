package account_test

import (
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/account"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/account/accounttest"
)

func TestGetEntitlements_ReturnsTheProductsTheHostHolds(t *testing.T) {
	fake := accounttest.NewFake(t, pluginID)
	client := newClient(t, fake)
	linkEnd := time.Unix(4_900_000_000, 0)
	fake.SetEntitlements([]account.Entitlement{
		{Product: "link", ValidUntil: linkEnd, Phase: "active"},
		{Product: "sync", ValidUntil: time.Unix(1_800_000_000, 0), Phase: "grace", QuotaBytes: 100 << 30, IsTrial: true},
	})

	got, err := client.GetEntitlements(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 2 || got[0].Product != "link" || !got[0].ValidUntil.Equal(linkEnd) || got[0].Phase != "active" || got[0].QuotaBytes != 0 || got[0].IsTrial {
		t.Errorf("entitlements = %+v, want link first, active, valid until %s, no quota", got, linkEnd)
	}
	if got[1].Product != "sync" || got[1].Phase != "grace" || got[1].QuotaBytes != 100<<30 || !got[1].IsTrial {
		t.Errorf("second entitlement = %+v, want sync in grace with 100 GiB and a trial", got[1])
	}
}

func TestGetEntitlements_ReturnsNoProductForAnAccountThatOwnsNone(t *testing.T) {
	fake := accounttest.NewFake(t, pluginID)
	client := newClient(t, fake)

	got, err := client.GetEntitlements(t.Context())

	if err != nil || len(got) != 0 {
		t.Errorf("entitlements = %+v, error = %v, want none and no error", got, err)
	}
}

func TestGetEntitlements_ReturnsErrNoIdentityWhereTheDeviceHoldsNone(t *testing.T) {
	fake := accounttest.NewFake(t, pluginID)
	fake.NoIdentity()
	client := newClient(t, fake)

	_, err := client.GetEntitlements(t.Context())

	if !errors.Is(err, account.ErrNoIdentity) {
		t.Errorf("error = %v, want ErrNoIdentity", err)
	}
}

func TestGetEntitlements_ReturnsErrSyncUnavailableWhereTheHostCannotReachTheServer(t *testing.T) {
	fake := accounttest.NewFake(t, pluginID)
	client := newClient(t, fake)
	fake.FailEntitlements(true)

	_, err := client.GetEntitlements(t.Context())

	if !errors.Is(err, account.ErrSyncUnavailable) {
		t.Errorf("error = %v, want ErrSyncUnavailable", err)
	}
	fake.FailEntitlements(false)
	if _, err := client.GetEntitlements(t.Context()); err != nil {
		t.Errorf("error after the failure ended = %v, want none", err)
	}
}
