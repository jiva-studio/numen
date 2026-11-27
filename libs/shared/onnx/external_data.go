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

// PlaceModelFiles returns the path of the graph among files, which maps the repository-relative name
// of each file of one model to its local path. A graph with external data is linked, with its data,
// into a folder of cache under the relative names the graph encodes.
func PlaceModelFiles(cache, graph string, files map[string]string) (string, error) {
	if len(files) == 1 {
		return files[graph], nil
	}
	if cache == "" {
		return "", errors.New("no cache directory for ONNX external data")
	}
	key := sha256.New()
	for _, name := range slices.Sorted(maps.Keys(files)) {
		if _, err := fmt.Fprintf(key, "%s\x00%s\x00", name, files[name]); err != nil {
			return "", err
		}
	}
	dir := filepath.Join(cache, "external-data", hex.EncodeToString(key.Sum(nil)))
	release, err := lockDir(dir)
	if err != nil {
		return "", err
	}
	err = linkGraphFiles(dir, path.Dir(graph), files)
	return filepath.Join(dir, path.Base(graph)), errors.Join(err, release())
}

func linkGraphFiles(dir, folder string, files map[string]string) error {
	for name, src := range files {
		rel, err := filepath.Rel(filepath.FromSlash(folder), filepath.FromSlash(name))
		if err != nil || !filepath.IsLocal(rel) {
			return fmt.Errorf("invalid external model file %q", name)
		}
		dest := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
			return err
		}
		if _, err := os.Stat(dest); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err := os.Link(src, dest); err != nil {
			if err := copyGraphFile(filepath.Dir(dest), src, dest); err != nil {
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
