package hostvault

import (
	"context"
)

// Scan updates the index for the files named.
func (f Files) Scan(ctx context.Context, paths []string) error {
	v, err := f.session.open()
	if err != nil {
		return err
	}
	f.session.notifyFileWritten()
	if f.Index == nil {
		return nil
	}
	return f.Index(ctx, v, paths)
}
