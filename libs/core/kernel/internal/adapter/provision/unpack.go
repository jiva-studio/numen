package provision

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/libs/shared/atomicfile"
)

func unpackMember(archive, member, sum, dir string) (string, error) {
	if strings.HasSuffix(strings.ToLower(archive), ".zip") {
		return unpackZipMember(archive, member, sum, dir)
	}
	return unpackTarMember(archive, member, sum, dir)
}

func unpackTarMember(archive, member, sum, dir string) (_ string, err error) {
	file, err := os.Open(archive)
	if err != nil {
		return "", err
	}
	defer closeAndJoin(&err, file)
	unzipped, err := gzip.NewReader(file)
	if err != nil {
		return "", fmt.Errorf("%s: %w", archive, err)
	}
	defer closeAndJoin(&err, unzipped)
	entries := tar.NewReader(unzipped)
	for {
		entry, err := entries.Next()
		if errors.Is(err, io.EOF) {
			return "", fmt.Errorf("%s holds no %s", archive, member)
		}
		if err != nil {
			return "", err
		}
		if entry.Typeflag == tar.TypeReg && isMember(entry.Name, member) {
			return writeMember(dir, entry.Name, entries, sum)
		}
	}
}

func unpackZipMember(archive, member, sum, dir string) (_ string, err error) {
	entries, err := zip.OpenReader(archive)
	if err != nil {
		return "", err
	}
	defer closeAndJoin(&err, entries)
	for _, entry := range entries.File {
		if !entry.Mode().IsRegular() || !isMember(entry.Name, member) {
			continue
		}
		file, err := entry.Open()
		if err != nil {
			return "", err
		}
		pos, err := writeMember(dir, entry.Name, file, sum)
		return pos, errors.Join(err, file.Close())
	}
	return "", fmt.Errorf("%s holds no %s", archive, member)
}

func isMember(name, member string) bool {
	if !strings.Contains(member, "/") {
		name = path.Base(name)
	}
	matched, err := path.Match(member, strings.TrimPrefix(name, "./"))
	return err == nil && matched
}

func writeMember(dir, name string, from io.Reader, sum string) (string, error) {
	base := cleanName(path.Base(name))
	if isReserved(base) {
		return "", fmt.Errorf("%q is a name the folder keeps for itself", base)
	}
	pos := filepath.Join(dir, base)
	checked := &memberReader{source: from, digest: sha256.New(), wanted: sum}
	if err := atomicfile.WriteFile(pos, checked, 0o755); err != nil {
		return "", fmt.Errorf("unpacking %s: %w", name, err)
	}
	return pos, nil
}

// memberReader hands on the bytes of an archive member and, at their end, refuses a member that does
// not hold the pinned sum, so the write that reads it never puts the file in place.
type memberReader struct {
	source io.Reader
	digest hash.Hash
	wanted string
}

func (r *memberReader) Read(chunk []byte) (int, error) {
	n, err := r.source.Read(chunk)
	if _, writeErr := r.digest.Write(chunk[:n]); writeErr != nil {
		return n, writeErr
	}
	if errors.Is(err, io.EOF) {
		if found := hex.EncodeToString(r.digest.Sum(nil)); found != r.wanted {
			return n, fmt.Errorf("%w: the member holds %s, and %s is pinned", ErrChecksum, found, r.wanted)
		}
	}
	return n, err
}
