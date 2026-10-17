package onnxruntime

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// libraryEnvs name the library a person points the host at. The first set one
// is used as given.
var libraryEnvs = []string{"NUMEN_ONNX_RUNTIME_LIB", "NUMEN_ONNX_RUNTIME", "NUMEN_OCR_RUNTIME"}

// nixStorePattern names where Nix keeps packages.
var nixStorePattern = filepath.Join("/nix/store", "*-onnxruntime-*", "lib")

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
		if at := os.Getenv(name); at != "" {
			return at
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
	for _, at := range found {
		if info, err := os.Stat(at); err == nil && info.Mode().IsRegular() {
			out = append(out, at)
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

// findOnMachine returns the first library this machine holds without fetching
// one: beside the plugin or the host, where an earlier run unpacked one (when it
// holds the pinned sum), in the folders shared libraries live in, and in the
// package store. machine replaces the folders shared libraries live in when it
// is not nil.
func findOnMachine(pluginDir, home, librarySum string, machine []string) string {
	name := getLibraryName(runtime.GOOS)
	for _, dir := range getBesideDirs(pluginDir) {
		if at := filepath.Join(dir, name); isFile(at) {
			return at
		}
	}
	if at := filepath.Join(home, name); home != "" && isFile(at) && verifyFile(at, librarySum) == nil {
		return at
	}
	isDefault := machine == nil
	if machine == nil {
		machine = getMachineDirs()
	}
	for _, dir := range machine {
		if found := findMatching(filepath.Join(dir, name+"*")); len(found) > 0 {
			return found[0]
		}
	}
	if !isDefault {
		return ""
	}
	if found := findMatching(filepath.Join(nixStorePattern, name+"*")); len(found) > 0 {
		return found[0]
	}
	return ""
}

// isFile reports whether a regular file stands at the path.
func isFile(at string) bool {
	info, err := os.Stat(at)
	return err == nil && info.Mode().IsRegular()
}

// normalize lower-cases a name and trims the space around it.
func normalize(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
