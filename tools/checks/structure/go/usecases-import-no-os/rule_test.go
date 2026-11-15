package usecasesimportnoos

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
)

func findOSPathImports(root string) ([]string, int, error) {
	var wrong []string
	read := 0
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		read++
		for _, imp := range file.Imports {
			to, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				continue
			}
			if to == "path/filepath" || to == "io/fs" {
				wrong = append(wrong, path+" imports "+to+": use cases operate on slash-separated vault paths and domain sentinels, not OS filepath or io/fs")
			}
		}
		return nil
	})
	return wrong, read, err
}
