package onnx

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
)

func placeGraph(cache, graph string, files map[string]string) (string, error) {
	if len(files) == 1 {
		return files[graph], nil
	}
	if cache == "" {
		return "", errors.New("no cache directory for ONNX external data")
	}
	key := sha256.New()
	for _, name := range slices.Sorted(maps.Keys(files)) {
		fmt.Fprintf(key, "%s\x00%s\x00", name, files[name])
	}
	dir := filepath.Join(cache, "external-data", hex.EncodeToString(key.Sum(nil)))
	release, err := lockDir(dir)
	if err != nil {
		return "", err
	}
	err = linkGraphFiles(dir, files)
	return filepath.Join(dir, path.Base(graph)), errors.Join(err, release())
}

func linkGraphFiles(dir string, files map[string]string) error {
	for name, src := range files {
		dest := filepath.Join(dir, path.Base(name))
		if _, err := os.Stat(dest); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err := os.Link(src, dest); err != nil {
			if err := copyGraphFile(dir, src, dest); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyGraphFile(dir, src, dest string) (err error) {
	input, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, input.Close()) }()
	temp, err := os.CreateTemp(dir, ".part-*")
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(temp, input)
	if err = errors.Join(copyErr, temp.Close()); err == nil {
		err = os.Rename(temp.Name(), dest)
	}
	if err != nil {
		return errors.Join(err, os.Remove(temp.Name()))
	}
	return nil
}
