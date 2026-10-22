package onnxruntime

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
)

// libraryEnvs name the library a person points the host at. The first set one
// is used as given.
var libraryEnvs = []string{"NUMEN_ONNX_RUNTIME_LIB", "NUMEN_ONNX_RUNTIME", "NUMEN_OCR_RUNTIME"}

// getLibraryName returns what the library is called on an operating system.
func getLibraryName(goos string) string {
	switch goos {
	case "darwin":
		return "libonnxruntime.dylib"
	case "windows":
		return "onnxruntime.dll"
	}
	return "libonnxruntime.so"
}

// getEnvLibrary returns the library a person named in the environment, or empty.
func getEnvLibrary() string {
	for _, name := range libraryEnvs {
		if pos := os.Getenv(name); pos != "" {
			return pos
		}
	}
	return ""
}

// getMachineDirs returns where this machine keeps shared libraries.
func getMachineDirs() []string {
	out := filepath.SplitList(os.Getenv("LD_LIBRARY_PATH"))
	if home, err := os.UserHomeDir(); err == nil {
		out = append(out, filepath.Join(home, ".nix-profile", "lib"))
	}
	return append(out,
		"/run/current-system/sw/lib",
		"/usr/local/lib",
		"/usr/lib64",
		"/usr/lib/"+runtime.GOARCH+"-linux-gnu",
		"/usr/lib/x86_64-linux-gnu",
		"/usr/lib/aarch64-linux-gnu",
		"/usr/lib",
		"/lib64",
		"/lib",
	)
}

// findMatching returns the regular files one pattern names.
func findMatching(pattern string) []string {
	found, err := filepath.Glob(pattern)
	if err != nil {
		return nil
	}
	var out []string
	for _, item := range found {
		if info, err := os.Stat(item); err == nil && info.Mode().IsRegular() {
			out = append(out, item)
		}
	}
	return out
}

// getBesideDirs returns the folders a library may stand beside: the plugin, its
// bin folder and the host, each with its models folder.
func getBesideDirs(pluginDir string) []string {
	dirs := []string{pluginDir, filepath.Join(pluginDir, "bin")}
	if self, err := os.Executable(); err == nil {
		dirs = append(dirs, filepath.Dir(self))
	}
	var out []string
	for _, dir := range dirs {
		if dir != "" {
			out = append(out, dir, filepath.Join(dir, "models"))
		}
	}
	return out
}

// getCandidates returns every library this machine holds without fetching one,
// in the order they are tried: beside the plugin or the host, where an earlier
// run unpacked one (when it holds the pinned sum), in the folders shared
// libraries live in, in the package store, and in the folders the platform
// loader searches. A machine list that is not nil replaces the folders shared
// libraries live in, and the package store and the loader folders are then not
// searched.
func getCandidates(pluginDir, home, librarySum string, machine []string) []string {
	return getCandidatesInStore(pluginDir, home, librarySum, machine, filepath.Join("/nix/store", "*-onnxruntime-*", "lib"))
}

// getCandidatesInStore searches the machine and a package store pattern.
func getCandidatesInStore(pluginDir, home, librarySum string, machine []string, storePattern string) []string {
	name := getLibraryName(runtime.GOOS)
	var out []string
	for _, dir := range getBesideDirs(pluginDir) {
		if pos := filepath.Join(dir, name); isFile(pos) {
			out = append(out, pos)
		}
	}
	if offset := filepath.Join(home, name); home != "" && isFile(offset) && verifyFile(offset, librarySum) == nil {
		out = append(out, offset)
	}
	isDefault := machine == nil
	if isDefault {
		machine = getMachineDirs()
	}
	out = append(out, findInDirs(machine, name)...)
	if isDefault {
		out = append(out, findMatching(filepath.Join(storePattern, name+"*"))...)
		out = append(out, findInDirs(getLoaderDirs(runtime.GOOS), name)...)
	}
	return removeDuplicates(out)
}

// findInDirs returns every file named for the library in the folders. A folder
// that is not an absolute path is not searched.
func findInDirs(dirs []string, name string) []string {
	var out []string
	for _, dir := range dirs {
		if filepath.IsAbs(dir) {
			out = append(out, findMatching(filepath.Join(dir, name+"*"))...)
		}
	}
	return out
}

// getLoaderDirs returns the further folders the platform loader searches for a
// library named without a folder: the folders /etc/ld.so.conf names, the dyld
// search paths on darwin, and System32 and PATH on windows.
func getLoaderDirs(goos string) []string {
	switch goos {
	case "darwin":
		out := filepath.SplitList(os.Getenv("DYLD_LIBRARY_PATH"))
		out = append(out, filepath.SplitList(os.Getenv("DYLD_FALLBACK_LIBRARY_PATH"))...)
		if home, err := os.UserHomeDir(); err == nil {
			out = append(out, filepath.Join(home, "lib"))
		}
		return out
	case "windows":
		var out []string
		if root := os.Getenv("SystemRoot"); root != "" {
			out = append(out, filepath.Join(root, "System32"), root)
		}
		return append(out, filepath.SplitList(os.Getenv("PATH"))...)
	}
	return readLoaderConfig("/etc/ld.so.conf", map[string]bool{})
}

// readLoaderConfig returns the folders a loader configuration file names,
// following its include lines. seen holds the files already read.
func readLoaderConfig(pos string, visited map[string]bool) []string {
	if visited[pos] {
		return nil
	}
	visited[pos] = true
	raw, err := os.ReadFile(pos)
	if err != nil {
		return nil
	}
	var out []string
	for _, line := range strings.Split(string(raw), "\n") {
		line, _, _ = strings.Cut(line, "#")
		fields := strings.FieldsFunc(line, isConfigSeparator)
		switch {
		case len(fields) == 0, fields[0] == "hwcap":
		case fields[0] == "include":
			for _, pattern := range fields[1:] {
				out = append(out, readIncludedConfig(pos, pattern, visited)...)
			}
		default:
			out = append(out, fields...)
		}
	}
	return out
}

// readIncludedConfig returns the folders the files one include pattern names
// hold. A relative pattern stands beside the file that includes it.
func readIncludedConfig(from, pattern string, seenCount map[string]bool) []string {
	if !filepath.IsAbs(pattern) {
		pattern = filepath.Join(filepath.Dir(from), pattern)
	}
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil
	}
	var out []string
	for _, match := range matches {
		out = append(out, readLoaderConfig(match, seenCount)...)
	}
	return out
}

// isConfigSeparator reports whether r separates the folders on one line of a
// loader configuration file.
func isConfigSeparator(r rune) bool {
	return r == ' ' || r == '\t' || r == '\r' || r == ',' || r == ':'
}

// removeDuplicates returns the paths with each file kept once, at the first
// path that names it.
func removeDuplicates(paths []string) []string {
	observed := make(map[string]bool, len(paths))
	var out []string
	for _, item := range paths {
		resolved, err := filepath.EvalSymlinks(item)
		if err != nil {
			resolved = item
		}
		if !observed[resolved] {
			observed[resolved] = true
			out = append(out, item)
		}
	}
	return out
}

func isFile(pos string) bool {
	info, err := os.Stat(pos)
	return err == nil && info.Mode().IsRegular()
}

// normalize lower-cases a name and trims the space around it.
func normalize(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

var versionPart = regexp.MustCompile(`[0-9]+`)

// sortNewestFirst orders package store paths by the numbers in the package name
// that follows the store hash, highest first. The newest package links the
// newest C library, and the library loaded first fixes the C library every
// later one resolves against.
func sortNewestFirst(paths []string) {
	slices.SortStableFunc(paths, func(a, b string) int {
		return -slices.Compare(getVersion(a), getVersion(b))
	})
}

// getVersion returns the numbers in the store package name of a path.
func getVersion(path string) []int {
	rel, err := filepath.Rel("/nix/store", path)
	if err != nil {
		return nil
	}
	pkg, _, _ := strings.Cut(rel, string(filepath.Separator))
	_, name, _ := strings.Cut(pkg, "-")
	var out []int
	for _, digits := range versionPart.FindAllString(name, -1) {
		n, _ := strconv.Atoi(digits)
		out = append(out, n)
	}
	return out
}
