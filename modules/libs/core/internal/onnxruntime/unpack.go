package onnxruntime

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// unpack takes the library out of an archive, once, and says where it now is.
//
// The archive carries the sum this release publishes before anything is taken
// out of it, and the library carries its own before it is left in the cache. A
// library already there carrying another sum is written over.
//
// The library is looked for by name at any depth, because the folder inside the
// archive is named after the release. The plain name inside is a link to the
// file carrying the version, so what is taken out is the file: a link copied out
// of an archive points at nothing.
func unpack(archive, name string, found release) (string, error) {
	at := filepath.Join(filepath.Dir(archive), name)
	if verify(at, found.library) == nil {
		return at, nil
	}
	if err := verify(archive, found.sum); err != nil {
		return "", err
	}
	var err error
	if strings.HasSuffix(archive, ".zip") {
		err = fromZip(archive, name, at)
	} else {
		err = fromTgz(archive, name, at)
	}
	if err != nil {
		return "", err
	}
	if err := verify(at, found.library); err != nil {
		os.Remove(at)
		return "", err
	}
	return at, nil
}

// verify says whether one file carries the sum published for it.
func verify(at, sum string) error {
	info, err := os.Stat(at)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a file", at)
	}
	file, err := os.Open(at)
	if err != nil {
		return err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	if found := hex.EncodeToString(hash.Sum(nil)); found != sum {
		return fmt.Errorf("%s carries %s, and %s is published", at, found, sum)
	}
	return nil
}

func fromTgz(archive, name, at string) error {
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

	held := tar.NewReader(unzipped)
	for {
		entry, err := held.Next()
		if err == io.EOF {
			return fmt.Errorf("%s holds no %s", archive, name)
		}
		if err != nil {
			return err
		}
		if entry.Typeflag != tar.TypeReg || !isLibrary(path.Base(entry.Name), name) {
			continue
		}
		return write(at, held)
	}
}

func fromZip(archive, name, at string) error {
	held, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer held.Close()
	for _, entry := range held.File {
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
	return ext != "" &&
		strings.HasPrefix(found, strings.TrimSuffix(name, ext)+".") &&
		strings.HasSuffix(found, ext)
}
