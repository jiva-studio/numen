package settings

import "context"

// WriteSecret stores the key of a secret a plugin declares and records that the
// secret is stored. The key is kept in the key store and never in the file.
type WriteSecret struct {
	write func(ctx context.Context, pluginID, name, key string) error
}

func NewWriteSecret(write func(ctx context.Context, pluginID, name, key string) error) *WriteSecret {
	return &WriteSecret{write: write}
}

func (u *WriteSecret) Execute(ctx context.Context, pluginID, name, key string) error {
	return u.write(ctx, pluginID, name, key)
}

// DeleteSecret removes the key, if any, of a secret a plugin declares and clears
// the record that it is stored.
type DeleteSecret struct {
	delete func(ctx context.Context, pluginID, name string) error
}

func NewDeleteSecret(remove func(ctx context.Context, pluginID, name string) error) *DeleteSecret {
	return &DeleteSecret{delete: remove}
}

func (u *DeleteSecret) Execute(ctx context.Context, pluginID, name string) error {
	return u.delete(ctx, pluginID, name)
}
