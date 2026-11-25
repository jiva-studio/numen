package synctest

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/cloud"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/dropdownload"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/fetchfile"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/lifecycle"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/syncvault"
	"github.com/jiva-studio/numen/tools/syncserver/fakeserver"
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
	return fetchfile.Deps{
		Stage: c.WireDeps().Downloads, Revisions: c.Cloud, Blobs: cloud.NewBlobs(c.Cloud), Crypto: c.Account, Host: c.Host, State: c.State,
		Shares: share.NewReadShares(c.ShareDeps()), Identity: c.Account, Chain: c.Cloud,
	}
}

// DropDeps are the dependencies of the removal of a download.
func (c *Computer) DropDeps() dropdownload.Deps {
	return dropdownload.Deps{Host: c.Host, State: c.State}
}

// GetLifecycleDeps are the dependencies of the lifecycle: the loop over the vaults at the interval, and the stream
// the test gives.
func (c *Computer) GetLifecycleDeps(interval time.Duration, stream lifecycle.Stream, changed <-chan domain.VaultID) lifecycle.Deps {
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
	c.Vaults.With(id).WriteFile(path, content)
	return content
}

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

const (
	// ComputerVault and PhoneVault are the names of the two vaults of a PhoneWorld.
	ComputerVault = "Test"
	PhoneVault    = "Phone"
)

// PhoneWorld is one account on a server with a computer and a phone, the phone connected to the synced vault of
// the computer, as the background of the phone features has it.
type PhoneWorld struct {
	Server   *Server
	Clock    *Clock
	Computer *Computer
	Phone    *Computer
	fetch    *fetchfile.FetchFile
}

// NewPhoneWorld makes the world. No pass has run on the phone.
func NewPhoneWorld(t *testing.T) *PhoneWorld {
	t.Helper()
	clock := NewClock(GetStart())
	server := NewServer(t, clock, fakeserver.Config{})
	words := NewWords(t)
	computer := NewFirstComputer(t, server, clock, ComputerVault, words)
	computer.Connect(t, ComputerVault)
	phone := NewPhone(t, server, clock, PhoneVault, words)
	phone.ConnectExisting(t, PhoneVault, computer.Link(t, ComputerVault).RemoteVaultID)
	return &PhoneWorld{
		Server: server, Clock: clock, Computer: computer, Phone: phone, fetch: fetchfile.NewFetchFile(phone.FetchDeps()),
	}
}

// SyncComputer runs a pass of the computer.
func (w *PhoneWorld) SyncComputer(t *testing.T) {
	t.Helper()
	if err := w.Computer.Sync(t.Context(), ComputerVault); err != nil {
		t.Fatal(err)
	}
}

// SyncPhone runs a pass of the phone.
func (w *PhoneWorld) SyncPhone(t *testing.T) {
	t.Helper()
	if err := w.Phone.Sync(t.Context(), PhoneVault); err != nil {
		t.Fatal(err)
	}
}

// Fetch downloads the absent file of the phone.
func (w *PhoneWorld) Fetch(ctx context.Context, path string) error {
	return w.fetch.Execute(ctx, PhoneVault, path)
}

func (w *PhoneWorld) DropDownload(ctx context.Context, path string) error {
	return dropdownload.NewDropDownload(w.Phone.DropDeps()).Execute(ctx, PhoneVault, path)
}

// WriteGuide writes the file of 5 MiB on the computer and sends it, and the phone pass lists it.
func (w *PhoneWorld) WriteGuide(t *testing.T, path string) string {
	t.Helper()
	content := w.Computer.WriteFileOfSize(ComputerVault, path, 5<<20)
	w.Clock.Advance(time.Minute)
	w.SyncComputer(t)
	w.SyncPhone(t)
	return content
}

// Stream is the stream of hints of the server, the network edge of a phone: it counts the streams that are open
// and lets a test send a hint down them.
type Stream struct {
	open   atomic.Int32
	opened atomic.Int32
	mu     sync.Mutex
	hints  []func(domain.VaultID)
}

// Watch holds a stream open until ctx ends.
func (s *Stream) Watch(ctx context.Context, changed func(domain.VaultID)) error {
	s.open.Add(1)
	s.opened.Add(1)
	defer s.open.Add(-1)
	s.mu.Lock()
	s.hints = append(s.hints, changed)
	s.mu.Unlock()
	<-ctx.Done()
	return ctx.Err()
}

// CountOpen returns how many streams are open now.
func (s *Stream) CountOpen() int { return int(s.open.Load()) }

// CountOpened returns how many streams were opened in all.
func (s *Stream) CountOpened() int { return int(s.opened.Load()) }

// SendHint tells every open stream that the vault changed on the server.
func (s *Stream) SendHint(id domain.VaultID) {
	s.mu.Lock()
	calls := append([]func(domain.VaultID){}, s.hints...)
	s.mu.Unlock()
	for _, call := range calls {
		call(id)
	}
}
