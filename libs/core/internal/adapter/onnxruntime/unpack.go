package onnxruntime

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
)

// unpack takes the library called name out of an archive and writes it at at.
//
// The library is looked for by name at any depth: the folder inside the archive
// is named after the release. The plain name inside is a link to the file
// carrying the version, and what is taken out is that file.
func unpack(archive, name, pos string) error {
	if strings.HasSuffix(archive, ".zip") {
		return unpackZip(archive, name, pos)
	}
	return unpackTgz(archive, name, pos)
}

func unpackTgz(archive, name, pos string) (err error) {
	file, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer closeAndJoin(&err, file)
	unzipped, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("%s: %w", archive, err)
	}
	defer closeAndJoin(&err, unzipped)

	target := tar.NewReader(unzipped)
	for {
		entry, err := target.Next()
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("%s holds no %s", archive, name)
		}
		if err != nil {
			return err
		}
		if entry.Typeflag != tar.TypeReg || !isLibrary(path.Base(entry.Name), name) {
			continue
		}
		return write(pos, target)
	}
}

func unpackZip(archive, name, pos string) (err error) {
	target, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer closeAndJoin(&err, target)
	for _, entry := range target.File {
		if !isLibrary(path.Base(entry.Name), name) {
			continue
		}
		file, err := entry.Open()
		if err != nil {
			return err
		}
		return errors.Join(write(pos, file), file.Close())
	}
	return fmt.Errorf("%s holds no %s", archive, name)
}

// write writes one file through a temporary name in the same directory, so an
// interrupted unpacking leaves nothing that looks finished.
func write(pos string, from io.Reader) error {
	part := pos + ".part"
	file, err := os.OpenFile(part, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(file, from); err != nil {
		return errors.Join(err, file.Close(), os.Remove(part))
	}
	if err := file.Close(); err != nil {
		return errors.Join(err, os.Remove(part))
	}
	return os.Rename(part, pos)
}

// isLibrary reports whether one name in an archive is the library.
//
// It is the exact name, that name with a version after it, or the name carrying
// a version before its extension, which is where a mac library keeps one. It is
// not one of the libraries published in the same archive: those are named for
// what they add, so they carry the same prefix and a different word.
func isLibrary(found, name string) bool {
	if found == name || strings.HasPrefix(found, name+".") {
		return true
	}
	ext := path.Ext(name)
	return strings.HasPrefix(found, strings.TrimSuffix(name, ext)+".") &&
		strings.HasSuffix(found, ext)
}
