package hosttest

import (
	"sort"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/libs/notes/markdown"
)

// kindOf is what sort of source a file is, from its name alone.
func kindOf(path string) domain.SourceKind {
	switch strings.ToLower(path[strings.LastIndexByte(path, '.')+1:]) {
	case "md":
		return domain.KindNote
	case "url":
		return domain.KindURL
	case "mp3", "wav", "m4a", "flac", "ogg":
		return domain.KindRecording
	case "epub", "pdf":
		return domain.KindBook
	}
	return domain.KindNote
}

// isNote says whether the index holds the file as a note.
func isNote(path string) bool { return strings.HasSuffix(strings.ToLower(path), domain.NoteExtension) }

// isIgnored says whether the vault leaves the file alone: a file in a hidden folder, or a hidden file.
func isIgnored(filePath string) bool {
	for _, segment := range strings.Split(filePath, "/") {
		if strings.HasPrefix(segment, ".") {
			return true
		}
	}
	return false
}

// parsed is every note the index holds. The index is a picture of the files as they were when it was
// last brought level, and a file changed since is read again only once the index is told of it. The
// caller holds the lock.
func (v *Vault) parsed() map[string]domain.Note {
	out := make(map[string]domain.Note, len(v.indexed))
	for filePath, held := range v.indexed {
		out[filePath] = held
	}
	return out
}

// rescan brings the index level with the files named, the way the host does when it is told of a
// change. The caller holds the lock.
func (v *Vault) rescan(paths ...string) {
	for _, filePath := range paths {
		at, err := v.files.stat(filePath)
		if err != nil || !isNote(filePath) || isIgnored(filePath) {
			delete(v.indexed, filePath)
			continue
		}
		raw, err := v.files.read(filePath)
		if err != nil {
			delete(v.indexed, filePath)
			continue
		}
		v.indexed[filePath] = markdown.Parse(at, raw)
	}
}

// Reindex brings the index level with every file, as a walk of the vault does.
func (v *Vault) Reindex() {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	v.indexed = map[string]domain.Note{}
	v.rescan(v.files.paths()...)
}

func sortedPaths(notes map[string]domain.Note) []string {
	paths := make([]string, 0, len(notes))
	for path := range notes {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

// candidates is everything a name could mean in the vault: a path from the root, the same with the
// extension, or a file with that name anywhere.
func candidates(notes map[string]domain.Note, name string) []string {
	base := domain.FoldName(domain.LinkName(name))
	var out []string
	for _, path := range sortedPaths(notes) {
		if path == name || path == name+domain.NoteExtension || domain.FoldName(domain.Basename(path)) == base {
			out = append(out, path)
		}
	}
	return out
}

// resolve says where a link lands.
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
		r.To, r.IsAmbiguous = domain.ChooseMatch(from, r.Target.Value, candidates(notes, r.Target.Value))
		if r.To != "" {
			r.ToVault = v.vault.ID
		}
	}
}

// vaultsHolding is this vault and the ones beside it.
func (v *Vault) vaultsHolding() []*Vault { return v.world.members }

// dedupe folds links that turn out to point at the same note: the described one wins over a mention
// in prose.
func dedupe(links []domain.ResolvedLink) []domain.ResolvedLink {
	seen := map[string]int{}
	var out []domain.ResolvedLink
	for _, l := range links {
		key := l.To
		if key == "" {
			key = "\x00" + l.Target.String()
		}
		at, known := seen[key]
		if !known {
			seen[key] = len(out)
			out = append(out, l)
			continue
		}
		if out[at].Role == domain.RoleRef && l.Role != domain.RoleRef {
			out[at] = l
		}
	}
	return out
}
