package load

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/mod/modfile"
)

const sharedLeafReason = "a library in libs/shared is a leaf: it requires only third-party modules and other libs/shared modules"

const extractableReason = "an extractable module requires only third-party modules, the plugin SDK and libs/shared"

// GetSharedLeafRefusal explains why a module in libs/shared may not require dep, or returns "".
func GetSharedLeafRefusal(dep string) string {
	if !strings.HasPrefix(dep, NumenRoot) || IsSharedLibrary(dep) {
		return ""
	}
	return sharedLeafReason
}

// GetExtractableRefusal explains why a module of the plugin SDK or a plugin may not require dep, or returns "".
// mod is the module that requires it.
func GetExtractableRefusal(mod, dep string) string {
	if !strings.HasPrefix(dep, NumenRoot) || IsWithin(dep, mod) || IsSDK(dep) || IsSharedLibrary(dep) {
		return ""
	}
	return extractableReason
}

// GetRequirementViolations lists what a go.mod in the folder dir (relative to the repository root) requires, direct
// or indirect, and builds from a folder, that refuse rejects.
func GetRequirementViolations(dir string, file *modfile.File, refuse func(dep string) string) []string {
	var found []string
	for _, required := range file.Require {
		if why := refuse(required.Mod.Path); why != "" {
			found = append(found, "requires "+required.Mod.Path+": "+why)
		}
	}
	for _, replaced := range file.Replace {
		target := replaced.New.Path
		if replaced.New.Version == "" {
			folder := filepath.ToSlash(filepath.Join(dir, target))
			if strings.HasPrefix(folder, "../") {
				found = append(found, "builds "+replaced.Old.Path+" from "+target+", outside the repository")
				continue
			}
			target = NumenRoot + folder
		}
		if why := refuse(target); why != "" {
			found = append(found, "builds "+replaced.Old.Path+" from "+replaced.New.Path+": "+why)
		}
	}
	return found
}

// ListModulesBelow lists the folders, relative to the repository root, that hold a go.mod at any depth below
// the repository folder root. It fails when there is none.
func ListModulesBelow(t *testing.T, root string) []string {
	t.Helper()
	var dirs []string
	err := filepath.WalkDir(filepath.Join(RepoRootDir, root), func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && entry.Name() == "node_modules" {
			return filepath.SkipDir
		}
		if !entry.IsDir() && entry.Name() == "go.mod" {
			dir, err := filepath.Rel(RepoRootDir, filepath.Dir(path))
			if err != nil {
				return err
			}
			dirs = append(dirs, filepath.ToSlash(dir))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(dirs) == 0 {
		t.Fatalf("no module under %s: a rule checked against nothing", root)
	}
	return dirs
}

// GetPluginModuleDirs lists the plugin module folders, relative to the repository root.
func GetPluginModuleDirs(t *testing.T) []string {
	t.Helper()
	dirs := ListPluginModules(t)
	for i, dir := range dirs {
		rel, err := filepath.Rel(RepoRootDir, dir)
		if err != nil {
			t.Fatal(err)
		}
		dirs[i] = filepath.ToSlash(rel)
	}
	return dirs
}
