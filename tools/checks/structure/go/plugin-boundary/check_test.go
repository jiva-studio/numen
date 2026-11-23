// Package pluginboundary is the check `go/plugin-boundary`, run on the tree with the planted violations that prove it.
package pluginboundary

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// isPlugin and getPluginRequirements are the rule on a go.mod, so they are tested
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
		platforms []load.Platform
	}{
		{"the core", "..", true, load.ListShippedPlatforms()},
		{"notes", filepath.Join("..", "..", "..", "shared", "note"), true, load.ListShippedPlatforms()},
		{"protocol", filepath.Join("..", "..", "..", "shared", "protocol"), true, load.ListShippedPlatforms()},
		{"the desktop", filepath.Join("..", "..", "..", "..", "apps", "desktop"), false, load.ListShippedPlatforms()[:3]},
	}
	for _, build := range builds {
		for _, one := range build.platforms {
			t.Run(build.module+"/"+one.GOOS+"/"+one.GOARCH, func(t *testing.T) {
				packages := listPackages(t, build.dir, one.GOOS, one.GOARCH)
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

// No module requires a plugin its own code does not register as linked into its binary: a requirement is how a
// build evaluates to be compiled from one.
func TestNoModuleRequiresAPlugin(t *testing.T) {
	for _, one := range findPluginRequirementsInModules(t, filepath.Join("..", "..", "..", "..")) {
		t.Error(one)
	}
}

// TestModuleWalk_FindsThePlantedRequirement verifies that the walk finds every requirement of a plugin the code of
// its module does not register as linked in, skips node_modules and hidden folders, and fails on a tree without a go.mod.
func TestModuleWalk_FindsThePlantedRequirement(t *testing.T) {
	alpha := "require " + load.PluginsPrefix + "alpha v0.0.0\n"
	beta := "require " + load.PluginsPrefix + "beta v0.0.0\n"
	imports := "import (\n\talpha \"" + load.PluginsPrefix + "alpha\"\n\talphaservice \"" + load.PluginsPrefix + "alpha/backend/service\"\n)\n\n"
	registers := "package main\n\n" + imports +
		"func register(roles roles) error {\n\treturn roles.Discovery.RegisterBuiltIn(alpha.Manifest, serve(alphaservice.Mount))\n}\n"
	uses := "package main\n\n" + imports +
		"func register(roles roles) error {\n\treturn roles.Discovery.Register(alpha.Manifest, serve(alphaservice.Mount))\n}\n"
	mountOnly := "package main\n\n" + imports +
		"func register(roles roles) error {\n\treturn roles.Discovery.RegisterBuiltIn(other.Manifest, serve(alphaservice.Mount, alpha.Name))\n}\n"
	root := load.WriteTreeFixture(t, map[string]string{
		"libs/core/go.mod":                     "module example.test/core\n\ngo 1.26\n\n" + alpha,
		"libs/clean/go.mod":                    "module example.test/clean\n\ngo 1.26\n",
		"apps/desktop/go.mod":                  "module example.test/desktop\n\ngo 1.26\n\n" + alpha,
		"apps/desktop/cmd/numen/main_test.go":  registers,
		"apps/imported/go.mod":                 "module example.test/imported\n\ngo 1.26\n\n" + alpha,
		"apps/imported/cmd/numen/core.go":      uses,
		"apps/mounted/go.mod":                  "module example.test/mounted\n\ngo 1.26\n\n" + alpha,
		"apps/mounted/cmd/numen/core.go":       mountOnly,
		"apps/mobile/go.mod":                   "module example.test/mobile\n\ngo 1.26\n\n" + alpha + beta,
		"apps/mobile/cmd/numen-linked/core.go": registers,
		"apps/mobile/nested/go.mod":            "module example.test/nested\n\ngo 1.26\n\n" + beta,
		"apps/mobile/nested/core.go":           strings.ReplaceAll(registers, "alpha", "beta"),
		"node_modules/x/go.mod":                "module example.test/x\n\ngo 1.26\n\n" + alpha,
		".hidden/go.mod":                       "module example.test/hidden\n\ngo 1.26\n\n" + alpha,
	})
	found := findPluginRequirementsInModules(t, root)
	want := []string{
		"apps/desktop/go.mod requires a plugin: requires " + load.PluginsPrefix + "alpha",
		"apps/imported/go.mod requires a plugin: requires " + load.PluginsPrefix + "alpha",
		"apps/mobile/go.mod requires a plugin: requires " + load.PluginsPrefix + "beta",
		"apps/mounted/go.mod requires a plugin: requires " + load.PluginsPrefix + "alpha",
		"libs/core/go.mod requires a plugin: requires " + load.PluginsPrefix + "alpha",
	}
	if !slices.Equal(found, want) {
		t.Errorf("the walk found %v, want %v", found, want)
	}
}

func findPluginRequirementsInModules(t *testing.T, root string) []string {
	// Fails the test when it reads no go.mod file.
	t.Helper()
	var wrong []string
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
		requirements := getPluginRequirements(load.ReadGoMod(t, path))
		if len(requirements) == 0 {
			return nil
		}
		registered := listRegisteredPlugins(t, filepath.Dir(path))
		for _, one := range requirements {
			if !isRegistered(one.module, registered) {
				wrong = append(wrong, dir+"/go.mod requires a plugin: "+one.line)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if found == 0 {
		t.Fatalf("no go.mod found under %s: a rule checked against nothing", root)
	}
	return wrong
}

func TestMain(m *testing.M) {
	os.Exit(load.RunTests(m))
}
