package port

import "context"

// ScheduleStore is where what a replay of the answers worked out about one
// vault is kept between launches.
//
// It is a cache and belongs to the installation, never to the vault: what is in
// it is computed from the answers, and the answers are the vault's. It is also
// the one thing here rewritten whole, and a file rewritten whole inside a
// folder somebody synchronises is a file that conflicts.
//
// A vault with nothing kept for it gets fs.ErrNotExist, which is the answer that
// nothing has been worked out yet.
type ScheduleStore interface {
	Read(ctx context.Context, vaultID string) ([]byte, error)
	Write(ctx context.Context, vaultID string, content []byte) error
}
