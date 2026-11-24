package identity

import (
	"context"
	"reflect"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

// GetPendingKeyCheck captures the pending restore of d and returns a function that reports whether its keys are zeroed.
func GetPendingKeyCheck(d Deps) func() bool {
	store, release := d.progress.restores.hold()
	restore := store.current
	release()
	return func() bool {
		_, release := d.progress.restores.hold()
		defer release()
		return reflect.DeepEqual(restore.recovery, identitylib.RecoveryKey{}) && reflect.DeepEqual(restore.device, identitylib.DeviceKey{})
	}
}

// SeededCreate creates an identity of any words: it first hands them to the store of generated words.
type SeededCreate struct {
	deps  Deps
	inner *CreateIdentity
}

func NewSeededCreate(d Deps) *SeededCreate {
	return &SeededCreate{deps: d, inner: NewCreateIdentity(d)}
}

func (s *SeededCreate) Execute(ctx context.Context, words []string) (identitydomain.Identity, error) {
	s.deps.progress.created.put(words)
	return s.inner.Execute(ctx, words)
}
