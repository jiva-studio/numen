package spelling

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

func getBritishStems() []string {
	var stems []string
	for _, american := range []string{
		"normaliz", "quantiz", "recogniz", "initializ", "serializ", "synchroniz",
		"minimiz", "maximiz", "customiz", "authoriz", "sanitiz", "summariz", "prioritiz", "localiz", "utiliz",
	} {
		stems = append(stems, strings.ReplaceAll(american, "z", "s"))
	}
	for _, american := range []string{"neighbor", "color", "behavior"} {
		stems = append(stems, strings.ReplaceAll(american, "or", "our"))
	}
	return append(stems, "cent"+"re")
}

func getBritishStem(name string) string {
	lower := strings.ToLower(name)
	for _, stem := range getBritishStems() {
		if strings.Contains(lower, stem) {
			return stem
		}
	}
	return ""
}

func isGeneratedFile(file *ast.File) bool {
	for _, group := range file.Comments {
		for _, one := range group.List {
			if strings.HasPrefix(one.Text, "// Code generated") && strings.HasSuffix(one.Text, "DO NOT EDIT.") {
				return true
			}
		}
	}
	return false
}

func listBritishIdentifiers(path string, src any) ([]load.Finding, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil || isGeneratedFile(file) {
		return nil, err
	}
	var found []load.Finding
	ast.Inspect(file, func(node ast.Node) bool {
		if one, is := node.(*ast.Ident); is && getBritishStem(one.Name) != "" {
			found = append(found, load.NewFinding(fset, one.Pos(), one.Name))
		}
		return true
	})
	return found, nil
}
