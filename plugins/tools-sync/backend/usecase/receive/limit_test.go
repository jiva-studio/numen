package receive_test

import (
	"context"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/receive"
)

// limitedShares is the share service of a server that limits the downloads of the share.
type limitedShares struct{ ports.Shares }

func (limitedShares) GetShareBlobUrls(_ context.Context, _ []byte, _ [][]byte) ([]*syncv1.BlobDownload, error) {
	return nil, syncprotocol.RateLimitedError{RetryAfter: time.Minute}
}

func TestPullShare_StopsOnDownloadLimitAndPullsNoteOnNextPass(t *testing.T) {
	w := newWorld(t)
	w.owner.WriteNote(owned, "Food/Soup.md", "carrots")
	w.syncOwner(t)
	received := w.receive(t, w.reader, w.shareWith(t, w.reader, "Anna"))
	w.owner.WriteNote(owned, "Food/Bread.md", "flour")
	w.syncOwner(t)
	limited := w.reader.ReceiveDeps()
	limited.Shares = limitedShares{Shares: limited.Shares}

	result, err := receive.NewPullShare(limited).Execute(t.Context(), received.VaultID)

	if err != nil {
		t.Fatalf("a limited download is no failure of the pass: %v", err)
	}
	if len(result.Applied) != 0 {
		t.Errorf("applied %v while the server limited the downloads, want nothing", result.Applied)
	}
	wantNoNote(t, w.reader, received.VaultID, "Bread.md")
	if err := w.reader.Sync(t.Context(), received.VaultID); err != nil {
		t.Fatal(err)
	}
	wantNote(t, w.reader, received.VaultID, "Bread.md", "flour")
}
