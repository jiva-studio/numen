package syncprotocol

import "time"

const (
	MaxBlobSize       = 64 << 20
	MaxBatchRevisions = 5000
	MaxReserveBlobs   = 100
	MaxBlobURLs       = 100
	MaxEncSize        = 16 << 10
	DefaultPageSize   = 200
	MaxPageSize       = 1000
)

const (
	MinPartSize      = 5 << 20
	MaxPartSize      = 5 << 30
	MaxParts         = 10000
	DefaultPartSize  = 16 << 20
	MaxPartURLs      = 100
	MaxMultipartSize = 5<<40 - 5<<20
)

const (
	WatchHeartbeat    = 20 * time.Second
	MinWatchHeartbeat = time.Second
	MaxWatchHeartbeat = 30 * time.Second
)

const (
	MaxLinkMessage   = 256 << 10
	LinkWaitMax      = 20
	MaxChainBytes    = 192 << 10
	MaxActiveDevices = 32
)
