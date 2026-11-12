// Command backend_commands reads Go source files with go/ast and prints the command ids registered
// on a command.Registry as JSON. It reads file paths from standard input, one per line: the first
// line is libs/plugin/sdk-go/command/shared_commands.go, and the remaining lines are the plugin
// backend Go files.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
)

type registration struct {
	File string `json:"file"`
	ID   string `json:"id"`
}

func main() {
	var paths []string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		if path := strings.TrimSpace(scanner.Text()); path != "" {
			paths = append(paths, path)
		}
	}
	if len(paths) < 1 {
		fmt.Fprintln(os.Stderr, "usage: backend_commands < paths")
		os.Exit(2)
	}
	sharedConsts, err := readSharedConstants(paths[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", paths[0], err)
		os.Exit(2)
	}
	found := []registration{}
	for _, path := range paths[1:] {
		entries, err := readFileRegistrations(path, sharedConsts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
			os.Exit(2)
		}
		found = append(found, entries...)
	}
	encoded, err := json.Marshal(found)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	os.Stdout.Write(encoded)
}

func readSharedConstants(path string) (map[string]string, error) {
	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		return nil, err
	}
	consts := map[string]string{}
	for _, decl := range parsed.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			valSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range valSpec.Names {
				if i < len(valSpec.Values) {
					if text, ok := stringLiteral(valSpec.Values[i]); ok {
						consts[name.Name] = text
					}
				}
			}
		}
	}
	return consts, nil
}

func readFileRegistrations(path string, sharedConsts map[string]string) ([]registration, error) {
	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		return nil, err
	}
	commandPkgs := findCommandPackages(parsed)
	localConsts := map[string]string{}
	for _, decl := range parsed.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			valSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range valSpec.Names {
				if i < len(valSpec.Values) {
					if text, ok := resolveValue(valSpec.Values[i], localConsts, sharedConsts, commandPkgs); ok {
						localConsts[name.Name] = text
					}
				}
			}
		}
	}
	var found []registration
	for _, decl := range parsed.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		registries := findCommandRegistries(fn.Body, commandPkgs)
		if len(registries) == 0 {
			continue
		}
		rangeKeys := findRangeKeys(fn.Body, localConsts, sharedConsts, commandPkgs)
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Add" {
				return true
			}
			recv, ok := sel.X.(*ast.Ident)
			if !ok || !registries[recv.Name] {
				return true
			}
			for _, id := range resolveCommandIDs(call.Args[0], localConsts, sharedConsts, rangeKeys, commandPkgs) {
				found = append(found, registration{File: path, ID: id})
			}
			return true
		})
	}
	return found, nil
}

func findCommandPackages(parsed *ast.File) map[string]bool {
	pkgs := map[string]bool{}
	for _, imp := range parsed.Imports {
		importPath, ok := stringLiteral(imp.Path)
		if !ok || !strings.HasSuffix(importPath, "/command") {
			continue
		}
		if imp.Name != nil && imp.Name.Name != "_" && imp.Name.Name != "." {
			pkgs[imp.Name.Name] = true
		} else {
			pkgs["command"] = true
		}
	}
	return pkgs
}

func findCommandRegistries(body *ast.BlockStmt, commandPkgs map[string]bool) map[string]bool {
	registries := map[string]bool{}
	ast.Inspect(body, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for i, rhs := range assign.Rhs {
			call, ok := rhs.(*ast.CallExpr)
			if !ok {
				continue
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "New" {
				continue
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok || !commandPkgs[pkg.Name] {
				continue
			}
			if i < len(assign.Lhs) {
				if ident, ok := assign.Lhs[i].(*ast.Ident); ok {
					registries[ident.Name] = true
				}
			}
		}
		return true
	})
	return registries
}

func findRangeKeys(body *ast.BlockStmt, localConsts, sharedConsts map[string]string, commandPkgs map[string]bool) map[string][]string {
	keys := map[string][]string{}
	ast.Inspect(body, func(n ast.Node) bool {
		rangeStmt, ok := n.(*ast.RangeStmt)
		if !ok {
			return true
		}
		keyIdent, ok := rangeStmt.Key.(*ast.Ident)
		if !ok {
			return true
		}
		comp, ok := rangeStmt.X.(*ast.CompositeLit)
		if !ok {
			return true
		}
		var list []string
		for _, elt := range comp.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			if text, ok := resolveValue(kv.Key, localConsts, sharedConsts, commandPkgs); ok {
				list = append(list, text)
			} else {
				list = append(list, "<unreadable>")
			}
		}
		keys[keyIdent.Name] = append(keys[keyIdent.Name], list...)
		return true
	})
	return keys
}

func resolveCommandIDs(expr ast.Expr, localConsts, sharedConsts map[string]string, rangeKeys map[string][]string, commandPkgs map[string]bool) []string {
	if text, ok := resolveValue(expr, localConsts, sharedConsts, commandPkgs); ok {
		return []string{text}
	}
	if ident, ok := expr.(*ast.Ident); ok {
		if list, hasRange := rangeKeys[ident.Name]; hasRange {
			return list
		}
	}
	return []string{""}
}

func resolveValue(expr ast.Expr, localConsts, sharedConsts map[string]string, commandPkgs map[string]bool) (string, bool) {
	if text, ok := stringLiteral(expr); ok {
		return text, true
	}
	switch x := expr.(type) {
	case *ast.Ident:
		if text, ok := localConsts[x.Name]; ok {
			return text, true
		}
	case *ast.SelectorExpr:
		if pkg, ok := x.X.(*ast.Ident); ok && commandPkgs[pkg.Name] {
			if text, ok := sharedConsts[x.Sel.Name]; ok {
				return text, true
			}
		}
	}
	return "", false
}

func stringLiteral(expr ast.Expr) (string, bool) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	text, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return text, true
}
