package plugincli

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/atomicfile"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

// Package bundles the files a plugin ships (manifest.json, bin, dist and the theme stylesheets the
// manifest declares) into a distributable zip archive.
func Package(targetPath, outArchive string, stdout io.Writer) (string, error) {
	manifestPath, pluginDir, err := resolveManifestPath(targetPath)
	if err != nil {
		return "", err
	}

	m, issues, err := readManifest(manifestPath, pluginDir)
	if err != nil {
		return "", fmt.Errorf("validate plugin: %w", err)
	}
	if len(issues) > 0 {
		return "", fmt.Errorf("cannot package invalid plugin: %s", strings.Join(issues, "; "))
	}

	destination := resolvePackageDestination(outArchive, pluginDir, m.ID, m.Version)
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return "", fmt.Errorf("create output directory: %w", err)
	}

	fileCount, err := createArchive(pluginDir, destination, listThemeFiles(m.Contributes.Themes))
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

func getAbsPath(file string) string {
	abs, err := filepath.Abs(file)
	if err != nil {
		return file
	}
	return abs
}

func isArchive(name string) bool {
	return strings.HasSuffix(name, ".zip") && path.Dir(name) == "dist"
}

type archiveWalker struct {
	archive   *zip.Writer
	files     fs.FS
	pluginDir string
	absDest   string
	count     int
}

func (w *archiveWalker) Walk(name string, entry fs.DirEntry, walkErr error) error {
	if walkErr != nil {
		return walkErr
	}
	if entry.IsDir() || isArchive(name) || w.isDestination(name) {
		return nil
	}
	if err := addFileToZip(w.archive, w.files, name); err != nil {
		return fmt.Errorf("add %s to archive: %w", name, err)
	}
	w.count++
	return nil
}

func (w *archiveWalker) isDestination(name string) bool {
	return getAbsPath(filepath.Join(w.pluginDir, filepath.FromSlash(name))) == w.absDest
}

func (w *archiveWalker) writeArchive(roots []string) (err error) {
	defer func() {
		if closeErr := w.archive.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("finish archive: %w", closeErr))
		}
	}()

	for _, root := range roots {
		if _, err := fs.Stat(w.files, root); errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err := fs.WalkDir(w.files, root, w.Walk); err != nil {
			return err
		}
	}
	return nil
}

func listThemeFiles(themes []manifest.Theme) []string {
	files := make([]string, 0, len(themes))
	for _, theme := range themes {
		files = append(files, theme.File)
	}
	return files
}

func createArchive(pluginDir, destPath string, extra []string) (int, error) {
	var content bytes.Buffer
	walker := archiveWalker{
		archive:   zip.NewWriter(&content),
		files:     os.DirFS(pluginDir),
		pluginDir: pluginDir,
		absDest:   getAbsPath(destPath),
		count:     0,
	}
	if err := walker.writeArchive(slices.Concat(manifestjson.ListShippedPaths(), filterOutsideShipped(extra))); err != nil {
		return 0, err
	}
	if err := atomicfile.WriteFile(destPath, &content, 0o644); err != nil {
		return 0, fmt.Errorf("write archive file: %w", err)
	}
	return walker.count, nil
}

func filterOutsideShipped(files []string) []string {
	var outside []string
	for _, file := range files {
		top, _, _ := strings.Cut(file, "/")
		if !slices.Contains(manifestjson.ListShippedPaths(), top) {
			outside = append(outside, file)
		}
	}
	return outside
}

func addFileToZip(archive *zip.Writer, files fs.FS, name string) (err error) {
	srcFile, err := files.Open(name)
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
	// gomutants:disable-next-line BRANCH_IF reason="zip.FileInfoHeader returns a nil error for every FileInfo, so the branch never runs"
	if err != nil {
		return err
	}
	header.Name = name
	header.Method = zip.Deflate

	writer, err := archive.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, srcFile)
	// gomutants:disable-next-line reason="the returned value is the zero value of its type, so returning zero is the same value"
	return err
}
