// Command comments reports comments on Go struct fields, doc comments on unexported functions and doc
// comments whose words are the name and the signature of the declaration. It reads the paths of the files to check
// from standard input, one per line; with -fix it removes those comments and rewrites the files.
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

var documentedMarker = regexp.MustCompile(`^//numen:(settings|api)$`)

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
		if *fix {
			if !bytes.Equal(fixed, src) {
				if err := os.WriteFile(path, fixed, 0o644); err != nil {
					fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
					os.Exit(2)
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
	reported := map[*ast.CommentGroup]bool{}
	report := func(group *ast.CommentGroup, message string) {
		if group == nil || reported[group] {
			return
		}
		reported[group] = true
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
	reportRestating(fset, file, report)
	if len(cuts) == 0 {
		return findings, src, nil
	}
	fixed, err := format.Source(removeSpans(src, cuts))
	if err != nil {
		return nil, src, fmt.Errorf("format after removing comments: %w", err)
	}
	return findings, fixed, nil
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

func reportRestating(fset *token.FileSet, file *ast.File, report func(*ast.CommentGroup, string)) {
	// The comment is the group right above the declaration, or the last group between the previous declaration and this one that starts with
	// the name, even with a blank line between.
	pkg := file.Name.Name
	checked := map[*ast.CommentGroup]bool{}
	check := func(group *ast.CommentGroup, kind, name string, names []string) {
		if group == nil || checked[group] || hasMarker(group) {
			return
		}
		checked[group] = true
		text := stripComment(group)
		if matched, total, ok := isRestating(text, names); ok {
			report(group, fmt.Sprintf("doc comment on %s %s restates its name and signature (%d of %d words); say the fact the name and type cannot show", kind, name, matched, total))
		}
	}
	previous := token.NoPos
	for _, decl := range file.Decls {
		start := decl.Pos()
		var doc *ast.CommentGroup
		switch d := decl.(type) {
		case *ast.FuncDecl:
			doc = d.Doc
		case *ast.GenDecl:
			doc = d.Doc
		}
		if doc != nil {
			start = doc.Pos()
		}
		detached := findDetachedDoc(file, previous, start)
		switch d := decl.(type) {
		case *ast.FuncDecl:
			check(firstOf(d.Doc, detached, d.Name.Name), "function", d.Name.Name, listSignatureNames(pkg, d))
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					group := s.Doc
					if group == nil && len(d.Specs) == 1 {
						group = firstOf(d.Doc, detached, s.Name.Name)
					}
					check(group, "type", s.Name.Name, []string{pkg, s.Name.Name})
				case *ast.ValueSpec:
					group := s.Doc
					if group == nil && len(d.Specs) == 1 {
						group = firstOf(d.Doc, detached, s.Names[0].Name)
					}
					names := []string{pkg}
					for _, n := range s.Names {
						names = append(names, n.Name)
					}
					names = append(names, listTypeNames(s.Type)...)
					check(group, "value", s.Names[0].Name, names)
				}
			}
		}
		previous = decl.End()
	}
}

func firstOf(doc, detached *ast.CommentGroup, name string) *ast.CommentGroup {
	if doc != nil {
		return doc
	}
	if detached != nil && strings.HasPrefix(strings.TrimSpace(stripComment(detached)), name) {
		return detached
	}
	return nil
}

func findDetachedDoc(file *ast.File, from, to token.Pos) *ast.CommentGroup {
	// A blank line may separate the group from the declaration that follows.
	var found *ast.CommentGroup
	for _, group := range file.Comments {
		if group.Pos() > from && group.End() < to {
			found = group
		}
	}
	return found
}
