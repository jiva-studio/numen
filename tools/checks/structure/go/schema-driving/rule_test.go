package schemadriving

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

func findDrivenSchemaServers(root string) ([]string, int, error) {
	driving, err := load.ListDriving(root)
	if err != nil {
		return nil, 0, err
	}
	var wrong []string
	serving := 0
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		pkg := load.GetPackage(root, path)
		if !load.IsAdapter(pkg) && !load.IsShared(pkg) {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		if !load.NamesHandler(file) {
			return nil
		}
		serving++
		if !load.IsDriving(driving, pkg) {
			wrong = append(wrong, load.GetFamily(pkg)+" serves the schema and is named driven")
		}
		return nil
	})
	return wrong, serving, err
}
