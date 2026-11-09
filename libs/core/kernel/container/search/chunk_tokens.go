// Package search wires search, search policy, vectors and chunking.
package search

import "github.com/jiva-studio/numen/libs/core/kernel/usecase/note"

// ChunkTokens is the chunk size the embedding plugin reports, shared by every copy of Config.
type ChunkTokens = note.ChunkTokens
