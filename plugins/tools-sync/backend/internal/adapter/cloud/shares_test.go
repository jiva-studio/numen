package cloud_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/tools/syncserver/fakeserver"
)

func TestCloud_SignsInAgainOnceForEachCallOfTheShareService(t *testing.T) {
	c, _ := newWorld(t, fakeserver.Config{})
	shareID := newVaultID(t)
	var refreshed int
	stale := newCloud(c, "not a session", func(context.Context) error {
		refreshed++
		return nil
	})

	_, listErr := stale.ListShares(t.Context())
	_, _, logErr := stale.GetShareLog(t.Context(), shareID, 0)
	_, changesErr := stale.GetShareChanges(t.Context(), shareID, 0, 0)
	_, urlsErr := stale.GetShareBlobUrls(t.Context(), shareID, [][]byte{newVaultID(t)})

	for name, err := range map[string]error{"ListShares": listErr, "GetShareLog": logErr, "GetShareChanges": changesErr, "GetShareBlobUrls": urlsErr} {
		if !errors.Is(err, syncprotocol.ErrUnauthenticated) {
			t.Errorf("%s: error = %v, want ErrUnauthenticated", name, err)
		}
	}
	if refreshed != 4 {
		t.Errorf("sign-ins asked for = %d, want one for each of the four calls", refreshed)
	}
}

func TestCloud_ReadsNoShareOfAnAccountThatSharesNothing(t *testing.T) {
	c, _ := newWorld(t, fakeserver.Config{})

	shares, err := c.Cloud.ListShares(t.Context())

	if err != nil || len(shares) != 0 {
		t.Errorf("shares = %v, error = %v, want none", shares, err)
	}
	if _, _, err := c.Cloud.GetShareLog(t.Context(), newVaultID(t), 0); !errors.Is(err, syncprotocol.ErrNoShare) {
		t.Errorf("a share that does not exist: error = %v, want ErrNoShare", err)
	}
}
