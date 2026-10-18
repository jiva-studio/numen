package container

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// pluginsPrefix is where the plugins are compiled from. A package under it is a
// plugin, and a module under it is one too.
const pluginsPrefix = "github.com/jiva-studio/numen/modules/plugins/"

// compositionRoots are the modules that name plugins: the monolithic build of the
// desktop binds them in. Every other module meets a plugin only through the
// plugin contract.
var compositionRoots = []string{"apps/desktop-monolith"}

// isPlugin says whether a package is compiled from a plugin.
func isPlugin(importPath string) bool {
	return strings.HasPrefix(importPath, pluginsPrefix)
}

// mayNamePlugins says whether the module at a directory, slashed and relative to
// the root of the modules, may require plugins: a composition root may, and a
// plugin names no other, so its own module directive is the one line it carries.
func mayNamePlugins(dir string) bool {
	for _, root := range compositionRoots {
		if dir == root {
			return true
		}
	}
	return false
}

// getPluginRequirements is every line of a go.mod that requires or replaces a
// plugin, outside the module directive.
func getPluginRequirements(gomod string) []string {
	var named []string
	for line := range strings.SplitSeq(gomod, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "module ") || strings.HasPrefix(trimmed, "//") {
			continue
		}
		if strings.Contains(trimmed, pluginsPrefix) || strings.Contains(trimmed, "/plugins/") {
			named = append(named, trimmed)
		}
	}
	return named
}

// What isPlugin, mayNamePlugins and getPluginRequirements say is the whole of the
// rule, so it is asked directly. A walk that reads the tree and refuses nothing
// passes: these are the edges that have to come back refused.
func TestWhatThePluginBoundaryRefuses(t *testing.T) {
	for _, one := range []struct {
		dep      string
		isPlugin bool
	}{
		{pluginsPrefix + "reader-epub/backend/epub", true},
		{pluginsPrefix + "reader-pdf", true},
		{module + "usecase/document", false},
		{wire + "/gen/numen/v1", false},
		{"github.com/jiva-studio/numen/modules/pluginsx/other", false},
		{"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server", false},
	} {
		if got := isPlugin(one.dep); got != one.isPlugin {
			t.Errorf("isPlugin(%s) = %v, want %v", one.dep, got, one.isPlugin)
		}
	}

	for _, one := range []struct {
		dir string
		may bool
	}{
		{"libs/core", false},
		{"libs/protocol", false},
		{"apps/desktop", false},
		{"apps/mobile", false},
		{"apps/desktop-monolith", true},
		{"plugins/reader-epub", false},
		{"pluginsx", false},
	} {
		if got := mayNamePlugins(one.dir); got != one.may {
			t.Errorf("mayNamePlugins(%s) = %v, want %v", one.dir, got, one.may)
		}
	}

	require := "module example.com/x\n\nrequire (\n\t" + pluginsPrefix + "reader-epub v0.0.0\n)\n"
	replace := "module example.com/x\n\nreplace " + pluginsPrefix + "reader-epub => ../../plugins/reader-epub\n"
	for name, gomod := range map[string]string{"a require": require, "a replace": replace} {
		if len(getPluginRequirements(gomod)) == 0 {
			t.Errorf("%s of a plugin is not found", name)
		}
	}
	cross := "module " + pluginsPrefix + "reader-epub\n\nrequire " + pluginsPrefix + "reader-pdf v0.0.0\n"
	if len(getPluginRequirements(cross)) == 0 {
		t.Error("a requirement of one plugin by another is not found")
	}
	clean := "module " + pluginsPrefix + "reader-epub\n\nrequire example.com/y v1.0.0\n// " + pluginsPrefix + "x\n"
	if got := getPluginRequirements(clean); len(got) != 0 {
		t.Errorf("a go.mod naming no plugin is found to name %v", got)
	}
}

// listPackages is what one module's build resolves on one platform, and what
// each package of it is compiled from.
func listPackages(t *testing.T, dir, goos, goarch string, tags ...string) []listed {
	t.Helper()
	args := []string{"list", "-json"}
	if len(tags) > 0 {
		args = append(args, "-tags", strings.Join(tags, ","))
	}
	args = append(args, "./...")
	listing := exec.CommandContext(t.Context(), "go", args...)
	listing.Dir = dir
	listing.Env = append(os.Environ(), "GOOS="+goos, "GOARCH="+goarch)
	out, err := listing.Output()
	if err != nil {
		var ran *exec.ExitError
		if errors.As(err, &ran) {
			t.Fatalf("go list: %v: %s", err, ran.Stderr)
		}
		t.Fatalf("go list: %v", err)
	}
	var packages []listed
	decoder := json.NewDecoder(bytes.NewReader(out))
	for {
		var pkg listed
		err := decoder.Decode(&pkg)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("go list: %v", err)
		}
		packages = append(packages, pkg)
	}
	return packages
}

// listed is a package as go list says it, with everything it is compiled from at
// whatever remove.
type listed struct {
	ImportPath string
	Imports    []string
	Deps       []string
}

// The core and the desktop meet a plugin only through the protocol: they are
// compiled from no plugin and no module of the plugin SDK. What the build
// resolved is what is read, so a package reached through another answers for
// it too, on every platform and in the monolithic build.
//
// A test file is left out: a test stands outside the package it exercises.
func TestNothingTheCoreOrTheDesktopIsCompiledFromIsAPluginOrTheSDK(t *testing.T) {
	builds := []struct {
		module    string
		dir       string
		platforms []struct{ goos, goarch string }
	}{
		{"the core", "..", shipped},
		{"the desktop", filepath.Join("..", "..", "..", "apps", "desktop"), shipped[:3]},
	}
	for _, build := range builds {
		for _, one := range build.platforms {
			for _, tags := range [][]string{nil, {"plugins_monolith"}} {
				name := build.module + "/" + one.goos + "/" + one.goarch
				if tags != nil {
					name += "/monolith"
				}
				t.Run(name, func(t *testing.T) {
					packages := listPackages(t, build.dir, one.goos, one.goarch, tags...)
					if len(packages) == 0 {
						t.Fatal("go list found no package: a rule checked against nothing")
					}
					for _, pkg := range packages {
						for _, dep := range pkg.Deps {
							if isPlugin(dep) || isSDK(dep) {
								t.Errorf("%s reaches %s", pkg.ImportPath, dep)
							}
						}
					}
				})
			}
		}
	}
}

// No module requires a plugin but a composition root: a
// requirement is how a build comes to be compiled from one.
func TestNoModuleButACompositionRootRequiresAPlugin(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	found := 0
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == "node_modules" || entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Name() != "go.mod" {
			return nil
		}
		found++
		dir, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return err
		}
		dir = filepath.ToSlash(dir)
		if mayNamePlugins(dir) {
			return nil
		}
		gomod, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, line := range getPluginRequirements(string(gomod)) {
			t.Errorf("%s/go.mod requires a plugin: %s", dir, line)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if found < 2 {
		t.Fatalf("%d go.mod files found under %s: a rule checked against nothing", found, root)
	}
}
