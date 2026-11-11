package load

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/packages"
)

func LoadPackages(t *testing.T, dir string, env []string, patterns ...string) []*packages.Package {
	t.Helper()
	loaded, err := packages.Load(NewTypedConfig(dir, env), patterns...)
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func NewTypedConfig(dir string, env []string) *packages.Config {
	return &packages.Config{
		Dir:   dir,
		Env:   append(os.Environ(), env...),
		Tests: true,
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports |
			packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
	}
}

func ListWorkspaceModules(t *testing.T) []string {
	t.Helper()
	command := exec.CommandContext(t.Context(), "go", "work", "edit", "-json")
	command.Dir = RepoRootDir
	output, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	var work struct {
		Use []struct {
			DiskPath string `json:"DiskPath"`
		} `json:"Use"`
	}
	if err := json.Unmarshal(output, &work); err != nil {
		t.Fatal(err)
	}
	dirs := make([]string, len(work.Use))
	for i, one := range work.Use {
		abs, err := filepath.Abs(filepath.Join(RepoRootDir, one.DiskPath))
		if err != nil {
			t.Fatal(err)
		}
		dirs[i] = abs
	}
	return dirs
}
