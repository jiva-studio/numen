package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var (
	dryRun    = flag.Bool("dry-run", false, "dry run without modifying files")
	targetDir = flag.String("dir", ".", "directory to scan and process")
)

var bannedLocalNames = map[string]string{
	"at":      "pos",
	"by":      "delta",
	"held":    "target",
	"said":    "response",
	"written": "output",
	"asked":   "requested",
}

var replacementCandidates = map[string][]string{
	"at":      {"pos", "offset", "index", "targetPath", "targetOffset", "loc"},
	"by":      {"delta", "step", "shift", "stepDelta"},
	"held":    {"target", "activeItem", "retained", "targetItem"},
	"said":    {"response", "reply", "msg", "outMsg"},
	"written": {"output", "encoded", "savedData", "outBytes"},
	"asked":   {"requested", "query", "req", "inQuery"},
}

func chooseReplacement(name string, scopeHint string, usedNames map[string]bool) string {
	if name == "at" && strings.Contains(scopeHint, "range") {
		for _, cand := range []string{"item", "entry", "elem", "val"} {
			if !usedNames[cand] {
				usedNames[cand] = true
				return cand
			}
		}
	}
	candidates, ok := replacementCandidates[name]
	if !ok {
		return name
	}
	for _, cand := range candidates {
		if !usedNames[cand] {
			usedNames[cand] = true
			return cand
		}
	}
	// Fallback with suffix
	fallback := candidates[0] + "Val"
	usedNames[fallback] = true
	return fallback
}

func processFile(path string) (int, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return 0, err
	}

	renamedCount := 0
	targetObjects := make(map[*ast.Object]string)

	// Step 1: For each function, collect used identifier names and assign collision-free replacements
	ast.Inspect(file, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return true
		}

		usedNames := make(map[string]bool)
		ast.Inspect(fn, func(inner ast.Node) bool {
			if id, ok := inner.(*ast.Ident); ok {
				usedNames[id.Name] = true
			}
			return true
		})

		// Now inspect declarations inside this function
		ast.Inspect(fn, func(inner ast.Node) bool {
			switch stmt := inner.(type) {
			case *ast.RangeStmt:
				if valIdent, ok := stmt.Value.(*ast.Ident); ok {
					if _, banned := bannedLocalNames[valIdent.Name]; banned && valIdent.Obj != nil && valIdent.Obj.Kind == ast.Var {
						if _, already := targetObjects[valIdent.Obj]; !already {
							targetObjects[valIdent.Obj] = chooseReplacement(valIdent.Name, "range", usedNames)
						}
					}
				}
				if keyIdent, ok := stmt.Key.(*ast.Ident); ok && stmt.Tok == token.DEFINE {
					if _, banned := bannedLocalNames[keyIdent.Name]; banned && keyIdent.Obj != nil && keyIdent.Obj.Kind == ast.Var {
						if _, already := targetObjects[keyIdent.Obj]; !already {
							targetObjects[keyIdent.Obj] = chooseReplacement(keyIdent.Name, "range-key", usedNames)
						}
					}
				}

			case *ast.AssignStmt:
				if stmt.Tok == token.DEFINE {
					for _, lhs := range stmt.Lhs {
						if ident, ok := lhs.(*ast.Ident); ok {
							if _, banned := bannedLocalNames[ident.Name]; banned && ident.Obj != nil && ident.Obj.Kind == ast.Var {
								if _, already := targetObjects[ident.Obj]; !already {
									targetObjects[ident.Obj] = chooseReplacement(ident.Name, "var", usedNames)
								}
							}
						}
					}
				}

			case *ast.FuncLit:
				// Closure parameters
				if stmt.Type != nil && stmt.Type.Params != nil {
					for _, field := range stmt.Type.Params.List {
						for _, ident := range field.Names {
							if _, banned := bannedLocalNames[ident.Name]; banned && ident.Obj != nil && ident.Obj.Kind == ast.Var {
								if _, already := targetObjects[ident.Obj]; !already {
									targetObjects[ident.Obj] = chooseReplacement(ident.Name, "param", usedNames)
								}
							}
						}
					}
				}
			}
			return true
		})

		// Parameters of the function itself (for test helpers or private funcs)
		if fn.Type != nil && fn.Type.Params != nil && (strings.HasSuffix(path, "_test.go") || !fn.Name.IsExported()) {
			for _, field := range fn.Type.Params.List {
				for _, ident := range field.Names {
					if _, banned := bannedLocalNames[ident.Name]; banned && ident.Obj != nil && ident.Obj.Kind == ast.Var {
						if _, already := targetObjects[ident.Obj]; !already {
							targetObjects[ident.Obj] = chooseReplacement(ident.Name, "param", usedNames)
						}
					}
				}
			}
		}

		return true
	})

	if len(targetObjects) == 0 {
		return 0, nil
	}

	// Step 2: Apply renaming to all identifiers referencing target objects
	ast.Inspect(file, func(n ast.Node) bool {
		if kv, ok := n.(*ast.KeyValueExpr); ok {
			// Skip renaming kv.Key to prevent modifying struct field keys in composite literals
			ast.Inspect(kv.Value, func(inner ast.Node) bool {
				if ident, ok := inner.(*ast.Ident); ok && ident.Obj != nil {
					if newName, found := targetObjects[ident.Obj]; found {
						ident.Name = newName
						renamedCount++
					}
				}
				return true
			})
			return false
		}

		if ident, ok := n.(*ast.Ident); ok {
			if ident.Obj != nil {
				if newName, found := targetObjects[ident.Obj]; found {
					ident.Name = newName
					renamedCount++
				}
			}
		}
		return true
	})

	if renamedCount > 0 && !*dryRun {
		var buf bytes.Buffer
		if err := format.Node(&buf, fset, file); err != nil {
			return 0, err
		}
		if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
			return 0, err
		}
	}

	return len(targetObjects), nil
}

func main() {
	flag.Parse()

	totalFiles := 0
	totalVars := 0

	err := filepath.WalkDir(*targetDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == "node_modules" || name == ".git" || name == "dist" || name == ".direnv" || name == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") {
			vars, err := processFile(path)
			if err != nil {
				fmt.Printf("Error processing %s: %v\n", path, err)
				return nil
			}
			if vars > 0 {
				totalFiles++
				totalVars += vars
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Walk error: %v\n", err)
	}

	fmt.Printf("✨ Go AST Var Purger Completed (dry-run: %v):\n", *dryRun)
	fmt.Printf("   - Modified files   : %d\n", totalFiles)
	fmt.Printf("   - Renamed variables: %d\n", totalVars)
}
