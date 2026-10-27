// Package hosttest is a vault and a queue that answer as the host does, for the tests of a plugin
// that works them through package host. The vault's files are held in memory, or in a folder where
// a test can change them as another program would.
package hosttest

import (
	"context"
	"errors"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
)

// Vault is a vault the host shows. Its notes are in the index as soon as their files are there.
type Vault struct {
	world    *world
	vault    host.VaultInfo
	isShown  bool
	files    store
	indexed  map[string]host.Note
	rejected map[string]error
	// unwritable are the files that may be read and not written.
	unwritable map[string]error
	cache      map[string][]byte
	locked     chan struct{}
	scanned    [][]string
	watches    []chan host.Change
	unknown    bool
	state      host.IndexState
	moves      []chan struct{}
	folder     *folder
}

var _ host.Vault = (*Vault)(nil)

// New is a vault of the name, shown by the host, whose files are held in memory.
func New(name string) *Vault {
	return newVault(host.VaultInfo{ID: host.VaultID(name), Name: name, Path: "/vaults/" + name}, newMemory())
}

// OnDisk is a vault of the notes given, written into a folder of the test's own. The notes are
// files there, and a test that changes them changes the vault.
func OnDisk(tb testing.TB, name string, notes map[string]string) *Vault {
	tb.Helper()
	root := tb.TempDir()
	for filePath, body := range notes {
		target := filepath.Join(root, filepath.FromSlash(filePath))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			tb.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
			tb.Fatal(err)
		}
	}
	v := newVault(host.VaultInfo{ID: host.VaultID(name), Name: name, Path: root}, disk{root: root})
	v.Reindex()
	return v
}

func newVault(v host.VaultInfo, files store) *Vault {
	made := &Vault{
		vault:    v,
		isShown:  true,
		files:    files,
		indexed:  map[string]host.Note{},
		rejected: map[string]error{},

		unwritable: map[string]error{},
		cache:      map[string][]byte{},
		locked:     make(chan struct{}, 1),
		state:      host.IndexState{Status: host.Indexed},
	}
	made.world = &world{members: []*Vault{made}}
	return made
}

// world is the vaults a host knows. They are worked under one lock.
type world struct {
	mu      sync.Mutex
	members []*Vault
}

// Vault is the vault as the host names it: its identity, its name and where it is.
func (v *Vault) Vault() host.VaultInfo { return v.vault }

// RefuseWriting makes every write of the file fail with err, as a file in a folder the host may not
// change does, and leaves it readable. A nil err lets it be written again.
func (v *Vault) RefuseWriting(filePath string, err error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err == nil {
		delete(v.unwritable, filePath)
		return
	}
	v.unwritable[filePath] = err
}

// Refuse makes every read and write of the file fail with err, as a file the host may not open
// does. A nil err lets the file be worked again.
func (v *Vault) Refuse(filePath string, err error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err == nil {
		delete(v.rejected, filePath)
		return
	}
	v.rejected[filePath] = err
}

// Beside makes the vault known to this one, and this one to it, so that each lists the other and
// works it by identity. Vaults made known to one another are known to all that know either.
func (v *Vault) Beside(other *Vault) {
	if v.world == other.world {
		return
	}
	merged := v.world
	for _, member := range other.world.members {
		member.world = merged
		merged.members = append(merged.members, member)
	}
}

func (v *Vault) SetIndexState(state host.IndexState) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	v.state = state
}

// Move tells the watches of moves that something moved.
func (v *Vault) Move() {
	v.world.mu.Lock()
	watches := append([]chan struct{}(nil), v.moves...)
	v.world.mu.Unlock()
	for _, w := range watches {
		w <- struct{}{}
	}
}

// errGone is what a call on the files of a vault gets when its folder is not there any longer.
var errGone = errors.New("the vault's folder is gone")

// reach refuses a call that works the vault's folder when the folder is gone.
func (v *Vault) reach() error {
	if on, isDisk := v.files.(disk); isDisk {
		if _, err := os.Stat(on.root); err != nil {
			return errGone
		}
	}
	return nil
}

// gate refuses a call on a vault the host does not show, or does not know.
func (v *Vault) gate(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if v.unknown {
		return host.ErrUnknownVault
	}
	if !v.isShown {
		return host.ErrNoVault
	}
	return nil
}

// Hide makes the host show no vault.
func (v *Vault) Hide() {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	v.isShown = false
}

// File makes a file, or replaces it, as another window would, and the index is told of it as the
// watcher would tell it.
func (v *Vault) File(filePath string, content string) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	_, _ = v.files.write(filePath, []byte(content))
	v.rescan(filePath)
}

// Remove takes a file away, as another window would.
func (v *Vault) Remove(filePath string) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	_ = v.files.remove(filePath)
	v.rescan(filePath)
}

// Content is what a file holds now.
func (v *Vault) Content(filePath string) (string, bool) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	content, err := v.files.read(filePath)
	return string(content), err == nil
}

// Scans is the lists of paths the plugin asked the index to scan, in order.
func (v *Vault) Scans() [][]string {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	return append([][]string(nil), v.scanned...)
}

// Touch tells the watches of the files, as the host does when they changed outside the plugin.
func (v *Vault) Touch(paths ...string) {
	v.world.mu.Lock()
	watches := append([]chan host.Change(nil), v.watches...)
	v.world.mu.Unlock()
	for _, w := range watches {
		w <- host.Change{Paths: paths}
	}
}

func (v *Vault) check(ctx context.Context, filePath string) error {
	if err := v.gate(ctx); err != nil {
		return err
	}
	if err := v.reach(); err != nil {
		return err
	}
	if filePath == "" || strings.HasPrefix(filePath, "/") || path.Clean(filePath) != filePath || strings.HasPrefix(filePath, "..") {
		return host.ErrOutside
	}
	return v.rejected[filePath]
}

func (v *Vault) GetVault(ctx context.Context) (host.VaultInfo, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.gate(ctx); err != nil {
		return host.VaultInfo{}, err
	}
	return v.vault, nil
}

func (v *Vault) StatFile(ctx context.Context, filePath string) (host.FileStat, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.check(ctx, filePath); err != nil {
		return host.FileStat{}, err
	}
	at, err := v.files.stat(filePath)
	if err == nil && isIgnored(filePath) {
		return host.FileStat{}, host.ErrNotANote
	}
	return at, err
}

func (v *Vault) Read(ctx context.Context, filePath string) ([]byte, host.FileStat, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.check(ctx, filePath); err != nil {
		return nil, host.FileStat{}, err
	}
	fp, err := v.files.stat(filePath)
	if err != nil {
		return nil, host.FileStat{}, err
	}
	content, err := v.files.read(filePath)
	if err != nil {
		return nil, host.FileStat{}, err
	}
	return content, fp, nil
}

func (v *Vault) Write(ctx context.Context, filePath string, content []byte, baseline host.FileStat) (host.FileStat, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.check(ctx, filePath); err != nil {
		return host.FileStat{}, err
	}
	if failErr := v.unwritable[filePath]; failErr != nil {
		return host.FileStat{}, failErr
	}
	now, err := v.files.stat(filePath)
	if err != nil {
		return host.FileStat{}, err
	}
	if !baseline.IsZero() && !now.IsUnchanged(baseline) {
		return host.FileStat{}, host.ErrStale
	}
	return v.files.write(filePath, content)
}

func (v *Vault) Create(ctx context.Context, filePath string, content []byte) error {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.check(ctx, filePath); err != nil {
		return err
	}
	if failErr := v.unwritable[filePath]; failErr != nil {
		return failErr
	}
	_, err := v.files.create(filePath, content)
	return err
}

func (v *Vault) Lock(ctx context.Context) (func(), error) {
	select {
	case v.locked <- struct{}{}:
		return func() { <-v.locked }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (v *Vault) Scan(ctx context.Context, paths []string) error {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.gate(ctx); err != nil {
		return err
	}
	v.scanned = append(v.scanned, append([]string(nil), paths...))
	v.rescan(paths...)
	return nil
}

func (v *Vault) GetNotes(ctx context.Context, paths []string) (map[string]host.NoteRef, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.gate(ctx); err != nil {
		return nil, err
	}
	notes := v.parsed()
	out := map[string]host.NoteRef{}
	for _, p := range paths {
		if n, ok := notes[p]; ok {
			out[p] = host.NoteRef{Path: p, Title: n.Title, ID: n.ID}
		}
	}
	return out, nil
}

func (v *Vault) GetTypes(ctx context.Context, paths []string) (map[string]host.NoteType, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.gate(ctx); err != nil {
		return nil, err
	}
	notes := v.parsed()
	out := map[string]host.NoteType{}
	for _, p := range paths {
		if n, ok := notes[p]; ok {
			out[p] = n.Type
		}
	}
	return out, nil
}

func (v *Vault) GetNamedPaths(ctx context.Context, name string) ([]string, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.gate(ctx); err != nil {
		return nil, err
	}
	folded := host.FoldName(name)
	var out []string
	for _, p := range sortedPaths(v.parsed()) {
		if host.FoldName(host.Basename(p)) == folded {
			out = append(out, p)
		}
	}
	return out, nil
}

func (v *Vault) GetPathsOfType(ctx context.Context, noteType host.NoteType) ([]string, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.gate(ctx); err != nil {
		return nil, err
	}
	notes := v.parsed()
	var out []string
	for _, p := range sortedPaths(notes) {
		if notes[p].Type == noteType {
			out = append(out, p)
		}
	}
	return out, nil
}

func (v *Vault) GetLinks(ctx context.Context, from string) ([]host.ResolvedLink, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.gate(ctx); err != nil {
		return nil, err
	}
	notes := v.parsed()
	n, ok := notes[from]
	if !ok {
		return nil, nil
	}
	out := make([]host.ResolvedLink, 0, len(n.Links))
	for _, l := range n.Links {
		r := host.ResolvedLink{Link: l, From: from}
		v.resolve(notes, from, &r)
		out = append(out, r)
	}
	return dedupe(out), nil
}

func (v *Vault) GetBacklinks(ctx context.Context, to string) ([]host.ResolvedLink, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.gate(ctx); err != nil {
		return nil, err
	}
	notes := v.parsed()
	target, ok := notes[to]
	if !ok {
		return nil, nil
	}
	base := host.FoldName(host.Basename(to))
	var out []host.ResolvedLink
	for _, from := range sortedPaths(notes) {
		for _, l := range notes[from].Links {
			mayReach := (l.Target.Scheme == host.SchemeNote && target.ID != "" && l.Target.Value == target.ID) ||
				(l.Target.Scheme == host.SchemeName && host.FoldName(host.LinkName(l.Target.Value)) == base)
			if !mayReach {
				continue
			}
			r := host.ResolvedLink{Link: l, From: from}
			v.resolve(notes, from, &r)
			if r.To == to && r.ToVault == v.vault.ID {
				out = append(out, r)
			}
		}
	}
	return out, nil
}

// ResolveLinks answers each written link by the links the note holds whose target is written so.
func (v *Vault) ResolveLinks(ctx context.Context, from string, links []string) (map[string]host.ResolvedLink, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.gate(ctx); err != nil {
		return nil, err
	}
	notes := v.parsed()
	out := map[string]host.ResolvedLink{}
	for _, raw := range links {
		r := host.ResolvedLink{Link: host.Link{Target: host.ParseAddress(raw)}, From: from}
		v.resolve(notes, from, &r)
		if r.To != "" {
			out[raw] = r
		}
	}
	return out, nil
}

func (v *Vault) GetParseProblems(ctx context.Context) ([]host.VaultProblem, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.gate(ctx); err != nil {
		return nil, err
	}
	notes := v.parsed()
	var out []host.VaultProblem
	for _, p := range sortedPaths(notes) {
		for _, detail := range notes[p].Problems {
			out = append(out, host.VaultProblem{Path: p, Kind: host.CheckParse, Detail: detail})
		}
	}
	return out, nil
}

func (v *Vault) ReadCache(ctx context.Context, name string) ([]byte, bool, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.gate(ctx); err != nil {
		return nil, false, err
	}
	if err := v.reach(); err != nil {
		return nil, false, err
	}
	content, ok := v.cache[name]
	return append([]byte(nil), content...), ok, nil
}

func (v *Vault) WriteCache(ctx context.Context, name string, content []byte) error {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.gate(ctx); err != nil {
		return err
	}
	if err := v.reach(); err != nil {
		return err
	}
	v.cache[name] = append([]byte(nil), content...)
	return nil
}

func (v *Vault) Watch(ctx context.Context) (<-chan host.Change, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.gate(ctx); err != nil {
		return nil, err
	}
	in := make(chan host.Change, 16)
	out := make(chan host.Change)
	v.watches = append(v.watches, in)
	go func() {
		defer close(out)
		for {
			select {
			case c := <-in:
				select {
				case out <- c:
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}

func (v *Vault) GetVaults(ctx context.Context) ([]host.VaultInfo, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	out := []host.VaultInfo{}
	if v.isShown {
		out = append(out, v.vault)
	}
	for _, other := range v.world.members {
		if other != v && other.isShown {
			out = append(out, other.vault)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (v *Vault) With(id host.VaultID) host.Vault {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	return v.find(id)
}

// find is the vault of the identity among those the host knows. The caller holds the world's lock.
func (v *Vault) find(id host.VaultID) *Vault {
	if v.vault.ID == id {
		return v
	}
	for _, other := range v.world.members {
		if other.vault.ID == id {
			return other
		}
	}
	return &Vault{unknown: true, world: v.world}
}

func (v *Vault) EnsureIndexed(ctx context.Context) (host.IndexState, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.gate(ctx); err != nil {
		return host.IndexState{}, err
	}
	return v.state, nil
}

func (v *Vault) WatchMoves(ctx context.Context) (<-chan struct{}, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if v.unknown {
		return nil, host.ErrUnknownVault
	}
	in := make(chan struct{}, 16)
	out := make(chan struct{})
	v.moves = append(v.moves, in)
	go func() {
		defer close(out)
		for {
			select {
			case <-in:
				select {
				case out <- struct{}{}:
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}
