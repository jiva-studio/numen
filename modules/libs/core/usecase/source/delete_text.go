package source

import (
	"context"
	"errors"
	"io/fs"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// DeleteText throws away the text downloaded from a url's address. The url stands
// as it was, pointing where it points, and the copy downloaded for it is
// untouched.
func (u ImportURL) DeleteText(ctx context.Context, v domain.Vault, path string) error {
	at, _, store, err := u.pointed(ctx, v, path)
	if err != nil {
		return err
	}
	if err := forgotten(ctx, store, text.Fingerprint([]byte(string(at)))); err != nil {
		return err
	}
	return u.cut(ctx, v, path)
}

// forgotten takes away everything ever downloaded for an address.
func forgotten(ctx context.Context, store port.DerivedStore, hash string) error {
	for _, name := range text.AddressTexts(hash) {
		if err := store.Remove(ctx, name); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	return nil
}
