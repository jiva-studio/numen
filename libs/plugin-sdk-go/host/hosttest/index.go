package hosttest

import (
	"fmt"
	"sort"
	"strings"

	notesdomain "github.com/jiva-studio/numen/libs/notes/domain"
	notesmarkdown "github.com/jiva-studio/numen/libs/notes/markdown"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
)

func getKind(filePath string) domain.SourceKind {
	if isNote(filePath) {
		return domain.KindNote
	}
	return domain.KindFile
}

func isNote(path string) bool { return strings.HasSuffix(strings.ToLower(path), domain.NoteExtension) }

func isIgnored(filePath string) bool {
	for _, segment := range strings.Split(filePath, "/") {
		if strings.HasPrefix(segment, ".") {
			return true
		}
	}
	return false
}

func (v *Vault) parsed() map[string]domain.Note {
	out := make(map[string]domain.Note, len(v.indexed))
	for filePath, target := range v.indexed {
		out[filePath] = target
	}
	return out
}

func (v *Vault) rescan(paths ...string) {
	for _, filePath := range paths {
		fp, err := v.files.stat(filePath)
		if err != nil || !isNote(filePath) || isIgnored(filePath) {
			delete(v.indexed, filePath)
			continue
		}
		raw, err := v.files.read(filePath)
		if err != nil {
			delete(v.indexed, filePath)
			continue
		}
		v.indexed[filePath] = notesmarkdown.Parse(fp, raw)
	}
}

// Reindex updates the index for every file, as a walk of the vault does.
func (v *Vault) Reindex() {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	v.indexed = map[string]domain.Note{}
	paths, err := v.files.paths()
	if err != nil {
		v.report(fmt.Errorf("list files: %w", err))
	}
	v.rescan(paths...)
}

func sortedPaths(notes map[string]domain.Note) []string {
	paths := make([]string, 0, len(notes))
	for path := range notes {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func candidates(notes map[string]domain.Note, name string) []string {
	base := notesdomain.FoldName(notesdomain.LinkName(name))
	var out []string
	for _, path := range sortedPaths(notes) {
		if path == name || path == name+domain.NoteExtension || notesdomain.FoldName(domain.Basename(path)) == base {
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
		r.To, r.IsAmbiguous = notesdomain.ChooseMatch(from, r.Target.Value, candidates(notes, r.Target.Value))
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
