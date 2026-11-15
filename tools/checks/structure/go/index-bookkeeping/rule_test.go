package indexbookkeeping

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"strconv"
	"strings"
)

// migrationTable is the table in which the index records the migrations it ran.
const migrationTable = "SCHEMA_MIGRATIONS"

func isMigrationRecord(statement string) bool {
	return strings.HasPrefix(statement, "CREATE TABLE IF NOT EXISTS "+migrationTable+" ") ||
		strings.HasPrefix(statement, "INSERT INTO "+migrationTable+" ") ||
		strings.HasPrefix(statement, "SELECT ")
}

func isIdentifierByte(b byte) bool {
	return b == '_' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9'
}

func namesAnotherShapeOfTheMigrationTable(statement string) bool {
	for at := strings.Index(statement, migrationTable); at >= 0; {
		end := at + len(migrationTable)
		if end < len(statement) && isIdentifierByte(statement[end]) || strings.HasSuffix(statement[:at], "PRAGMA_TABLE_INFO('") {
			return true
		}
		next := strings.Index(statement[end:], migrationTable)
		if next < 0 {
			break
		}
		at = end + next
	}
	return false
}

func isDropOfATableNamedAtRunTime(statement string) bool {
	_, dropped, isDrop := strings.Cut(statement, "DROP TABLE")
	if !isDrop {
		return false
	}
	dropped = strings.TrimPrefix(strings.TrimSpace(dropped), "IF EXISTS")
	dropped = strings.TrimSpace(dropped)
	return dropped == "" || !isIdentifierByte(dropped[0])
}

func getStatementFindings(rel, sql string) []string {
	var wrong []string
	for one := range strings.SplitSeq(sql, ";") {
		statement := strings.ToUpper(strings.Join(strings.Fields(one), " "))
		if isDropOfATableNamedAtRunTime(statement) {
			wrong = append(wrong, rel+" drops a table it names at run time")
		}
		if !strings.Contains(statement, migrationTable) {
			continue
		}
		if namesAnotherShapeOfTheMigrationTable(statement) || !isMigrationRecord(statement) {
			wrong = append(wrong, rel+" changes the migration table")
		}
	}
	return wrong
}

func getFileFindings(rel string, file *ast.File) []string {
	var wrong []string
	ast.Inspect(file, func(node ast.Node) bool {
		literal, ok := node.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		text, err := strconv.Unquote(literal.Value)
		if err == nil {
			wrong = append(wrong, getStatementFindings(rel, text)...)
		}
		return true
	})
	return wrong
}

func getSourceFindings(tree fs.FS, rel string) ([]string, error) {
	text, err := fs.ReadFile(tree, rel)
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(rel, ".sql") {
		return getStatementFindings(rel, string(text)), nil
	}
	file, err := parser.ParseFile(token.NewFileSet(), rel, text, 0)
	if err != nil {
		return nil, err
	}
	return getFileFindings(rel, file), nil
}

func isIndexSource(path string) bool {
	return strings.HasSuffix(path, ".sql") || strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go")
}

func findBookkeepingRewrites(root string) (wrong []string, read int, err error) {
	dir, err := os.OpenRoot(root)
	if err != nil {
		return nil, 0, err
	}
	defer func() {
		if closeErr := dir.Close(); err == nil {
			err = closeErr
		}
	}()
	tree := dir.FS()
	err = fs.WalkDir(tree, ".", func(rel string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !isIndexSource(rel) {
			return err
		}
		found, err := getSourceFindings(tree, rel)
		if err != nil {
			return err
		}
		read++
		wrong = append(wrong, found...)
		return nil
	})
	return wrong, read, err
}
