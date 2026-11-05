package container

import "github.com/jiva-studio/numen/libs/core/internal/chunking"

// ChunkTokens is the chunk size the embedding plugin reports, shared by every copy of Config.
type ChunkTokens = chunking.Tokens
