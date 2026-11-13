package vector

import "github.com/jiva-studio/numen/libs/note/domain"

// QuantizedInt8 names the quantization of a stored vector's bytes: one signed
// byte per dimension, at Int8Scale. It is recorded because a blob does not say
// its contents.
const QuantizedInt8 = "int8"

// Int8Scale is the value that a stored byte of 127 represents. Bytes on two
// scales lie on different grids, so the recipe records the scale.
// One value serves every vector width: a wider vector uses fewer of the 127
// levels, and the cosine error does not grow with the width.
// The value 0.4 sets the headroom: a larger component of a unit vector is clamped.
const Int8Scale = 0.4

// Stored is one chunk's vector, in both stored representations.
// Value is the copy used for reranking, quantized as Kind specifies. Coarse is
// one bit per dimension, used by the coarse pass. Hash is the digest of the
// text the source read; the vector is stored under it after the chunk that
// referenced it is gone. Recipe is the key the vector is kept under.
type Stored struct {
	ChunkID domain.ChunkID
	Hash    []byte
	Recipe  string
	Kind    string
	Value   []byte
	Coarse  []byte
}

// ChunkCursor is a place in a walk over the chunks of one vault. The index
// writes it and reads it back, and the empty cursor is the beginning: a caller
// carries it from one answer to the next and reads nothing out of it.
type ChunkCursor string
