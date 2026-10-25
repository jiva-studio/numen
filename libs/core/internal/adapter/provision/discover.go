package provision

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// systemDirs are the folders a machine keeps programs and libraries in.
func getSystemDirs() []string {
	dirs := filepath.SplitList(os.Getenv("PATH"))
	return append(dirs,
		"/usr/local/bin", "/usr/bin", "/opt/homebrew/bin",
		"/usr/local/lib", "/usr/lib", "/usr/lib64", "/opt/homebrew/lib",
		"/usr/lib/"+runtime.GOARCH+"-linux-gnu", "/usr/lib/x86_64-linux-gnu", "/usr/lib/aarch64-linux-gnu",
	)
}

// discovery looks for a dependency the machine already holds.
type discovery struct {
	dirs []string
	// ldconfig lists the libraries the loader knows, as `ldconfig -p` prints them.
	ldconfig func() []byte
}

func newDiscovery() discovery {
	return discovery{dirs: getSystemDirs(), ldconfig: readLoaderCache}
}

func readLoaderCache() []byte {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "ldconfig", "-p").Output()
	if err != nil {
		return nil
	}
	return out
}

// find returns the path of the program or library the requirement names that
// this machine holds, or empty. Models and datasets are never on a machine
// before the host puts them there. A requirement that pins a checksum accepts
// only a file with those bytes.
func (d discovery) find(req pluginport.DependencyRequirement) string {
	found := d.locate(req)
	if found == "" || req.ChecksumSHA256 == "" {
		return found
	}
	if sum, err := hashFile(found); err != nil || sum != req.ChecksumSHA256 {
		return ""
	}
	return found
}

func (d discovery) locate(req pluginport.DependencyRequirement) string {
	if req.Kind != pluginport.DependencyKindBinary && req.Kind != pluginport.DependencyKindRuntime {
		return ""
	}
	name := req.ID
	if named := req.Metadata["binary"]; named != "" {
		name = named
	}
	if !isPlainName(name) {
		return ""
	}
	if found := findFile(d.dirs, getFileNames(name, req.Kind)); found != "" {
		return found
	}
	if req.Kind == pluginport.DependencyKindRuntime && d.ldconfig != nil {
		return findInLoaderCache(d.ldconfig(), name)
	}
	return ""
}

// hashFile returns the hex SHA-256 of the file.
func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// getFileNames are the file names a dependency goes by on this platform.
func getFileNames(name string, kind pluginport.DependencyKind) []string {
	if kind == pluginport.DependencyKindBinary {
		if runtime.GOOS == "windows" {
			return []string{name + ".exe", name}
		}
		return []string{name}
	}
	return []string{"lib" + name + ".so", "lib" + name + ".dylib", name + ".dll", "lib" + name + ".so.*"}
}

func findFile(dirs, names []string) string {
	for _, dir := range dirs {
		if found := findInDir(dir, names); found != "" {
			return found
		}
	}
	return ""
}

func findInDir(dir string, names []string) string {
	if dir == "" {
		return ""
	}
	for _, name := range names {
		matches, err := filepath.Glob(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		for _, match := range matches {
			if info, err := os.Stat(match); err == nil && info.Mode().IsRegular() {
				return match
			}
		}
	}
	return ""
}

// findInLoaderCache returns the path of the library in `ldconfig -p` output.
func findInLoaderCache(cache []byte, name string) string {
	prefix := "lib" + name + ".so"
	scanner := bufio.NewScanner(bytes.NewReader(cache))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		library, path, found := strings.Cut(line, "=>")
		if !found || !strings.HasPrefix(strings.TrimSpace(library), prefix) {
			continue
		}
		if path = strings.TrimSpace(path); path != "" {
			return path
		}
	}
	return ""
}
