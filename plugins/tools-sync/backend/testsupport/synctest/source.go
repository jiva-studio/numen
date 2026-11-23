package synctest

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

const blockSize = sha256.Size

// Source is a file of a size and a seed that is never held whole: any range of it is made when it is read, as the
// host reads a file by ranges, so a test sends a file larger than it keeps in memory.
type Source struct {
	stat domain.FileStat
	seed uint64
}

// NewSource returns the file at the path, of the size, whose bytes the seed fixes, last changed at modTime.
func NewSource(path string, size int64, seed uint64, modTime time.Time) *Source {
	s := &Source{stat: domain.FileStat{Path: path, Kind: "", Size: size, ModTime: modTime, FileID: 0, Hash: "", Version: 0}, seed: seed}
	hasher := sha256.New()
	const chunk = 1 << 20
	for start := int64(0); start < size; start += chunk {
		hasher.Write(s.GetBytes(start, min(chunk, size-start)))
	}
	s.stat.Hash = hex.EncodeToString(hasher.Sum(nil))
	return s
}

// GetStat returns what the host says of the file, with its hash.
func (s *Source) GetStat() domain.FileStat { return s.stat }

// GetPath returns the path of the file in the vault.
func (s *Source) GetPath() string { return s.stat.Path }

// GetBytes returns the bytes from start, at most length of them.
func (s *Source) GetBytes(start, length int64) []byte {
	end := min(start+length, s.stat.Size)
	if start >= end {
		return []byte{}
	}
	out := make([]byte, 0, end-start)
	for block := start / blockSize; int64(len(out)) < end-start; block++ {
		var input [16]byte
		binary.BigEndian.PutUint64(input[:8], s.seed)
		binary.BigEndian.PutUint64(input[8:], uint64(block))
		sum := sha256.Sum256(input[:])
		from := int64(0)
		if block == start/blockSize {
			from = start % blockSize
		}
		out = append(out, sum[from:min(blockSize, from+end-start-int64(len(out)))]...)
	}
	return out
}

// ReadBytes answers a ranged read of the file as the host does: at most host.MaxReadBytes at once.
func (s *Source) ReadBytes(_ context.Context, start, length int64) (host.ByteRange, error) {
	if start < 0 || length < 1 || length > host.MaxReadBytes {
		return host.ByteRange{}, fmt.Errorf("read %d bytes at %d: %w", length, start, host.ErrInvalidRange)
	}
	return host.ByteRange{Content: s.GetBytes(start, length), Stat: s.stat}, nil
}
