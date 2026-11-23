package syncprotocol

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
	MaxLinkMessage   = 256 << 10
	LinkWaitMax      = 20
	MaxChainBytes    = 192 << 10
	MaxActiveDevices = 32
)
