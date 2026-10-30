package plugincli

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

// Package bundles the files a plugin ships (manifest.json, bin and dist) into a distributable zip archive.
func Package(targetPath, outArchive string, stdout io.Writer) (string, error) {
	manifestPath, pluginDir, err := resolveManifestPath(targetPath)
	if err != nil {
		return "", err
	}

	issues, err := Validate(targetPath)
	if err != nil {
		return "", fmt.Errorf("validate plugin: %w", err)
	}
	if len(issues) > 0 {
		return "", fmt.Errorf("cannot package invalid plugin: %s", strings.Join(issues, "; "))
	}

	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", fmt.Errorf("read manifest: %w", err)
	}

	var m pluginport.PluginManifest
	if err := json.Unmarshal(manifestData, &m); err != nil {
		return "", fmt.Errorf("decode manifest: %w", err)
	}

	destination := resolvePackageDestination(outArchive, pluginDir, m.ID, m.Version)
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return "", fmt.Errorf("create output directory: %w", err)
	}

	fileCount, err := createArchive(pluginDir, destination)
	if err != nil {
		return "", err
	}

	if _, err := fmt.Fprintf(stdout, "Bundled %d files into %s\n", fileCount, destination); err != nil {
		return "", fmt.Errorf("report bundle: %w", err)
	}
	return destination, nil
}

func resolvePackageDestination(customOut, pluginDir, id, version string) string {
	if customOut != "" {
		return customOut
	}
	return filepath.Join(pluginDir, "dist", fmt.Sprintf("%s-%s.zip", id, version))
}

func isSamePath(path, absDest string) bool {
	absCurrent, err := filepath.Abs(path)
	if err != nil {
		return path == absDest
	}
	return absCurrent == absDest
}

// shippedPaths are the entries of a plugin directory an archive holds.
var shippedPaths = []string{"manifest.json", "bin", "dist"}

// isArchive reports whether the file is an archive in the top level of the dist folder.
func isArchive(pluginDir, path string) bool {
	return strings.HasSuffix(path, ".zip") && filepath.Dir(path) == filepath.Join(pluginDir, "dist")
}

type archiveWalker struct {
	archive   *zip.Writer
	pluginDir string
	absDest   string
	count     int
}

func (w *archiveWalker) Walk(path string, entry fs.DirEntry, walkErr error) error {
	if walkErr != nil {
		return walkErr
	}
	if entry.IsDir() || isSamePath(path, w.absDest) || isArchive(w.pluginDir, path) {
		return nil
	}

	relPath, relErr := filepath.Rel(w.pluginDir, path)
	if relErr != nil {
		return relErr
	}

	if err := addFileToZip(w.archive, path, relPath); err != nil {
		return fmt.Errorf("add %s to archive: %w", relPath, err)
	}
	w.count++
	return nil
}

func createArchive(pluginDir, destPath string) (count int, err error) {
	absDest, err := filepath.Abs(destPath)
	if err != nil {
		absDest = destPath
	}

	zipFile, err := os.Create(destPath)
	if err != nil {
		return 0, fmt.Errorf("create archive file: %w", err)
	}
	defer func() {
		if closeErr := zipFile.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close archive file: %w", closeErr))
		}
	}()

	archive := zip.NewWriter(zipFile)
	defer func() {
		if closeErr := archive.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("finish archive: %w", closeErr))
		}
	}()

	walker := archiveWalker{
		archive:   archive,
		pluginDir: pluginDir,
		absDest:   absDest,
		count:     0,
	}

	for _, shipped := range shippedPaths {
		root := filepath.Join(pluginDir, shipped)
		if _, err := os.Stat(root); errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err := filepath.WalkDir(root, walker.Walk); err != nil {
			return 0, err
		}
	}
	return walker.count, nil
}

func addFileToZip(archive *zip.Writer, srcPath, zipPath string) (err error) {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := srcFile.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
	}()

	info, err := srcFile.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = filepath.ToSlash(zipPath)
	header.Method = zip.Deflate

	writer, err := archive.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, srcFile)
	return err
}
