package synctest

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/dropdownload"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/fetchfile"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/lifecycle"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/syncvault"
)

// NewPhone makes a computer that is a phone: it restores the identity of the words, joins the account, signs in
// and lists the files that are downloaded on tap instead of downloading them.
func NewPhone(t *testing.T, server *Server, clock *Clock, vault string, words []string) *Computer {
	t.Helper()
	phone := NewSecondComputer(t, server, clock, vault, words)
	phone.LazyHeavy = true
	return phone
}

// FetchDeps are the dependencies of the download of an absent file.
func (c *Computer) FetchDeps() fetchfile.Deps {
	return fetchfile.Deps{Revisions: c.Cloud, Blobs: c.Cloud, Ranges: c.Cloud, Crypto: c.Account, Host: c.Host, State: c.State}
}

// DropDeps are the dependencies of the removal of a download.
func (c *Computer) DropDeps() dropdownload.Deps {
	return dropdownload.Deps{Host: c.Host, State: c.State}
}

// LifecycleDeps are the dependencies of the lifecycle: the loop over the vaults at the interval, and the stream
// the test gives.
func (c *Computer) LifecycleDeps(interval time.Duration, stream lifecycle.Stream, changed <-chan domain.VaultID) lifecycle.Deps {
	return lifecycle.Deps{
		Pass: c.NewSyncVault(), Loop: syncRunner{c: c, interval: interval, changed: changed}, Stream: stream,
		Host: c.Host, State: c.State, Clock: c.Clock,
	}
}

type syncRunner struct {
	c        *Computer
	interval time.Duration
	changed  <-chan domain.VaultID
}

func (r syncRunner) Execute(ctx context.Context) error {
	return syncvault.NewRunSync(r.c.RunDeps(r.interval, r.changed)).Execute(ctx)
}

// WriteFileOfSize writes a file of the size in bytes in the vault, as another window would.
func (c *Computer) WriteFileOfSize(id domain.VaultID, path string, size int) string {
	content := strings.Repeat("g", size)
	c.Vaults.With(id).File(path, content)
	return content
}

// ListAbsent returns the files of the vault listed as absent.
func (c *Computer) ListAbsent(t *testing.T, id domain.VaultID) []host.AbsentFile {
	t.Helper()
	files, err := c.Vaults.With(id).ListAbsentFiles(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// IsAbsent reports whether the vault lists the file as absent.
func (c *Computer) IsAbsent(t *testing.T, id domain.VaultID, path string) bool {
	t.Helper()
	for _, one := range c.ListAbsent(t, id) {
		if one.Path == path {
			return true
		}
	}
	return false
}
