package plugincli

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifestjson"
)

// Package bundles the files a plugin ships (manifest.json, bin, dist and the theme stylesheets the
// manifest declares) into a distributable zip archive.
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

	// gomutants:disable-next-line
	manifestData, err := os.ReadFile(manifestPath)
	// gomutants:disable-next-line
	if err != nil {
		return "", fmt.Errorf("read manifest: %w", err)
	}

	// gomutants:disable-next-line
	m, err := manifestjson.Decode(manifestData)
	// gomutants:disable-next-line
	if err != nil {
		return "", fmt.Errorf("decode manifest: %w", err)
	}

	destination := resolvePackageDestination(outArchive, pluginDir, m.ID, m.Version)
	// gomutants:disable-next-line
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return "", fmt.Errorf("create output directory: %w", err)
	}

	fileCount, err := createArchive(pluginDir, destination, listThemeFiles(m.Contributes.Themes))
	// gomutants:disable-next-line
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
	// gomutants:disable-next-line
	if err != nil {
		return path == absDest
	}
	return absCurrent == absDest
}

func getShippedPaths() []string { return []string{"manifest.json", "bin", "dist"} }

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
	// gomutants:disable-next-line
	if walkErr != nil {
		return walkErr
	}
	if entry.IsDir() || isSamePath(path, w.absDest) || isArchive(w.pluginDir, path) {
		return nil
	}

	relPath, relErr := filepath.Rel(w.pluginDir, path)
	// gomutants:disable-next-line
	if relErr != nil {
		return relErr
	}

	// gomutants:disable-next-line
	if err := addFileToZip(w.archive, path, relPath); err != nil {
		return fmt.Errorf("add %s to archive: %w", relPath, err)
	}
	w.count++
	return nil
}

func listThemeFiles(themes []manifest.Theme) []string {
	files := make([]string, 0, len(themes))
	for _, theme := range themes {
		files = append(files, filepath.FromSlash(theme.File))
	}
	return files
}

func createArchive(pluginDir, destPath string, extra []string) (count int, err error) {
	absDest, err := filepath.Abs(destPath)
	// gomutants:disable-next-line
	if err != nil {
		absDest = destPath
	}

	zipFile, err := os.Create(destPath)
	// gomutants:disable-next-line
	if err != nil {
		return 0, fmt.Errorf("create archive file: %w", err)
	}
	defer func() {
		// gomutants:disable-next-line
		if closeErr := zipFile.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close archive file: %w", closeErr))
		}
	}()

	archive := zip.NewWriter(zipFile)
	defer func() {
		// gomutants:disable-next-line
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

	for _, shipped := range slices.Concat(getShippedPaths(), filterOutsideShipped(extra)) {
		root := filepath.Join(pluginDir, shipped)
		if _, err := os.Stat(root); errors.Is(err, fs.ErrNotExist) {
			continue
		}
		// gomutants:disable-next-line
		if err := filepath.WalkDir(root, walker.Walk); err != nil {
			return 0, err
		}
	}
	return walker.count, nil
}

func filterOutsideShipped(files []string) []string {
	var outside []string
	for _, file := range files {
		top, _, _ := strings.Cut(file, string(filepath.Separator))
		if !slices.Contains(getShippedPaths(), top) {
			outside = append(outside, file)
		}
	}
	return outside
}

func addFileToZip(archive *zip.Writer, srcPath, zipPath string) (err error) {
	srcFile, err := os.Open(srcPath)
	// gomutants:disable-next-line
	if err != nil {
		return err
	}
	defer func() {
		// gomutants:disable-next-line
		if closeErr := srcFile.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
	}()

	info, err := srcFile.Stat()
	// gomutants:disable-next-line
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	// gomutants:disable-next-line
	if err != nil {
		return err
	}
	header.Name = filepath.ToSlash(zipPath)
	header.Method = zip.Deflate

	writer, err := archive.CreateHeader(header)
	// gomutants:disable-next-line
	if err != nil {
		return err
	}

	// gomutants:disable-next-line
	_, err = io.Copy(writer, srcFile)
	// gomutants:disable-next-line
	return err
}
