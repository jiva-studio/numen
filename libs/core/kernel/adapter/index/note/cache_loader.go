package note

import (
	"context"
	"database/sql"
	"encoding/json"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// vaultNotes is the notes of a vault as the cache reads them, each map by the path of the note.
type vaultNotes struct {
	notes          map[string]cachedNoteRef
	types          map[string]cachedNoteType
	rawFrontmatter map[string]sql.NullString
	byPath         map[string]string
	idByPath       map[string]string
	byFolded       map[string][]string
}

func loadVaultNotes(ctx context.Context, db *sql.DB, vault int64) (loaded vaultNotes, err error) {
	rows, err := db.QueryContext(ctx, vaultNotesSQL, vault)
	if err != nil {
		return vaultNotes{}, err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()
	loaded = vaultNotes{
		notes:          map[string]cachedNoteRef{},
		types:          map[string]cachedNoteType{},
		rawFrontmatter: map[string]sql.NullString{},
		byPath:         map[string]string{},
		idByPath:       map[string]string{},
		byFolded:       map[string][]string{},
	}
	for rows.Next() {
		var path, folded, title, noteType, id string
		var stored sql.NullString
		if err := rows.Scan(&path, &folded, &title, &noteType, &id, &stored); err != nil {
			return vaultNotes{}, err
		}
		loaded.notes[path] = cachedNoteRef{ref: domain.NoteRef{Path: path, Title: title, ID: id}, exists: true}
		loaded.types[path] = cachedNoteType{noteType: domain.NoteType(noteType), exists: true}
		loaded.rawFrontmatter[path] = stored
		loaded.byPath[path] = folded
		if id != "" {
			loaded.idByPath[path] = id
		}
		loaded.byFolded[folded] = append(loaded.byFolded[folded], path)
	}
	return loaded, rows.Err()
}

// vaultLinks is the links of a vault by the note they leave, the notes in the order they were read, and the
// identifiers the links to a note by identifier name.
type vaultLinks struct {
	order   []string
	grouped map[string][]rawVaultLink
	noteIDs []string
}

func loadVaultLinks(ctx context.Context, db *sql.DB, vault int64) (loaded vaultLinks, err error) {
	rows, err := db.QueryContext(ctx, vaultLinksSQL, vault)
	if err != nil {
		return vaultLinks{}, err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()
	loaded.grouped = map[string][]rawVaultLink{}
	seenIDs := map[string]bool{}
	for rows.Next() {
		var path, role, marker, folded string
		var l domain.ResolvedLink
		if err := rows.Scan(&path, &l.Target.Scheme, &l.Target.Value, &role,
			&l.Type, &l.Why, &l.Label, &marker, &folded); err != nil {
			return vaultLinks{}, err
		}
		l.From = path
		l.Role = domain.LinkRole(role)
		l.Marker = domain.LinkMarker(marker)
		if _, seen := loaded.grouped[path]; !seen {
			loaded.order = append(loaded.order, path)
		}
		loaded.grouped[path] = append(loaded.grouped[path], rawVaultLink{link: l, folded: folded})
		if l.Target.Scheme == domain.SchemeNote && !seenIDs[l.Target.Value] {
			seenIDs[l.Target.Value] = true
			loaded.noteIDs = append(loaded.noteIDs, l.Target.Value)
		}
	}
	return loaded, rows.Err()
}

func loadIdentifiedNotes(
	ctx context.Context, db *sql.DB, noteIDs []string,
) (_ map[string]identifiedNote, err error) {
	identified := make(map[string]identifiedNote, len(noteIDs))
	if len(noteIDs) == 0 {
		return identified, nil
	}
	idsJSON, err := json.Marshal(noteIDs)
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, notesByIdentifierAtSQL, string(idsJSON))
	if err != nil {
		return nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()
	for rows.Next() {
		var id, path string
		var vault domain.VaultID
		if err := rows.Scan(&id, &vault, &path); err != nil {
			return nil, err
		}
		if identified[id].path == "" {
			identified[id] = identifiedNote{vault: vault, path: path}
		}
	}
	return identified, rows.Err()
}

type cachedResolution struct {
	vaultID    domain.VaultID
	byPath     map[string]string
	byFolded   map[string][]string
	named      map[string][]string
	identified map[string]identifiedNote
}

func (r *cachedResolution) resolveLink(from string, l *domain.ResolvedLink) {
	switch l.Target.Scheme {
	case domain.SchemeNote:
		if found, ok := r.identified[l.Target.Value]; ok {
			l.ToVault, l.To = found.vault, found.path
		}
	case domain.SchemeName:
		name := l.Target.Value
		candidates, ok := r.named[name]
		if !ok {
			candidates = lookupCandidatesInMemory(name, r.byPath, r.byFolded)
			r.named[name] = candidates
		}
		l.To, l.IsAmbiguous = domain.ChooseMatch(from, name, candidates)
		if l.To != "" {
			l.ToVault = r.vaultID
		}
	}
}

func lookupCandidatesInMemory(name string, byPath map[string]string, byFolded map[string][]string) []string {
	base := domain.NormalizeName(domain.LinkName(name))
	candidates := byFolded[base]
	withExt := name + domain.NoteExtension
	_, hasExact := byPath[name]
	_, hasWithExt := byPath[withExt]
	needExact := hasExact && !slices.Contains(candidates, name)
	needWithExt := hasWithExt && withExt != name && !slices.Contains(candidates, withExt)
	if !needExact && !needWithExt {
		return candidates
	}
	merged := slices.Clone(candidates)
	if needExact {
		merged = append(merged, name)
	}
	if needWithExt {
		merged = append(merged, withExt)
	}
	slices.Sort(merged)
	return merged
}

func isBacklinkMatch(l domain.ResolvedLink, rawFolded, targetID, targetFolded string) bool {
	if l.Target.Scheme == domain.SchemeNote {
		return targetID != "" && l.Target.Value == targetID
	}
	return l.Target.Scheme == domain.SchemeName && rawFolded == targetFolded
}
