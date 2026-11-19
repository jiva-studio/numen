package note

import (
	"context"
	"database/sql"
	"encoding/json"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func loadVaultNotes(
	ctx context.Context, db *sql.DB, vault int64,
) (
	notes map[string]cachedNoteRef,
	types map[string]cachedNoteType,
	rawFrontmatter map[string]sql.NullString,
	byPath map[string]string,
	idByPath map[string]string,
	byFolded map[string][]string,
	err error,
) {
	rows, err := db.QueryContext(ctx, vaultNotesSQL, vault)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()
	notes = map[string]cachedNoteRef{}
	types = map[string]cachedNoteType{}
	rawFrontmatter = map[string]sql.NullString{}
	byPath = map[string]string{}
	idByPath = map[string]string{}
	byFolded = map[string][]string{}
	for rows.Next() {
		var path, folded, title, noteType, id string
		var stored sql.NullString
		if err := rows.Scan(&path, &folded, &title, &noteType, &id, &stored); err != nil {
			return nil, nil, nil, nil, nil, nil, err
		}
		notes[path] = cachedNoteRef{ref: domain.NoteRef{Path: path, Title: title, ID: id}, exists: true}
		types[path] = cachedNoteType{noteType: domain.NoteType(noteType), exists: true}
		rawFrontmatter[path] = stored
		byPath[path] = folded
		if id != "" {
			idByPath[path] = id
		}
		byFolded[folded] = append(byFolded[folded], path)
	}
	return notes, types, rawFrontmatter, byPath, idByPath, byFolded, rows.Err()
}

func loadVaultLinks(
	ctx context.Context, db *sql.DB, vault int64,
) (order []string, grouped map[string][]rawVaultLink, noteIDs []string, err error) {
	rows, err := db.QueryContext(ctx, vaultLinksSQL, vault)
	if err != nil {
		return nil, nil, nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()
	grouped = map[string][]rawVaultLink{}
	seenIDs := map[string]bool{}
	for rows.Next() {
		var path, role, marker, folded string
		var l domain.ResolvedLink
		if err := rows.Scan(&path, &l.Target.Scheme, &l.Target.Value, &role,
			&l.Type, &l.Why, &l.Label, &marker, &folded); err != nil {
			return nil, nil, nil, err
		}
		l.From = path
		l.Role = domain.LinkRole(role)
		l.Marker = domain.LinkMarker(marker)
		if _, seen := grouped[path]; !seen {
			order = append(order, path)
		}
		grouped[path] = append(grouped[path], rawVaultLink{link: l, folded: folded})
		if l.Target.Scheme == domain.SchemeNote && !seenIDs[l.Target.Value] {
			seenIDs[l.Target.Value] = true
			noteIDs = append(noteIDs, l.Target.Value)
		}
	}
	return order, grouped, noteIDs, rows.Err()
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
	base := domain.FoldName(domain.LinkName(name))
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
