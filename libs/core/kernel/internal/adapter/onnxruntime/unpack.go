package onnxruntime

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/atomicfile"
)

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

func write(pos string, from io.Reader) error {
	return atomicfile.WriteFile(context.Background(), pos, from, 0o755)
}

func isLibrary(found, name string) bool {
	if found == name || strings.HasPrefix(found, name+".") {
		return true
	}
	ext := path.Ext(name)
	return strings.HasPrefix(found, strings.TrimSuffix(name, ext)+".") &&
		strings.HasSuffix(found, ext)
}
