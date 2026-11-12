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

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

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
	dirs     []string
	ldconfig func() []byte
}

func newDiscovery() discovery {
	return discovery{dirs: getSystemDirs(), ldconfig: readLoaderCache}
}

func readLoaderCache() []byte {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, ldconfig := range getLoaderCachePrograms() {
		if _, err := os.Stat(ldconfig); err != nil {
			continue
		}
		out, err := exec.CommandContext(ctx, ldconfig, "-p").Output()
		if err != nil {
			return nil
		}
		return out
	}
	return nil
}

func getLoaderCachePrograms() []string {
	return []string{"/sbin/ldconfig", "/usr/sbin/ldconfig", "/usr/bin/ldconfig", "/bin/ldconfig"}
}

func (d discovery) find(req manifest.DependencyRequirement) string {
	found := d.locate(req)
	if found == "" || req.ChecksumSHA256 == "" {
		return found
	}
	if sum, err := hashFile(found); err != nil || sum != req.ChecksumSHA256 {
		return ""
	}
	return found
}

func (d discovery) locate(req manifest.DependencyRequirement) string {
	if req.Kind != manifest.DependencyKindBinary && req.Kind != manifest.DependencyKindRuntime {
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
	if req.Kind == manifest.DependencyKindRuntime && d.ldconfig != nil {
		return findInLoaderCache(d.ldconfig(), name)
	}
	return ""
}

func hashFile(path string) (_ string, err error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer closeAndJoin(&err, file)
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func getFileNames(name string, kind manifest.DependencyKind) []string {
	if kind == manifest.DependencyKindBinary {
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
