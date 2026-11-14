package windowport

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// TabKinds finds the kind of tab each of the notes at some paths opens in.
type TabKinds interface {
	Execute(ctx context.Context, v domain.Vault, paths []string) (map[string]string, error)
}
