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
func unpack(archive, name, at string) error {
	if strings.HasSuffix(archive, ".zip") {
		return unpackZip(archive, name, at)
	}
	return unpackTgz(archive, name, at)
}

func unpackTgz(archive, name, at string) error {
	file, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer file.Close()
	unzipped, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("%s: %w", archive, err)
	}
	defer unzipped.Close()

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
		return write(at, target)
	}
}

func unpackZip(archive, name, at string) error {
	target, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer target.Close()
	for _, entry := range target.File {
		if !isLibrary(path.Base(entry.Name), name) {
			continue
		}
		file, err := entry.Open()
		if err != nil {
			return err
		}
		defer file.Close()
		return write(at, file)
	}
	return fmt.Errorf("%s holds no %s", archive, name)
}

// write puts one file down through a name beside it, so that an unpacking
// interrupted leaves nothing that looks finished.
func write(at string, from io.Reader) error {
	part := at + ".part"
	file, err := os.OpenFile(part, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(file, from); err != nil {
		file.Close()
		os.Remove(part)
		return err
	}
	if err := file.Close(); err != nil {
		os.Remove(part)
		return err
	}
	return os.Rename(part, at)
}

// isLibrary says whether one name in an archive is the library.
//
// It is the plain name, that name with a version after it, or the name carrying
// a version before its extension, which is where a mac library keeps one. It is
// not one of the libraries published beside it: those are named for what they
// add, so they carry the same prefix and a different word.
func isLibrary(found, name string) bool {
	if found == name || strings.HasPrefix(found, name+".") {
		return true
	}
	ext := path.Ext(name)
	return strings.HasPrefix(found, strings.TrimSuffix(name, ext)+".") &&
		strings.HasSuffix(found, ext)
}
