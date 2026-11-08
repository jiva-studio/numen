// Command lint-comments reports comments on Go struct fields and doc comments on unexported
// functions. It reads the paths of the files to check from standard input, one per line; with
// -fix it removes those comments and rewrites the files.
//
// Exported fields of libs/plugin-sdk-go keep their comments: plugin authors read them in the
// package documentation. Directive lines (//go:, //export, gomutants, Deprecated:)
// are never reported.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"sort"
	"strings"
)

var directive = regexp.MustCompile(`^//(go:|lint:|export |line |extern |\s*gomutants:|\s*Deprecated:)`)

// Finding is one comment the rule rejects.
type Finding struct {
	Path    string
	Line    int
	Message string
}

func main() {
	fix := flag.Bool("fix", false, "remove the reported comments")
	flag.Parse()
	var findings []Finding
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		path := strings.TrimSpace(scanner.Text())
		if !strings.HasSuffix(path, ".go") {
			continue
		}
		src, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
			os.Exit(2)
		}
		found, fixed, err := Check(path, src)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
			os.Exit(2)
		}
		if *fix && len(found) > 0 {
			if err := os.WriteFile(path, fixed, 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
				os.Exit(2)
			}
			continue
		}
		findings = append(findings, found...)
	}
	var report strings.Builder
	for _, f := range findings {
		fmt.Fprintf(&report, "%s:%d: %s\n", f.Path, f.Line, f.Message)
	}
	if _, err := os.Stdout.WriteString(report.String()); err != nil {
		os.Exit(2)
	}
	if len(findings) > 0 {
		os.Exit(1)
	}
}

// Check returns the rejected comments of one Go source and the source with them removed.
// A generated file returns no findings; a file that does not parse returns the parse error.
func Check(path string, src []byte) ([]Finding, []byte, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return nil, src, fmt.Errorf("parse: %w", err)
	}
	if ast.IsGenerated(file) {
		return nil, src, nil
	}
	sdk := strings.Contains("/"+path, "/libs/plugin-sdk-go/") && !strings.HasSuffix(path, "_test.go")
	var findings []Finding
	var cuts []span
	report := func(group *ast.CommentGroup, message string) {
		if group == nil {
			return
		}
		reported := false
		for _, c := range group.List {
			if directive.MatchString(c.Text) {
				continue
			}
			cuts = append(cuts, span{fset.Position(c.Pos()).Offset, fset.Position(c.End()).Offset})
			if !reported {
				findings = append(findings, Finding{path, fset.Position(c.Pos()).Line, message})
				reported = true
			}
		}
	}
	ast.Inspect(file, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.StructType:
			for _, field := range x.Fields.List {
				if sdk && len(field.Names) > 0 && field.Names[0].IsExported() {
					continue
				}
				message := "comment on a struct field; the name and type state what the field holds"
				if len(field.Names) > 0 {
					message = fmt.Sprintf("comment on struct field %s; the name and type state what the field holds", field.Names[0].Name)
				}
				report(field.Doc, message)
				report(field.Comment, message)
			}
		case *ast.FuncDecl:
			if !x.Name.IsExported() {
				report(x.Doc, fmt.Sprintf("doc comment on unexported function %s; the name and signature state what it does", x.Name.Name))
			}
		}
		return true
	})
	if len(cuts) == 0 {
		return nil, src, nil
	}
	fixed, err := format.Source(removeSpans(src, cuts))
	if err != nil {
		return nil, src, fmt.Errorf("format after removing comments: %w", err)
	}
	return findings, fixed, nil
}

type span struct{ from, to int }

func removeSpans(src []byte, cuts []span) []byte {
	sort.Slice(cuts, func(i, j int) bool { return cuts[i].from > cuts[j].from })
	out := append([]byte(nil), src...)
	for _, s := range cuts {
		start := s.from
		for start > 0 && (out[start-1] == ' ' || out[start-1] == '\t') {
			start--
		}
		end := s.to
		for end < len(out) && (out[end] == ' ' || out[end] == '\t') {
			end++
		}
		atLineStart := start == 0 || out[start-1] == '\n'
		atLineEnd := end >= len(out) || out[end] == '\n'
		switch {
		case atLineStart && atLineEnd:
			if end < len(out) {
				end++
			}
			out = append(out[:start:start], out[end:]...)
		case atLineEnd:
			out = append(out[:start:start], out[s.to:]...)
		default:
			out = append(out[:s.from:s.from], out[end:]...)
		}
	}
	return out
}
