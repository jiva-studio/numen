package vault_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"slices"
	"strings"
	"testing"
	"time"

	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// scanFile is one entry a scanVault walks: what the walk reports, and what a read
// of it answers.
type scanFile struct {
	kind    domain.SourceKind
	size    int64
	modTime time.Time
	readErr error
}

// scanVault is a vault held in memory, answering to a scan as a reader and, for
// the rows a scan writes, as every repository it asks.
type scanVault struct {
	files   map[string]scanFile
	openErr error
	walkErr error

	indexedVal     map[string]domain.Fingerprint
	knownErr       error
	registerErr    error
	saveErr        error
	failSaveAt     int
	removeErr      error
	reportErr      error
	opens          int
	reads          []string
	saved          [][]string
	removed        []string
	reports        int
	saveCalls      int
	registerCalled bool
}

func (m *scanVault) Open(context.Context, string) (io.ReadSeekCloser, error) { return nil, nil }

func (m *scanVault) WalkFiles(ctx context.Context, fn func(domain.Fingerprint) error) error {
	if m.walkErr != nil {
		return m.walkErr
	}
	for path, f := range m.files {
		if err := fn(domain.Fingerprint{Path: path, Kind: f.kind, Size: f.size, ModTime: f.modTime}); err != nil {
			return err
		}
	}
	return nil
}

func (m *scanVault) Read(_ context.Context, path string) ([]byte, error) {
	m.reads = append(m.reads, path)
	if err := m.files[path].readErr; err != nil {
		return nil, err
	}
	return []byte("# " + path), nil
}

func (m *scanVault) List(context.Context, string) ([]domain.Entry, error) { return nil, nil }

func (m *scanVault) GetFingerprint(context.Context, string) (domain.Fingerprint, error) {
	return domain.Fingerprint{}, nil
}

func (m *scanVault) Register(context.Context, domain.VaultID) error {
	m.registerCalled = true
	return m.registerErr
}

func (m *scanVault) Unregister(context.Context, domain.VaultID) error { return nil }

func (m *scanVault) GetFingerprints(context.Context, domain.VaultID) (map[string]domain.Fingerprint, error) {
	return m.indexedVal, m.knownErr
}

func (m *scanVault) Save(_ context.Context, _ domain.VaultID, notes []domain.NoteChunks) error {
	m.saveCalls++
	if m.saveErr != nil && m.saveCalls >= m.failSaveAt {
		return m.saveErr
	}
	paths := make([]string, len(notes))
	for i, n := range notes {
		paths[i] = n.Note.Fingerprint.Path
	}
	m.saved = append(m.saved, paths)
	return nil
}

func (m *scanVault) Remove(_ context.Context, _ domain.VaultID, paths []string) error {
	m.removed = append(m.removed, paths...)
	return m.removeErr
}

func (m *scanVault) ReportChanges(context.Context) error {
	m.reports++
	return m.reportErr
}

func (m *scanVault) scan() vaults.Scan {
	return vaults.NewScan(scanReaders{m}, m, m, m, m)
}

// scanReaders is the readers of a scan, answering for the one scanVault.
type scanReaders struct{ vault *scanVault }

func (r scanReaders) Open(domain.Vault) (vaultport.VaultReader, error) {
	r.vault.opens++
	if r.vault.openErr != nil {
		return nil, r.vault.openErr
	}
	return r.vault, nil
}

func noteFile(modTime time.Time) scanFile {
	return scanFile{kind: domain.KindNote, size: 10, modTime: modTime}
}

var epoch = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

func TestAScanOfACancelledContextTouchesNothing(t *testing.T) {
	t.Parallel()
	m := &scanVault{files: map[string]scanFile{"a.md": noteFile(epoch)}}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	res, err := m.scan().Execute(ctx, domain.Vault{ID: "v", Name: "v"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want the cancellation", err)
	}
	if m.opens != 0 || m.registerCalled || res.Notes != 0 {
		t.Errorf("a cancelled scan went on: opens=%d registered=%v result=%+v", m.opens, m.registerCalled, res)
	}
}

func TestAScanStopsAtTheFirstStepThatFails(t *testing.T) {
	t.Parallel()
	boom := errors.New("boom")
	cases := []struct {
		name    string
		set     func(*scanVault)
		message string
		reached func(*scanVault) bool
		indexed int
	}{
		{"the vault cannot be opened", func(m *scanVault) { m.openErr = boom }, "boom", func(m *scanVault) bool { return m.registerCalled }, 0},
		{"the vault cannot be registered", func(m *scanVault) { m.registerErr = boom }, "register vault: boom", func(m *scanVault) bool { return len(m.reads) > 0 }, 0},
		{"the index cannot be read", func(m *scanVault) { m.knownErr = boom }, "read index: boom", func(m *scanVault) bool { return len(m.reads) > 0 }, 0},
		{"the walk fails", func(m *scanVault) { m.walkErr = boom }, "boom", func(m *scanVault) bool { return len(m.reads) > 0 }, 0},
		{"the vault cannot be told which notes are gone", func(m *scanVault) { m.removeErr = boom }, "remove deleted notes: boom", func(m *scanVault) bool { return m.reports > 0 }, 1},
		{"the index upkeep fails", func(m *scanVault) { m.reportErr = boom }, "index upkeep for named: boom", func(*scanVault) bool { return false }, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := &scanVault{files: map[string]scanFile{"a.md": noteFile(epoch)}}
			tc.set(m)

			res, err := m.scan().Execute(t.Context(), domain.Vault{ID: "v", Name: "named"})
			if res.Indexed != tc.indexed {
				t.Errorf("result after %s counted %d indexed, want %d", tc.name, res.Indexed, tc.indexed)
			}
			if !errors.Is(err, boom) || err.Error() != tc.message {
				t.Errorf("error = %v, want %q wrapping the cause", err, tc.message)
			}
			if tc.reached(m) {
				t.Errorf("the scan went on after %s", tc.name)
			}
		})
	}
}

func TestAFailedGroupWriteKeepsWhatEarlierGroupsCounted(t *testing.T) {
	t.Parallel()
	const perGroup = 500
	m := &scanVault{files: map[string]scanFile{}, saveErr: errors.New("disk full"), failSaveAt: 2}
	for i := range perGroup + 1 {
		m.files[fmt.Sprintf("n%04d.md", i)] = noteFile(epoch.Add(time.Duration(i) * time.Second))
	}

	res, err := m.scan().Execute(t.Context(), domain.Vault{ID: "v", Name: "v"})
	if err == nil {
		t.Fatal("a scan whose last group could not be written reported success")
	}
	if res.Indexed != perGroup {
		t.Errorf("counted %d notes as indexed, want the %d of the group that was written", res.Indexed, perGroup)
	}
	if m.reports != 0 {
		t.Error("the index was measured after a scan that failed")
	}
}

func TestAFailedWriteNamesTheFirstAndLastNoteOfTheGroup(t *testing.T) {
	t.Parallel()
	m := &scanVault{
		files: map[string]scanFile{
			"a.md": noteFile(epoch.Add(3 * time.Second)),
			"b.md": noteFile(epoch.Add(2 * time.Second)),
			"c.md": noteFile(epoch.Add(1 * time.Second)),
		},
		saveErr: errors.New("disk full"), failSaveAt: 1,
	}

	_, err := m.scan().Execute(t.Context(), domain.Vault{ID: "v", Name: "named"})
	if err == nil || !strings.Contains(err.Error(), "index 3 notes of named, a.md to c.md: disk full") {
		t.Errorf("error = %v, want the group named by its first and last note", err)
	}
}

func TestAScanGoesOnPastWhatItCannotIndex(t *testing.T) {
	t.Parallel()
	newest := epoch.Add(time.Hour)
	cases := []struct {
		name  string
		first scanFile
		want  func(vaults.ScanResult) bool
	}{
		{"a source of another kind", scanFile{kind: domain.KindBook, size: 1, modTime: newest}, func(r vaults.ScanResult) bool { return r.Assets == 1 }},
		{"a file that vanished", scanFile{kind: domain.KindNote, size: 1, modTime: newest, readErr: fs.ErrNotExist}, func(r vaults.ScanResult) bool { return r.Vanished == 1 }},
		{"a file nobody can read", scanFile{kind: domain.KindNote, size: 1, modTime: newest, readErr: errors.New("denied")}, func(r vaults.ScanResult) bool { return r.Unreadable == 1 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := &scanVault{files: map[string]scanFile{
				"first.md":  tc.first,
				"second.md": noteFile(epoch.Add(2 * time.Minute)),
				"third.md":  noteFile(epoch.Add(time.Minute)),
			}}

			res, err := m.scan().Execute(t.Context(), domain.Vault{ID: "v", Name: "v"})
			if err != nil {
				t.Fatal(err)
			}
			if !tc.want(res) {
				t.Errorf("the first file went uncounted: %+v", res)
			}
			if !slices.Equal(slices.Concat(m.saved...), []string{"second.md", "third.md"}) {
				t.Errorf("wrote %v after %s, want the two notes behind it", m.saved, tc.name)
			}
		})
	}
}

func TestARebuildReadsWhatTheIndexRemembers(t *testing.T) {
	t.Parallel()
	m := &scanVault{
		files:      map[string]scanFile{"a.md": noteFile(epoch)},
		indexedVal: map[string]domain.Fingerprint{"a.md": {Path: "a.md", Size: 10, ModTime: epoch}},
	}

	scan := m.scan()
	res, err := scan.Execute(t.Context(), domain.Vault{ID: "v", Name: "v"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Unchanged != 1 || res.Indexed != 0 {
		t.Fatalf("an unchanged file was not skipped: %+v", res)
	}

	scan.ShouldRebuildIndex = true
	res, err = scan.Execute(t.Context(), domain.Vault{ID: "v", Name: "v"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Unchanged != 0 || res.Indexed != 1 {
		t.Errorf("a rebuild skipped a file the index remembers: %+v", res)
	}
}

func TestTheIndexIsMeasuredAfterTheSmallestChange(t *testing.T) {
	t.Parallel()
	t.Run("one note indexed", func(t *testing.T) {
		t.Parallel()
		m := &scanVault{files: map[string]scanFile{"a.md": noteFile(epoch)}}
		if _, err := m.scan().Execute(t.Context(), domain.Vault{ID: "v", Name: "v"}); err != nil {
			t.Fatal(err)
		}
		if m.reports != 1 {
			t.Errorf("measured %d times after one note was indexed, want 1", m.reports)
		}
	})
	t.Run("one note removed", func(t *testing.T) {
		t.Parallel()
		m := &scanVault{
			files:      map[string]scanFile{},
			indexedVal: map[string]domain.Fingerprint{"gone.md": {Path: "gone.md", Size: 10, ModTime: epoch}},
		}
		res, err := m.scan().Execute(t.Context(), domain.Vault{ID: "v", Name: "v"})
		if err != nil {
			t.Fatal(err)
		}
		if res.Removed != 1 || m.reports != 1 {
			t.Errorf("removed %d, measured %d times, want 1 and 1", res.Removed, m.reports)
		}
	})
}

func TestAGroupThatFailsWhenFullStopsTheScanWithWhatWasCounted(t *testing.T) {
	t.Parallel()
	const perGroup = 500
	boom := errors.New("disk full")
	m := &scanVault{files: map[string]scanFile{}, saveErr: boom, failSaveAt: 1}
	for i := range perGroup + 1 {
		m.files[fmt.Sprintf("n%04d.md", i)] = noteFile(epoch.Add(time.Duration(i) * time.Second))
	}

	res, err := m.scan().Execute(t.Context(), domain.Vault{ID: "v", Name: "v"})
	if !errors.Is(err, boom) {
		t.Fatalf("error = %v, want the one the index gave", err)
	}
	if res.Notes != perGroup {
		t.Errorf("the scan had counted %d notes when its group failed, want %d", res.Notes, perGroup)
	}
	if len(m.reads) != perGroup {
		t.Errorf("read %d files, want the %d that filled the group", len(m.reads), perGroup)
	}
}

func TestAWalkWaitingForItsTurnStopsWhenCancelled(t *testing.T) {
	t.Parallel()
	target := newHeldWalk()
	readers := heldWalks{reader: target}
	db := openIndex(t)
	v := domain.Vault{ID: "one", Path: t.TempDir()}

	first := scanner(readers, db)
	isDone := make(chan error, 1)
	go func() { _, err := first.Execute(t.Context(), v); isDone <- err }()
	if !target.hasWalkBegun() {
		t.Fatal("the first walk never began")
	}
	t.Cleanup(func() { close(target.let); <-isDone })

	ctx, cancel := context.WithCancel(t.Context())
	errCh := make(chan error, 1)
	second := scanner(readers, db)
	go func() { _, err := second.Execute(ctx, v); errCh <- err }()
	time.Sleep(200 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("error = %v, want the cancellation", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("a walk waiting for its turn kept waiting after it was cancelled")
	}
}
