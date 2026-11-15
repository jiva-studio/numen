package serialization

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
)

func getSerializationImports() map[string]bool {
	return map[string]bool{
		"encoding/json":              true,
		"encoding/xml":               true,
		"gopkg.in/yaml.v3":           true,
		"github.com/BurntSushi/toml": true,
		"google.golang.org/protobuf": true,
	}
}

func listSerializationTags() []string {
	return []string{"json:", "yaml:", "sql:", "db:", "xml:", "protobuf:"}
}

// layerRoot is a folder of an inner layer; the domain of the core may import a format package and carries no
// format tag.
type layerRoot struct {
	dir             string
	isImportAllowed bool
}

func getImportFindings(path string, file *ast.File) []string {
	var wrong []string
	for _, imp := range file.Imports {
		to, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}
		if getSerializationImports()[to] || strings.HasPrefix(to, "google.golang.org/protobuf/") {
			wrong = append(wrong, path+" imports "+to+": inner layers do not parse wire or storage formats")
		}
	}
	return wrong
}

func getTagFindings(path string, file *ast.File) []string {
	var wrong []string
	ast.Inspect(file, func(n ast.Node) bool {
		spec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}
		st, ok := spec.Type.(*ast.StructType)
		if !ok || st.Fields == nil {
			return true
		}
		for _, field := range st.Fields.List {
			if field.Tag == nil {
				continue
			}
			for _, tag := range listSerializationTags() {
				if strings.Contains(field.Tag.Value, tag) {
					fieldName := "<embedded>"
					if len(field.Names) > 0 {
						fieldName = field.Names[0].Name
					}
					wrong = append(wrong, path+" "+spec.Name.Name+"."+fieldName+" has serialization tag "+field.Tag.Value+": format tags belong on adapter DTOs")
				}
			}
		}
		return true
	})
	return wrong
}

func findSerialization(roots []layerRoot) ([]string, int, error) {
	var wrong []string
	read := 0
	for _, root := range roots {
		err := filepath.WalkDir(root.dir, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				return err
			}
			read++
			if !root.isImportAllowed {
				wrong = append(wrong, getImportFindings(path, file)...)
			}
			wrong = append(wrong, getTagFindings(path, file)...)
			return nil
		})
		if err != nil {
			return nil, read, err
		}
	}
	return wrong, read, nil
}
