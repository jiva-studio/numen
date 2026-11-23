package note

import (
	"context"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/frontmatter"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// TabKindHeads reads the type and the frontmatter of notes from the index, never from their files.
type TabKindHeads interface {
	// GetHeads returns the heads of the notes at the paths that the index holds; a path with no
	// indexed note is absent from the result.
	GetHeads(ctx context.Context, vaultID domain.VaultID, paths []string) (map[string]frontmatter.Head, error)
}

// TabKinds finds the kind of tab each of the notes at some paths opens in, from the type and the
// frontmatter the index holds, in one question and without touching a file.
type TabKinds struct {
	heads  TabKindHeads
	choice *plugin.TabKindChoice
}

// NewTabKinds makes the tab kinds over the heads of the index, chosen by choice. Without heads no note has a kind.
func NewTabKinds(heads TabKindHeads, choice *plugin.TabKindChoice) *TabKinds {
	return &TabKinds{heads: heads, choice: choice}
}

// ListTabKinds answers by path, and leaves out a path with no kind of tab: a path the index holds no note
// at, as a note whose file cannot be read, has none.
func (t *TabKinds) ListTabKinds(ctx context.Context, v domain.Vault, paths []string) (map[string]string, error) {
	if len(paths) == 0 || t.heads == nil {
		return map[string]string{}, nil
	}
	heads, err := t.heads.GetHeads(ctx, v.ID, paths)
	if err != nil {
		return nil, err
	}
	return t.choice.ChooseMany(heads), nil
}
