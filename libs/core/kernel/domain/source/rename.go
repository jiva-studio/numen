package source

import (
	"slices"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// FindRemoved returns the sources other than notes that getUnder reports at and under the paths,
// with the content hash each was read with. A source never read has no hash and matches nothing.
func FindRemoved(
	paths []string,
	getUnder func(path string) ([]domain.FileStat, error),
	getText func(path string) (Text, bool, error),
) ([]Text, error) {
	var out []Text
	for _, path := range paths {
		under, err := getUnder(path)
		if err != nil {
			return nil, err
		}
		for _, ref := range under {
			if ref.Kind == domain.KindNote {
				continue
			}
			text, ok, err := getText(ref.Path)
			if err != nil {
				return nil, err
			}
			if ok && text.ContentHash != "" {
				out = append(out, text)
			}
		}
	}
	return out, nil
}

// MoveRenamed moves each removed source to the path among added whose file holds the same content,
// and returns the paths it moved from. getStat and getHash read the added file; getHash is called
// only for a file of the same size as a removed source.
func MoveRenamed(
	removed []Text,
	added []string,
	getStat func(path string) (domain.FileStat, error),
	getHash func(path string) (string, error),
	move func(from, to string) error,
) ([]string, error) {
	var moved []string
	for _, path := range added {
		if len(removed) == 0 {
			break
		}
		i := findSameContent(removed, path, getStat, getHash)
		if i < 0 {
			continue
		}
		if err := move(removed[i].Path, path); err != nil {
			return moved, err
		}
		moved = append(moved, removed[i].Path)
		removed = slices.Delete(removed, i, i+1)
	}
	return moved, nil
}

func findSameContent(
	removed []Text,
	path string,
	getStat func(path string) (domain.FileStat, error),
	getHash func(path string) (string, error),
) int {
	ref, err := getStat(path)
	if err != nil || !slices.ContainsFunc(removed, func(old Text) bool { return old.Size == ref.Size }) {
		return -1
	}
	hash, err := getHash(path)
	if err != nil {
		return -1
	}
	return slices.IndexFunc(removed, func(old Text) bool { return old.ContentHash == hash })
}
