package plugin

import "context"

// applyChange runs change as an operation of the secret use cases does: it commits, then asks the
// host to refresh with the plugins that change names.
func applyChange(host *Host, ctx context.Context, change func() ([]string, error)) error {
	readers, err := change()
	if err != nil {
		return err
	}
	return host.Refresh(ctx, readers...)
}
