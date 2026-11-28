package main

import (
	"crypto/sha256"
	"encoding/binary"
)

// labelStream is the stream of bytes of one label: the same label gives the same bytes, so the same label gives the same account.
type labelStream struct {
	label   string
	counter uint64
	block   []byte
}

func newLabelStream(label string) *labelStream {
	return &labelStream{label: label, counter: 0, block: nil}
}

func (s *labelStream) Read(p []byte) (int, error) {
	for n := 0; n < len(p); {
		if len(s.block) == 0 {
			var counter [8]byte
			binary.BigEndian.PutUint64(counter[:], s.counter)
			sum := sha256.Sum256(append([]byte("numen sync test account "+s.label), counter[:]...))
			s.block, s.counter = sum[:], s.counter+1
		}
		copied := copy(p[n:], s.block)
		s.block = s.block[copied:]
		n += copied
	}
	return len(p), nil
}
