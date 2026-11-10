package file

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// ListTabKinds is the kind of tab each of the files opens in.
func (h *Folders) ListTabKinds(ctx context.Context, v domain.Vault, paths []string) (map[string]string, error) {
	return h.listTabKinds(ctx, v, paths)
}

func (h *Folders) listTabKinds(ctx context.Context, v domain.Vault, paths []string) (map[string]string, error) {
	kinds := map[string]string{}
	if h.getFileKinds == nil || h.chooseTabKind == nil || len(paths) == 0 {
		return kinds, nil
	}
	listing, err := h.getFileKinds.Execute(ctx, v, paths)
	if err != nil {
		return nil, err
	}
	for path, properties := range listing.Properties {
		if kind := h.chooseTabKind.Execute(path, listing.Types[path], properties); kind != "" {
			kinds[path] = kind
		}
	}
	return kinds, nil
}
