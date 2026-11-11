package pluginboundary

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"golang.org/x/mod/modfile"
	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// isPlugin and getPluginRequirements are the whole rule, so they are tested
// directly: a walk over the tree that reports no violation would also pass if the
// rule detected nothing.
func TestPluginBoundaryRule_DetectsPluginImports(t *testing.T) {
	for _, one := range []struct {
		dep      string
		isPlugin bool
	}{
		{load.PluginsPrefix + "readers.epub/backend/epub", true},
		{load.PluginsPrefix + "readers.pdf", true},
		{load.Module + "usecase/document", false},
		{load.Wire + "/gen/numen/v1", false},
		{"github.com/jiva-studio/numen/pluginsx/other", false},
		{"github.com/jiva-studio/numen/libs/plugin/sdk-go/server", false},
	} {
		if got := load.IsPlugin(one.dep); got != one.isPlugin {
			t.Errorf("isPlugin(%s) = %v, want %v", one.dep, got, one.isPlugin)
		}
	}

	require := "module example.com/x\n\nrequire (\n\t" + load.PluginsPrefix + "readers.epub v0.0.0\n)\n"
	replace := "module example.com/x\n\nreplace " + load.PluginsPrefix + "readers.epub => ../../plugins/readers-epub\n"
	folder := "module example.com/x\n\nreplace example.com/y => ../../plugins/readers-epub\n"
	for name, gomod := range map[string]string{"a require": require, "a replace": replace, "a plugin folder": folder} {
		if len(getPluginRequirements(parseGoMod(t, gomod))) == 0 {
			t.Errorf("%s of a plugin is not found", name)
		}
	}
	cross := "module " + load.PluginsPrefix + "readers.epub\n\nrequire " + load.PluginsPrefix + "readers.pdf v0.0.0\n"
	if len(getPluginRequirements(parseGoMod(t, cross))) == 0 {
		t.Error("a requirement of one plugin by another is not found")
	}
	clean := "module " + load.PluginsPrefix + "readers.epub\n\nrequire example.com/y v1.0.0\n// " + load.PluginsPrefix + "x\n"
	if got := getPluginRequirements(parseGoMod(t, clean)); len(got) != 0 {
		t.Errorf("a go.mod naming no plugin is found to name %v", got)
	}
}

// TestPluginBoundary_CoreAndDesktopExcludePluginsAndSDK checks that the core and
// the desktop are compiled from no plugin and no module of the plugin SDK.
// The resolved build dependencies are checked, so a plugin imported
// transitively is detected too, on every platform. Test files are excluded.
func TestPluginBoundary_CoreAndDesktopExcludePluginsAndSDK(t *testing.T) {
	builds := []struct {
		module    string
		dir       string
		forbidSDK bool
		platforms []struct{ goos, goarch string }
	}{
		{"the core", "..", true, load.ListShippedPlatforms()},
		{"notes", filepath.Join("..", "..", "..", "note"), true, load.ListShippedPlatforms()},
		{"protocol", filepath.Join("..", "..", "protocol"), true, load.ListShippedPlatforms()},
		{"the desktop", filepath.Join("..", "..", "..", "..", "apps", "desktop"), false, load.ListShippedPlatforms()[:3]},
	}
	for _, build := range builds {
		for _, one := range build.platforms {
			t.Run(build.module+"/"+one.goos+"/"+one.goarch, func(t *testing.T) {
				packages := listPackages(t, build.dir, one.goos, one.goarch)
				if len(packages) == 0 {
					t.Fatal("go list found no package: a rule checked against nothing")
				}
				for _, pkg := range packages {
					for _, dep := range pkg.Deps {
						if load.IsPlugin(dep) || (build.forbidSDK && load.IsSDK(dep)) {
							t.Errorf("%s reaches %s", pkg.ImportPath, dep)
						}
					}
				}
			})
		}
	}
}

// No module requires a plugin: a requirement is how a build evaluates to be
// compiled from one.
func TestNoModuleRequiresAPlugin(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..")
	found := 0
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != root && (entry.Name() == "node_modules" || strings.HasPrefix(entry.Name(), ".")) {
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
		for _, line := range getPluginRequirements(load.ReadGoMod(t, path)) {
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
