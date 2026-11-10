package plugin

import "context"

func applyChange(host *Host, ctx context.Context, change func() ([]string, error)) error {
	readers, err := change()
	if err != nil {
		return err
	}
	return host.Config.Refresh(ctx, readers...)
}
