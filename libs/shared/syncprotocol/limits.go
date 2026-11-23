package syncprotocol

import "time"

const (
	MaxBlobSize       = 64 << 20
	MaxBatchRevisions = 500
	MaxReserveBlobs   = 100
	MaxBlobURLs       = 100
	MaxEncSize        = 16 << 10
	DefaultPageSize   = 200
	MaxPageSize       = 1000
)

const (
	WatchHeartbeat    = 20 * time.Second
	MinWatchHeartbeat = time.Second
	MaxWatchHeartbeat = 30 * time.Second
)
