// Command lint-comments reports comments on Go struct fields, doc comments on unexported
// functions and doc comments of more than five lines on exported symbols. It reads the paths of the files to check from standard input, one per line; with
// -fix it removes those comments and rewrites the files.
//
// The exported fields of a type whose doc comment carries a marker line keep their comments:
//
//	//numen:settings  apps/docs/tools/manual.mjs writes the field comments into the settings reference
//	//numen:api       plugin authors read the field comments in the SDK documentation
//
// Directive lines (//go:, //export, //numen:, gomutants, Deprecated:) are never reported.
package main

import (
	"bufio"
	"bytes"
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

var directive = regexp.MustCompile(`^//(go:|lint:|numen:|export |line |extern |\s*gomutants:|\s*Deprecated:)`)

// maxDocLines is the most lines of text a doc comment on an exported symbol may have.
const maxDocLines = 5

var documentedMarker = regexp.MustCompile(`^//numen:(settings|api)$`)

// Finding is one comment the rule rejects. Long marks a doc comment over maxDocLines, which -fix does not remove.
type Finding struct {
	Path    string
	Line    int
	Message string
	Long    bool
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
		if *fix {
			if !bytes.Equal(fixed, src) {
				if err := os.WriteFile(path, fixed, 0o644); err != nil {
					fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
					os.Exit(2)
				}
			}
			for _, f := range found {
				if f.Long {
					findings = append(findings, f)
				}
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
// A doc comment over maxDocLines lines on an exported symbol is a finding that is not removed.
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
	documented := getDocumentedStructs(file)
	var findings []Finding
	var cuts []span
	reportLength := func(group *ast.CommentGroup, name string) {
		if n := countDocLines(group); n > maxDocLines {
			findings = append(findings, Finding{Path: path, Line: fset.Position(group.Pos()).Line, Message: fmt.Sprintf("doc comment on exported %s has %d lines; A doc comment is at most five lines: say the fact the name and type cannot show", name, n), Long: true})
		}
	}
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
				findings = append(findings, Finding{Path: path, Line: fset.Position(c.Pos()).Line, Message: message})
				reported = true
			}
		}
	}
	ast.Inspect(file, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.StructType:
			for _, field := range x.Fields.List {
				if documented[x] && len(field.Names) > 0 && field.Names[0].IsExported() {
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
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Name.IsExported() {
				reportLength(d.Doc, d.Name.Name)
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				if name, doc := getSpecDoc(d, spec); name != "" {
					reportLength(doc, name)
				}
			}
		}
	}
	if len(cuts) == 0 {
		return findings, src, nil
	}
	fixed, err := format.Source(removeSpans(src, cuts))
	if err != nil {
		return nil, src, fmt.Errorf("format after removing comments: %w", err)
	}
	return findings, fixed, nil
}

func getSpecDoc(decl *ast.GenDecl, spec ast.Spec) (string, *ast.CommentGroup) {
	var name string
	var doc *ast.CommentGroup
	switch s := spec.(type) {
	case *ast.TypeSpec:
		if s.Name.IsExported() {
			name = s.Name.Name
		}
		doc = s.Doc
	case *ast.ValueSpec:
		for _, n := range s.Names {
			if n.IsExported() {
				name = n.Name
				break
			}
		}
		doc = s.Doc
	}
	if doc == nil && len(decl.Specs) == 1 {
		doc = decl.Doc
	}
	return name, doc
}

func countDocLines(group *ast.CommentGroup) int {
	if group == nil {
		return 0
	}
	count := 0
	for _, c := range group.List {
		if directive.MatchString(c.Text) {
			continue
		}
		count += strings.Count(c.Text, "\n") + 1
	}
	return count
}

func getDocumentedStructs(file *ast.File) map[*ast.StructType]bool {
	documented := map[*ast.StructType]bool{}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}
			doc := typeSpec.Doc
			if doc == nil && len(gen.Specs) == 1 {
				doc = gen.Doc
			}
			if hasMarker(doc) {
				documented[structType] = true
			}
		}
	}
	return documented
}

func hasMarker(doc *ast.CommentGroup) bool {
	if doc == nil {
		return false
	}
	for _, c := range doc.List {
		if documentedMarker.MatchString(c.Text) {
			return true
		}
	}
	return false
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
