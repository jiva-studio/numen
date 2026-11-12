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

func getLibraryEnvs() []string { return []string{"NUMEN_ONNX_RUNTIME_LIB", "NUMEN_ONNX_RUNTIME"} }

func getLibraryName(goos string) string {
	switch goos {
	case "darwin":
		return "libonnxruntime.dylib"
	case "windows":
		return "onnxruntime.dll"
	}
	return "libonnxruntime.so"
}

func getEnvLibrary() string {
	for _, name := range getLibraryEnvs() {
		if pos := os.Getenv(name); pos != "" {
			return pos
		}
	}
	return ""
}

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

func findMatchingPaths(pattern string) []string {
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

func getAdjacentDirs(pluginDir string) []string {
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

func getCandidates(pluginDir, home, librarySum string, machine []string) []string {
	return getCandidatesInStore(pluginDir, home, librarySum, machine, filepath.Join("/nix/store", "*-onnxruntime-*", "lib"))
}

func getCandidatesInStore(pluginDir, home, librarySum string, machine []string, storePattern string) []string {
	name := getLibraryName(runtime.GOOS)
	var out []string
	for _, dir := range getAdjacentDirs(pluginDir) {
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
		out = append(out, findMatchingPaths(filepath.Join(storePattern, name+"*"))...)
		out = append(out, findInDirs(getLoaderDirs(runtime.GOOS), name)...)
	}
	return removeDuplicates(out)
}

func findInDirs(dirs []string, name string) []string {
	var out []string
	for _, dir := range dirs {
		if filepath.IsAbs(dir) {
			out = append(out, findMatchingPaths(filepath.Join(dir, name+"*"))...)
		}
	}
	return out
}

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
	for line := range strings.SplitSeq(string(raw), "\n") {
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

func isConfigSeparator(r rune) bool {
	return r == ' ' || r == '\t' || r == '\r' || r == ',' || r == ':'
}

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

func normalize(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

var versionPart = regexp.MustCompile(`[0-9]+`)

func sortNewestFirst(paths []string) {
	slices.SortStableFunc(paths, func(a, b string) int {
		return -slices.Compare(getVersion(a), getVersion(b))
	})
}

func getVersion(path string) []int {
	rel, err := filepath.Rel("/nix/store", path)
	if err != nil {
		return nil
	}
	pkg, _, _ := strings.Cut(rel, string(filepath.Separator))
	_, name, _ := strings.Cut(pkg, "-")
	var out []int
	for _, digits := range versionPart.FindAllString(name, -1) {
		n, err := strconv.Atoi(digits)
		if err != nil {
			return nil
		}
		out = append(out, n)
	}
	return out
}
