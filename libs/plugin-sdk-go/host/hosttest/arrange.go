package hosttest

import (
	"context"
	"fmt"
	"path"
	"slices"
	"strconv"
	"strings"

	notesdomain "github.com/jiva-studio/numen/libs/notes/domain"
	notesmarkdown "github.com/jiva-studio/numen/libs/notes/markdown"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
)

// trashDir is the folder of the vault that removed notes are moved to.
const trashDir = ".trash"

// RenameNote sets the title key of the note's frontmatter, or leaves the note unwritten when it has no
// title key and its filename can carry the title, and moves the file to the filename of the title in
// its folder. It repairs no link.
func (v *Vault) RenameNote(ctx context.Context, notePath, title string) (host.NoteRename, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.checkNote(ctx, notePath); err != nil {
		return host.NoteRename{}, err
	}
	title = strings.TrimSpace(title)
	name, exact, err := notesdomain.Filename(title)
	if err != nil {
		return host.NoteRename{}, err
	}
	target := path.Join(path.Dir(notePath), name+path.Ext(notePath))
	if !strings.EqualFold(notePath, target) && v.isPresent(target) {
		return host.NoteRename{}, notesdomain.NewDetailError(fmt.Sprintf("%s already exists; %s is unchanged", target, notePath), host.ErrAlreadyExists)
	}
	isFilenamed, err := v.writeTitle(notePath, title, exact)
	if err != nil {
		return host.NoteRename{}, err
	}
	out := host.NoteRename{Path: notePath, Title: title, IsFilenamed: isFilenamed}
	if target == notePath {
		return out, nil
	}
	if err := v.moveFile(notePath, target); err != nil {
		return out, err
	}
	out.Path = target
	out.Move = &domain.MoveResult{From: notePath, To: target, IsMoved: true}
	return out, nil
}

// writeTitle writes the title into the frontmatter of the note, unless the note has no title key and
// its filename carries the title exactly, and reports whether the filename carries it. The caller
// holds the lock.
func (v *Vault) writeTitle(notePath, title string, isExact bool) (bool, error) {
	raw, err := v.files.read(notePath)
	if err != nil {
		return false, err
	}
	doc, err := notesmarkdown.Open(raw)
	if err != nil {
		return false, err
	}
	if _, isTitled := doc.Title(); !isTitled && isExact {
		return true, nil
	}
	if err := doc.SetTitle(title); err != nil {
		return false, err
	}
	if _, err := v.files.write(notePath, doc.Bytes()); err != nil {
		return false, err
	}
	v.rescan(notePath)
	return false, nil
}

// MoveNotes moves each file to the folder under its own filename. It repairs no link.
func (v *Vault) MoveNotes(ctx context.Context, paths []string, folder string) ([]host.NoteMove, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.gate(ctx); err != nil {
		return nil, err
	}
	if folder != "" && !isInside(folder) {
		return nil, host.ErrOutside
	}
	out := make([]host.NoteMove, 0, len(paths))
	for _, from := range paths {
		one := host.NoteMove{Move: domain.MoveResult{From: from}}
		target := path.Join(folder, path.Base(from))
		one.Err = v.checkFile(ctx, from)
		if one.Err == nil && target != from {
			one.Err = v.moveFile(from, target)
		}
		if one.Err == nil && target != from {
			one.Move.To, one.Move.IsMoved = target, true
		}
		out = append(out, one)
	}
	return out, nil
}

// RemoveNotes moves each note to `.trash/<path>`, or to a numbered name there when the trash holds one,
// and reports the notes whose links reached it and now reach nothing.
func (v *Vault) RemoveNotes(ctx context.Context, paths []string) ([]host.NoteRemoval, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.gate(ctx); err != nil {
		return nil, err
	}
	out := make([]host.NoteRemoval, 0, len(paths))
	for _, notePath := range paths {
		one := host.NoteRemoval{Path: notePath}
		one.Err = v.checkNote(ctx, notePath)
		if one.Err == nil {
			one.TrashPath, one.BrokenLinks, one.Err = v.trashNote(notePath)
		}
		out = append(out, one)
	}
	return out, nil
}

// trashNote moves the note into the trash and returns where it is and the notes whose links reached
// it and reach nothing now. The caller holds the lock.
func (v *Vault) trashNote(notePath string) (string, []string, error) {
	backlinks := v.getBacklinks(v.parsed(), notePath)
	target := path.Join(trashDir, notePath)
	for attempt := 2; v.isPresent(target); attempt++ {
		ext := path.Ext(notePath)
		target = path.Join(trashDir, strings.TrimSuffix(notePath, ext)+"-"+strconv.Itoa(attempt)+ext)
	}
	if err := v.moveFile(notePath, target); err != nil {
		return "", nil, err
	}
	return target, v.getDangling(backlinks), nil
}

// getDangling is the notes, by path, of the links given that reach nothing now. The caller holds the
// lock.
func (v *Vault) getDangling(links []domain.ResolvedLink) []string {
	notes := v.parsed()
	var out []string
	for _, link := range links {
		if _, isIndexed := notes[link.From]; !isIndexed {
			continue
		}
		again := domain.ResolvedLink{Link: link.Link, From: link.From}
		v.resolve(notes, link.From, &again)
		if again.To == "" && !slices.Contains(out, link.From) {
			out = append(out, link.From)
		}
	}
	return out
}

// checkFile fails a call on a file that is not there, that the vault skips, or that may not be
// written. The caller holds the lock.
func (v *Vault) checkFile(ctx context.Context, filePath string) error {
	if err := v.check(ctx, filePath); err != nil {
		return err
	}
	if isIgnored(filePath) {
		return host.ErrNotANote
	}
	if _, err := v.files.stat(filePath); err != nil {
		return err
	}
	return v.unwritable[filePath]
}

// checkNote is checkFile for a note: a folder or a file that is not a note is host.ErrNotANote. The
// caller holds the lock.
func (v *Vault) checkNote(ctx context.Context, notePath string) error {
	if v.isFolder(notePath) {
		return host.ErrNotANote
	}
	if err := v.checkFile(ctx, notePath); err != nil {
		return err
	}
	if !isNote(notePath) {
		return host.ErrNotANote
	}
	return nil
}

// isFolder reports whether a file of the vault is under the path. The caller holds the lock.
func (v *Vault) isFolder(folder string) bool {
	paths, err := v.files.paths()
	if err != nil {
		return false
	}
	for _, filePath := range paths {
		if strings.HasPrefix(filePath, folder+"/") {
			return true
		}
	}
	return false
}

// isPresent reports whether a file is at the path. The caller holds the lock.
func (v *Vault) isPresent(filePath string) bool {
	_, err := v.files.stat(filePath)
	return err == nil
}

// moveFile moves the file to the target, which must be free unless it differs only in case, and
// brings the index current with both paths. The caller holds the lock.
func (v *Vault) moveFile(from, target string) error {
	if !strings.EqualFold(from, target) && v.isPresent(target) {
		return notesdomain.NewDetailError(fmt.Sprintf("%s already exists; %s is not moved", target, from), host.ErrAlreadyExists)
	}
	content, err := v.files.read(from)
	if err != nil {
		return err
	}
	if err := v.files.remove(from); err != nil {
		return err
	}
	if _, err := v.files.write(target, content); err != nil {
		return err
	}
	v.rescan(from, target)
	return nil
}

// isInside reports whether a relative path stays inside the vault.
func isInside(filePath string) bool {
	return !strings.HasPrefix(filePath, "/") && path.Clean(filePath) == filePath && !strings.HasPrefix(filePath, "..")
}
