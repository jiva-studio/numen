// Package hosttest is a vault held in memory that answers as the host does, for the tests of a plugin
// that works a vault through package host.
package hosttest

import (
	"context"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/host"
)

// Vault is a vault in memory. A file that is written is a note when it was made with Note, and
// every write moves the file's modification time on.
type Vault struct {
	mu      sync.Mutex
	vault   domain.Vault
	shown   bool
	files   map[string][]byte
	at      map[string]domain.Fingerprint
	notes   map[string]domain.NoteRef
	types   map[string]domain.NoteType
	links   map[string][]domain.ResolvedLink
	data    map[string][]byte
	cache   map[string][]byte
	problem []domain.VaultProblem
	clock   time.Time
	locked  chan struct{}
	scanned [][]string
	watches []chan host.Change
}

var _ host.Vault = (*Vault)(nil)

// New is a vault of the name, shown by the host.
func New(name string) *Vault {
	return &Vault{
		vault:  domain.Vault{ID: domain.VaultID(name), Name: name, Path: "/vaults/" + name},
		shown:  true,
		files:  map[string][]byte{},
		at:     map[string]domain.Fingerprint{},
		notes:  map[string]domain.NoteRef{},
		types:  map[string]domain.NoteType{},
		links:  map[string][]domain.ResolvedLink{},
		data:   map[string][]byte{},
		cache:  map[string][]byte{},
		clock:  time.Unix(1_700_000_000, 0),
		locked: make(chan struct{}, 1),
	}
}

// Hide makes the host show no vault.
func (v *Vault) Hide() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.shown = false
}

// Note makes a file that the index knows as a note of the type and the title.
func (v *Vault) Note(notePath string, noteType domain.NoteType, title string, content string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.put(notePath, []byte(content))
	v.notes[notePath] = domain.NoteRef{Path: notePath, Title: title}
	v.types[notePath] = noteType
}

// File makes a file the index knows nothing of.
func (v *Vault) File(filePath string, content string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.put(filePath, []byte(content))
}

// SetLinks sets the links the index holds from a note.
func (v *Vault) SetLinks(from string, links ...domain.ResolvedLink) {
	v.mu.Lock()
	defer v.mu.Unlock()
	for i := range links {
		links[i].From = from
	}
	v.links[from] = links
}

// SetProblems sets what the parse of the vault turned up.
func (v *Vault) SetProblems(problems ...domain.VaultProblem) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.problem = problems
}

// Content is what a file holds now.
func (v *Vault) Content(filePath string) (string, bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	content, ok := v.files[filePath]
	return string(content), ok
}

// Scans is the lists of paths the plugin asked the index to scan, in order.
func (v *Vault) Scans() [][]string {
	v.mu.Lock()
	defer v.mu.Unlock()
	return append([][]string(nil), v.scanned...)
}

// Touch tells the watches of the files, as the host does when they changed outside the plugin.
func (v *Vault) Touch(paths ...string) {
	v.mu.Lock()
	watches := append([]chan host.Change(nil), v.watches...)
	v.mu.Unlock()
	for _, w := range watches {
		w <- host.Change{Paths: paths}
	}
}

func (v *Vault) put(filePath string, content []byte) domain.Fingerprint {
	v.clock = v.clock.Add(time.Second)
	v.files[filePath] = content
	at := domain.Fingerprint{Path: filePath, Size: int64(len(content)), ModTime: v.clock}
	v.at[filePath] = at
	return at
}

func (v *Vault) check(filePath string) error {
	if !v.shown {
		return host.ErrNoVault
	}
	if filePath == "" || strings.HasPrefix(filePath, "/") || path.Clean(filePath) != filePath || strings.HasPrefix(filePath, "..") {
		return host.ErrOutside
	}
	return nil
}

func (v *Vault) GetVault(context.Context) (domain.Vault, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if !v.shown {
		return domain.Vault{}, host.ErrNoVault
	}
	return v.vault, nil
}

func (v *Vault) GetFingerprint(_ context.Context, filePath string) (domain.Fingerprint, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if err := v.check(filePath); err != nil {
		return domain.Fingerprint{}, err
	}
	at, ok := v.at[filePath]
	if !ok {
		return domain.Fingerprint{}, host.ErrNotExist
	}
	return at, nil
}

func (v *Vault) Read(_ context.Context, filePath string) ([]byte, domain.Fingerprint, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if err := v.check(filePath); err != nil {
		return nil, domain.Fingerprint{}, err
	}
	content, ok := v.files[filePath]
	if !ok {
		return nil, domain.Fingerprint{}, host.ErrNotExist
	}
	return append([]byte(nil), content...), v.at[filePath], nil
}

func (v *Vault) Write(_ context.Context, filePath string, content []byte, baseline domain.Fingerprint) (domain.Fingerprint, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if err := v.check(filePath); err != nil {
		return domain.Fingerprint{}, err
	}
	now, ok := v.at[filePath]
	if !ok {
		return domain.Fingerprint{}, host.ErrNotExist
	}
	if now.Size != baseline.Size || !now.ModTime.Equal(baseline.ModTime) {
		return domain.Fingerprint{}, host.ErrStale
	}
	return v.put(filePath, append([]byte(nil), content...)), nil
}

func (v *Vault) Create(_ context.Context, filePath string, content []byte) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if err := v.check(filePath); err != nil {
		return err
	}
	if _, ok := v.files[filePath]; ok {
		return host.ErrOccupied
	}
	v.put(filePath, append([]byte(nil), content...))
	return nil
}

func (v *Vault) Lock(ctx context.Context) (func(), error) {
	select {
	case v.locked <- struct{}{}:
		return func() { <-v.locked }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (v *Vault) Scan(_ context.Context, paths []string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if !v.shown {
		return host.ErrNoVault
	}
	v.scanned = append(v.scanned, append([]string(nil), paths...))
	return nil
}

func (v *Vault) GetNotes(_ context.Context, paths []string) (map[string]domain.NoteRef, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if !v.shown {
		return nil, host.ErrNoVault
	}
	out := map[string]domain.NoteRef{}
	for _, p := range paths {
		if ref, ok := v.notes[p]; ok {
			out[p] = ref
		}
	}
	return out, nil
}

func (v *Vault) GetTypes(_ context.Context, paths []string) (map[string]domain.NoteType, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if !v.shown {
		return nil, host.ErrNoVault
	}
	out := map[string]domain.NoteType{}
	for _, p := range paths {
		if t, ok := v.types[p]; ok {
			out[p] = t
		}
	}
	return out, nil
}

func (v *Vault) GetPathsOfType(_ context.Context, noteType domain.NoteType) ([]string, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if !v.shown {
		return nil, host.ErrNoVault
	}
	var out []string
	for p, t := range v.types {
		if t == noteType {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out, nil
}

func (v *Vault) GetLinks(_ context.Context, from string) ([]domain.ResolvedLink, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if !v.shown {
		return nil, host.ErrNoVault
	}
	return append([]domain.ResolvedLink(nil), v.links[from]...), nil
}

func (v *Vault) GetBacklinks(_ context.Context, to string) ([]domain.ResolvedLink, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if !v.shown {
		return nil, host.ErrNoVault
	}
	var out []domain.ResolvedLink
	for _, links := range v.links {
		for _, l := range links {
			if l.To == to {
				out = append(out, l)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].From < out[j].From })
	return out, nil
}

// ResolveLinks answers each written link by the links the note holds whose target is written so.
func (v *Vault) ResolveLinks(_ context.Context, from string, written []string) (map[string]domain.ResolvedLink, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if !v.shown {
		return nil, host.ErrNoVault
	}
	out := map[string]domain.ResolvedLink{}
	for _, w := range written {
		for _, l := range v.links[from] {
			if l.Target.Value == w {
				out[w] = l
				break
			}
		}
	}
	return out, nil
}

func (v *Vault) GetParseProblems(context.Context) ([]domain.VaultProblem, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if !v.shown {
		return nil, host.ErrNoVault
	}
	return append([]domain.VaultProblem(nil), v.problem...), nil
}

func (v *Vault) ReadData(_ context.Context, name string) ([]byte, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if !v.shown {
		return nil, host.ErrNoVault
	}
	content, ok := v.data[name]
	if !ok {
		return nil, host.ErrNotExist
	}
	return append([]byte(nil), content...), nil
}

func (v *Vault) AppendData(_ context.Context, name string, content []byte) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if !v.shown {
		return host.ErrNoVault
	}
	v.data[name] = append(v.data[name], content...)
	return nil
}

func (v *Vault) ListData(context.Context) ([]host.DataFile, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if !v.shown {
		return nil, host.ErrNoVault
	}
	out := make([]host.DataFile, 0, len(v.data))
	for name, content := range v.data {
		out = append(out, host.DataFile{Name: name, Size: int64(len(content))})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (v *Vault) ReadCache(_ context.Context, name string) ([]byte, bool, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if !v.shown {
		return nil, false, host.ErrNoVault
	}
	content, ok := v.cache[name]
	return append([]byte(nil), content...), ok, nil
}

func (v *Vault) WriteCache(_ context.Context, name string, content []byte) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if !v.shown {
		return host.ErrNoVault
	}
	v.cache[name] = append([]byte(nil), content...)
	return nil
}

func (v *Vault) Watch(ctx context.Context) (<-chan host.Change, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if !v.shown {
		return nil, host.ErrNoVault
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
