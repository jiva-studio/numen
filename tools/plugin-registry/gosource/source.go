package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net"
	"path"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// fileFacts is what one Go file shows, read as far as the parser gets when it has syntax errors.
// Tests counts the test and fuzz functions go test runs, in a file named *_test.go. Calls are the
// uses of the network client API of findClientAPIOf, a call with its first string-literal argument
// that is a URL or an address. URLs are the string literals with an http, https, ws or wss scheme,
// and addresses the string literals of the form host:port.
type fileFacts struct {
	Tests     int      `json:"tests"`
	Calls     []call   `json:"calls"`
	URLs      []string `json:"urls"`
	Addresses []string `json:"addresses"`
}

type call struct {
	Name   string `json:"name"`
	Target string `json:"target,omitempty"`
}

// useKind is how a name of a network client API counts: a function by a call, a type by a
// composite literal, a variable or function by any reference.
type useKind int

const (
	usedByCall useKind = iota
	usedByLiteral
	usedByReference
)

type clientAPI struct {
	name string
	use  useKind
}

const (
	httpClient     = "http.Client"
	httpGet        = "http.Get"
	httpNewRequest = "http.NewRequest"
	netDial        = "net.Dial"
	tlsDial        = "tls.Dial"
	websocketDial  = "websocket.Dial"
	grpcDial       = "grpc.Dial"
	dial           = "Dial"
)

func findClientAPIOf(importPath, name string) (clientAPI, bool) {
	switch importPath {
	case "net/http":
		return findHTTPAPI(name)
	case "net":
		return findNetAPI(name)
	case "crypto/tls":
		if slices.Contains([]string{dial, "DialWithDialer"}, name) {
			return clientAPI{tlsDial, usedByCall}, true
		}
	case "google.golang.org/grpc":
		if slices.Contains([]string{dial, "DialContext", "NewClient"}, name) {
			return clientAPI{grpcDial, usedByCall}, true
		}
	case "github.com/gorilla/websocket", "github.com/coder/websocket", "nhooyr.io/websocket", "golang.org/x/net/websocket":
		if slices.Contains([]string{dial, "DefaultDialer"}, name) {
			return clientAPI{websocketDial, usedByReference}, true
		}
	}
	return clientAPI{}, false
}

func findHTTPAPI(name string) (clientAPI, bool) {
	switch name {
	case "Client", "Transport":
		return clientAPI{httpClient, usedByLiteral}, true
	case "DefaultClient", "DefaultTransport":
		return clientAPI{httpClient, usedByReference}, true
	case "Get", "Head", "Post", "PostForm":
		return clientAPI{httpGet, usedByCall}, true
	case "NewRequest", "NewRequestWithContext":
		return clientAPI{httpNewRequest, usedByCall}, true
	}
	return clientAPI{}, false
}

func findNetAPI(name string) (clientAPI, bool) {
	switch name {
	case dial, "DialTimeout":
		return clientAPI{netDial, usedByCall}, true
	case "Dialer":
		return clientAPI{netDial, usedByLiteral}, true
	}
	return clientAPI{}, false
}

// imports resolves an identifier to the packages it can come from: the package of its qualifier,
// or every dot import for a bare identifier.
type imports struct {
	named map[string]string
	dot   []string
}

func readImports(file *ast.File) imports {
	found := imports{named: map[string]string{}}
	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		switch {
		case spec.Name == nil:
			found.named[path.Base(importPath)] = importPath
		case spec.Name.Name == ".":
			found.dot = append(found.dot, importPath)
		case spec.Name.Name != "_":
			found.named[spec.Name.Name] = importPath
		}
	}
	return found
}

func (in imports) readQualifiedName(expr ast.Expr) ([]string, string) {
	switch node := expr.(type) {
	case *ast.SelectorExpr:
		pkg, ok := node.X.(*ast.Ident)
		if !ok {
			return nil, ""
		}
		importPath, ok := in.named[pkg.Name]
		if !ok {
			return nil, ""
		}
		return []string{importPath}, node.Sel.Name
	case *ast.Ident:
		return in.dot, node.Name
	}
	return nil, ""
}

func (in imports) findClientAPI(expr ast.Expr, use useKind) (clientAPI, bool) {
	paths, name := in.readQualifiedName(expr)
	for _, importPath := range paths {
		if api, ok := findClientAPIOf(importPath, name); ok && api.use == use {
			return api, true
		}
	}
	return clientAPI{}, false
}

func (in imports) isName(expr ast.Expr, importPath, name string) bool {
	paths, found := in.readQualifiedName(expr)
	return found == name && slices.Contains(paths, importPath)
}

type fileReader struct {
	in    imports
	facts fileFacts
}

func (r *fileReader) visit(node ast.Node) bool {
	switch node := node.(type) {
	case *ast.ImportSpec:
		return false
	case *ast.BasicLit:
		r.addLiteral(node)
	case *ast.CallExpr:
		r.addUse(node.Fun, usedByCall, readTarget(node.Args))
	case *ast.CompositeLit:
		r.addUse(node.Type, usedByLiteral, "")
	case *ast.SelectorExpr:
		r.addUse(node, usedByReference, "")
		ast.Inspect(node.X, r.visit)
		return false
	case *ast.Ident:
		r.addUse(node, usedByReference, "")
	}
	return true
}

func (r *fileReader) addUse(expr ast.Expr, use useKind, target string) {
	if api, ok := r.in.findClientAPI(expr, use); ok {
		r.facts.Calls = append(r.facts.Calls, call{Name: api.name, Target: target})
	}
}

func (r *fileReader) addLiteral(lit *ast.BasicLit) {
	value, ok := readString(lit)
	switch {
	case !ok:
	case isURL(value):
		r.facts.URLs = append(r.facts.URLs, value)
	case isAddress(value):
		r.facts.Addresses = append(r.facts.Addresses, value)
	}
}

func readSource(name string, src []byte) fileFacts {
	reader := fileReader{facts: fileFacts{Calls: []call{}, URLs: []string{}, Addresses: []string{}}}
	file, err := parser.ParseFile(token.NewFileSet(), name, src, parser.SkipObjectResolution)
	if err != nil && file == nil {
		return reader.facts
	}
	reader.in = readImports(file)
	if strings.HasSuffix(name, "_test.go") {
		reader.facts.Tests = countTests(file, reader.in)
	}
	ast.Inspect(file, reader.visit)
	return reader.facts
}

func countTests(file *ast.File, in imports) int {
	count := 0
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || fn.Type.TypeParams != nil || fn.Type.Results != nil {
			continue
		}
		switch {
		case isTestName(fn.Name.Name, "Test") && hasOnlyParam(fn, in, "T"):
			count++
		case isTestName(fn.Name.Name, "Fuzz") && hasOnlyParam(fn, in, "F"):
			count++
		}
	}
	return count
}

func isTestName(name, prefix string) bool {
	rest, ok := strings.CutPrefix(name, prefix)
	if !ok {
		return false
	}
	first, _ := utf8.DecodeRuneInString(rest)
	return rest == "" || !unicode.IsLower(first)
}

func hasOnlyParam(fn *ast.FuncDecl, in imports, testingType string) bool {
	params := fn.Type.Params.List
	if len(params) != 1 || len(params[0].Names) > 1 {
		return false
	}
	pointer, ok := params[0].Type.(*ast.StarExpr)
	return ok && in.isName(pointer.X, "testing", testingType)
}

func readString(lit *ast.BasicLit) (string, bool) {
	if lit.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(lit.Value)
	return value, err == nil
}

func readTarget(args []ast.Expr) string {
	for _, arg := range args {
		lit, ok := arg.(*ast.BasicLit)
		if !ok {
			continue
		}
		if value, ok := readString(lit); ok && (isURL(value) || isAddress(value)) {
			return value
		}
	}
	return ""
}

func isURL(value string) bool {
	lower := strings.ToLower(value)
	for _, scheme := range [...]string{"http://", "https://", "ws://", "wss://"} {
		if strings.HasPrefix(lower, scheme) {
			return true
		}
	}
	return false
}

func isAddress(value string) bool {
	host, port, err := net.SplitHostPort(value)
	if err != nil || host == "" {
		return false
	}
	_, err = strconv.ParseUint(port, 10, 16)
	return err == nil
}
