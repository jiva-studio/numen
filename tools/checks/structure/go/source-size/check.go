package sourcesize

import (
	"bytes"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// maxFileLines and maxPackageLines bound the production source of one file and of one package directory.
// A file of five hundred lines is what a reader takes in at a sitting and what holds one use case or one
// adapter with its helpers; a package of three thousand lines is a dozen such files. A larger package is
// several concerns under one name, and a larger file is a package waiting to be cut.
const (
	maxFileLines    = 500
	maxPackageLines = 3000
)

func listSourceRoots() []string {
	return []string{"libs/core/kernel", "apps/desktop"}
}

func getSourceLines(path string) (int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return bytes.Count(raw, []byte("\n")) + btoi(len(raw) > 0 && raw[len(raw)-1] != 10), nil
}

func btoi(ok bool) int {
	if ok {
		return 1
	}
	return 0
}

func isGeneratedPath(path string) (bool, error) {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.PackageClauseOnly|parser.ParseComments)
	if err != nil {
		return false, err
	}
	for _, group := range file.Comments {
		if group.End() > file.Package {
			break
		}
		for _, one := range group.List {
			if strings.HasPrefix(one.Text, "// Code generated") && strings.Contains(one.Text, "DO NOT EDIT") {
				return true, nil
			}
		}
	}
	return false, nil
}

func getOversizedSources(root string, trees []string, fileMax, packageMax int) (found []string, files, dirs int, err error) {
	for _, tree := range trees {
		perDir := map[string]int{}
		walkErr := filepath.WalkDir(filepath.Join(root, tree), func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if name := entry.Name(); name == "node_modules" || name == "testdata" || strings.HasPrefix(name, ".") && path != filepath.Join(root, tree) {
					return fs.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			isGenerated, err := isGeneratedPath(path)
			if err != nil || isGenerated {
				return err
			}
			lines, err := getSourceLines(path)
			if err != nil {
				return err
			}
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return relErr
			}
			files++
			perDir[filepath.Dir(rel)] += lines
			if lines > fileMax {
				found = append(found, filepath.ToSlash(rel)+" has "+strconv.Itoa(lines)+" lines, more than "+strconv.Itoa(fileMax))
			}
			return nil
		})
		if walkErr != nil {
			return nil, 0, 0, walkErr
		}
		for dir, lines := range perDir {
			dirs++
			if limit := packageMax; lines > limit {
				found = append(found, filepath.ToSlash(dir)+" has "+strconv.Itoa(lines)+" lines in all, more than "+strconv.Itoa(limit))
			}
		}
	}
	slices.Sort(found)
	return found, files, dirs, nil
}
