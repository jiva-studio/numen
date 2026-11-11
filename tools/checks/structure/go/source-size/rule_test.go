package sourcesize

import (
	"go/scanner"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/mod/modfile"
)

// maxFileLines and maxPackageLines bound the lines of code of one file and of one package directory, and
// maxLineWidth bounds one line of a hand-written file. A file of five hundred lines is what a reader takes
// in at a sitting and what holds one use case or one adapter with its helpers; a package of three thousand
// lines is a dozen such files. A larger package is several concerns under one name, and a larger file is a
// package waiting to be cut. A line wider than two hundred characters is a table or a literal that a
// reader scrolls sideways through, and it hides lines from the count.
const (
	maxFileLines    = 500
	maxPackageLines = 3000
	maxLineWidth    = 200
)

func listModuleTrees(root string) ([]string, error) {
	raw, err := os.ReadFile(filepath.Join(root, "go.work"))
	if err != nil {
		return nil, err
	}
	work, err := modfile.ParseWork("go.work", raw, nil)
	if err != nil {
		return nil, err
	}
	trees := make([]string, 0, len(work.Use))
	for _, one := range work.Use {
		trees = append(trees, filepath.ToSlash(filepath.Clean(one.Path)))
	}
	slices.Sort(trees)
	return trees, nil
}

func countCodeLines(src []byte) int {
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(src))
	var scan scanner.Scanner
	scan.Init(file, src, nil, 0)
	lines := map[int]bool{}
	for {
		pos, tok, literal := scan.Scan()
		if tok == token.EOF {
			break
		}
		if tok == token.SEMICOLON && literal == "\n" {
			continue
		}
		first := file.Line(pos)
		last := first
		if tok == token.STRING {
			last += strings.Count(literal, "\n")
		}
		for line := first; line <= last; line++ {
			lines[line] = true
		}
	}
	return len(lines)
}

func listWideLines(src []byte, limit int) []int {
	var wide []int
	for i, line := range strings.Split(string(src), "\n") {
		if utf8.RuneCountInString(strings.TrimSuffix(line, "\r")) > limit {
			wide = append(wide, i+1)
		}
	}
	return wide
}

// sizeLimits are the bounds getOversizedSources enforces.
type sizeLimits struct {
	file, pkg, width int
}

func isSkippedDir(name string) bool {
	return name == "node_modules" || name == "testdata" || strings.HasPrefix(name, ".")
}

func getOversizedSources(root string, trees []string, limit sizeLimits) (found []string, files, dirs int, err error) {
	fsys := os.DirFS(root)
	for _, tree := range trees {
		perDir := map[string]int{}
		walkErr := fs.WalkDir(fsys, tree, func(rel string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if rel == tree {
					return nil
				}
				if _, statErr := fs.Stat(fsys, path.Join(rel, "go.mod")); isSkippedDir(entry.Name()) || statErr == nil {
					return fs.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(rel, ".go") {
				return nil
			}
			src, readErr := fs.ReadFile(fsys, rel)
			if readErr != nil {
				return readErr
			}
			header, isClaimed := getGeneratedHeader(src)
			if isClaimed && isProduced(fsys, rel, header) {
				return nil
			}
			if isClaimed {
				found = append(found, rel+":1 claims `"+header+"`, and no generator produces the file")
			}
			for _, line := range listWideLines(src, limit.width) {
				found = append(found, rel+":"+strconv.Itoa(line)+" is wider than "+strconv.Itoa(limit.width)+" characters")
			}
			if strings.HasSuffix(rel, "_test.go") {
				return nil
			}
			lines := countCodeLines(src)
			files++
			perDir[path.Dir(rel)] += lines
			if lines > limit.file {
				found = append(found, rel+" has "+strconv.Itoa(lines)+" lines of code, more than "+strconv.Itoa(limit.file))
			}
			return nil
		})
		if walkErr != nil {
			return nil, 0, 0, walkErr
		}
		for dir, lines := range perDir {
			dirs++
			if lines > limit.pkg {
				found = append(found, dir+" has "+strconv.Itoa(lines)+" lines of code in all, more than "+strconv.Itoa(limit.pkg))
			}
		}
	}
	slices.Sort(found)
	return found, files, dirs, nil
}
