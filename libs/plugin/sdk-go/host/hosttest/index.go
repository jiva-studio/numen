package hosttest

import (
	"context"
	"fmt"
	"maps"
	"sort"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	notesdomain "github.com/jiva-studio/numen/libs/shared/note/domain"
	notesmarkdown "github.com/jiva-studio/numen/libs/shared/note/markdown"
	"github.com/jiva-studio/numen/libs/shared/protocol"
)

func getKind(filePath string) domain.SourceKind {
	if isNote(filePath) {
		return domain.KindNote
	}
	return domain.KindFile
}

func isNote(path string) bool { return strings.HasSuffix(strings.ToLower(path), domain.NoteExtension) }

func isIgnored(filePath string) bool {
	for segment := range strings.SplitSeq(filePath, "/") {
		if strings.HasPrefix(segment, ".") {
			return true
		}
	}
	return false
}

func (v *Vault) parsed() map[string]domain.Note {
	out := make(map[string]domain.Note, len(v.indexed))
	maps.Copy(out, v.indexed)
	return out
}

func (v *Vault) rescan(paths ...string) {
	for _, filePath := range paths {
		v.rescanFile(filePath)
	}
}

func (v *Vault) rescanFile(filePath string) {
	fp, err := v.files.stat(filePath)
	if err != nil || isIgnored(filePath) {
		delete(v.indexed, filePath)
		v.feed.observe(filePath, nil, false, nil)
		return
	}
	raw, err := v.files.read(filePath)
	if err != nil {
		delete(v.indexed, filePath)
		v.feed.observe(filePath, nil, false, nil)
		return
	}
	if !isNote(filePath) {
		delete(v.indexed, filePath)
		v.feed.observe(filePath, raw, true, nil)
		return
	}
	note := notesmarkdown.Parse(fp, raw)
	v.indexed[filePath] = note
	v.feed.observe(filePath, raw, true, note.Frontmatter)
}

// Reindex updates the index for every file, as a walk of the vault does.
func (v *Vault) Reindex() {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	v.indexed = map[string]domain.Note{}
	paths, err := v.files.listPaths()
	if err != nil {
		v.report(fmt.Errorf("list files: %w", err))
	}
	v.rescan(paths...)
}

func listSortedPaths(notes map[string]domain.Note) []string {
	paths := make([]string, 0, len(notes))
	for path := range notes {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func findCandidates(notes map[string]domain.Note, name string) []string {
	base := notesdomain.NormalizeName(notesdomain.LinkName(name))
	var out []string
	for _, path := range listSortedPaths(notes) {
		if path == name || path == name+domain.NoteExtension || notesdomain.NormalizeName(domain.GetBasename(path)) == base {
			out = append(out, path)
		}
	}
	return out
}

func (v *Vault) resolve(notes map[string]domain.Note, from string, r *domain.ResolvedLink) {
	switch r.Target.Scheme {
	case domain.SchemeNote:
		for _, holder := range v.vaultsHolding() {
			for path, n := range holder.parsed() {
				if n.ID != "" && n.ID == r.Target.Value {
					r.To, r.ToVault = path, holder.vault.ID
					return
				}
			}
		}
	case domain.SchemeName:
		r.To, r.IsAmbiguous = notesdomain.ChooseMatch(from, r.Target.Value, findCandidates(notes, r.Target.Value))
		if r.To != "" {
			r.ToVault = v.vault.ID
		}
	}
}

func (v *Vault) vaultsHolding() []*Vault { return v.world.members }

func dedupe(links []domain.ResolvedLink) []domain.ResolvedLink {
	observed := map[string]int{}
	var out []domain.ResolvedLink
	for _, l := range links {
		key := l.To
		if key == "" {
			key = "\x00" + l.Target.String()
		}
		at, recognized := observed[key]
		if !recognized {
			observed[key] = len(out)
			out = append(out, l)
			continue
		}
		if out[at].Role == domain.RoleRef && l.Role != domain.RoleRef {
			out[at] = l
		}
	}
	return out
}

func (v *Vault) SetIndexState(state host.IndexState) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	v.state = state
}

// Scans returns the lists of paths the plugin requested the index to scan, in order.
func (v *Vault) Scans() [][]string {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	return append([][]string(nil), v.scanned...)
}

func (v *Vault) Scan(ctx context.Context, paths []string) error {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.checkContext(ctx); err != nil {
		return err
	}
	v.scanned = append(v.scanned, append([]string(nil), paths...))
	v.rescan(paths...)
	return nil
}

func (v *Vault) GetNotes(ctx context.Context, paths []string) (map[string]domain.NoteRef, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.checkContext(ctx); err != nil {
		return nil, err
	}
	notes := v.parsed()
	out := map[string]domain.NoteRef{}
	for _, p := range paths {
		if n, ok := notes[p]; ok {
			out[p] = domain.NoteRef{Path: p, Title: n.Title, ID: n.ID}
		}
	}
	return out, nil
}

func (v *Vault) GetTypes(ctx context.Context, paths []string) (map[string]domain.NoteType, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.checkContext(ctx); err != nil {
		return nil, err
	}
	notes := v.parsed()
	out := map[string]domain.NoteType{}
	for _, p := range paths {
		if n, ok := notes[p]; ok {
			out[p] = n.Type
		}
	}
	return out, nil
}

func (v *Vault) ListNamedPaths(ctx context.Context, name string) ([]string, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.checkContext(ctx); err != nil {
		return nil, err
	}
	folded := notesdomain.NormalizeName(name)
	var out []string
	for _, p := range listSortedPaths(v.parsed()) {
		if notesdomain.NormalizeName(domain.GetBasename(p)) == folded {
			out = append(out, p)
		}
	}
	return out, nil
}

func (v *Vault) ListPathsOfType(ctx context.Context, noteType domain.NoteType) ([]string, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.checkContext(ctx); err != nil {
		return nil, err
	}
	notes := v.parsed()
	var out []string
	for _, p := range listSortedPaths(notes) {
		if notes[p].Type == noteType {
			out = append(out, p)
		}
	}
	return out, nil
}

func (v *Vault) EnsureIndexed(ctx context.Context) (host.IndexState, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.checkContext(ctx); err != nil {
		return host.IndexState{}, err
	}
	return v.state, nil
}

// GetIndexChangedEvent returns the event the host emits when an indexing pass changed the index of the vault
// of the given identity, for a test that delivers it to a plugin.
func GetIndexChangedEvent(vaultID string) string {
	return protocol.GetIndexChangedEvent(vaultID)
}
