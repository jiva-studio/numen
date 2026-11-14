package indexbookkeeping

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
)

func listRewritingStatements() []string {
	return []string{
		"schema_migrations_next",
		"pragma_table_info('schema_migrations')",
		"ALTER TABLE schema_migrations",
	}
}

func getRemovedFunctions() map[string]bool {
	return map[string]bool{"resetIndex": true, "dropEverything": true, "getDropping": true}
}

// removedIdentifier is a removed function of the index, rejected wherever a package of the index names it.
const removedIdentifier = "findDroppableSQL"

func getFileFindings(rel string, file *ast.File, isTopLevel bool) []string {
	var wrong []string
	ast.Inspect(file, func(node ast.Node) bool {
		switch current := node.(type) {
		case *ast.BasicLit:
			if current.Kind != token.STRING {
				return true
			}
			text, err := strconv.Unquote(current.Value)
			if err != nil {
				return true
			}
			for _, statement := range listRewritingStatements() {
				if strings.Contains(text, statement) {
					wrong = append(wrong, rel+" holds "+strconv.Quote(statement))
				}
			}
		case *ast.Ident:
			if current.Name == removedIdentifier {
				wrong = append(wrong, rel+" names "+removedIdentifier)
			}
		case *ast.FuncDecl:
			if isTopLevel && current.Recv == nil && getRemovedFunctions()[current.Name.Name] {
				wrong = append(wrong, rel+" declares "+current.Name.Name)
			}
		}
		return true
	})
	return wrong
}

func findBookkeepingRewrites(root string) ([]string, int, error) {
	var wrong []string
	read := 0
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		read++
		wrong = append(wrong, getFileFindings(filepath.ToSlash(rel), file, filepath.Dir(rel) == ".")...)
		return nil
	})
	return wrong, read, err
}
