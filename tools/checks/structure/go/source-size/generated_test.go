package sourcesize

import (
	"bytes"
	"go/parser"
	"go/token"
	"io/fs"
	"path"
	"regexp"
	"strings"
)

var (
	generatorNamePattern = regexp.MustCompile(`by (\S+\.(?:mjs|js|ts|go|sh|py))`)
	goPathPattern        = regexp.MustCompile("['\"`]([^'\"`\\s]+\\.go)['\"`]")
	bufOutPattern        = regexp.MustCompile(`(?m)^\s*(?:-\s*)?out:\s*(\S+)`)
)

func getGeneratedHeader(src []byte) (string, bool) {
	file, err := parser.ParseFile(token.NewFileSet(), "", src, parser.PackageClauseOnly|parser.ParseComments)
	if err != nil {
		return "", false
	}
	for _, group := range file.Comments {
		if group.End() > file.Package {
			break
		}
		for _, one := range group.List {
			if strings.HasPrefix(one.Text, "// Code generated") && strings.Contains(one.Text, "DO NOT EDIT") {
				return one.Text, true
			}
		}
	}
	return "", false
}

func isProduced(fsys fs.FS, rel, header string) bool {
	return isBufOutput(fsys, rel) || isScriptOutput(fsys, rel, header)
}

func isBufOutput(fsys fs.FS, rel string) bool {
	for dir := path.Dir(rel); dir != "."; dir = path.Dir(dir) {
		raw, err := fs.ReadFile(fsys, path.Join(dir, "buf.gen.yaml"))
		if err != nil {
			continue
		}
		for _, match := range bufOutPattern.FindAllSubmatch(raw, -1) {
			out := path.Join(dir, string(bytes.Trim(match[1], `"'`)))
			if strings.HasPrefix(rel, out+"/") {
				return true
			}
		}
		return false
	}
	return false
}

func isScriptOutput(fsys fs.FS, rel, header string) bool {
	for _, named := range generatorNamePattern.FindAllStringSubmatch(header, -1) {
		raw, err := fs.ReadFile(fsys, named[1])
		if err != nil {
			continue
		}
		for _, literal := range goPathPattern.FindAllStringSubmatch(string(raw), -1) {
			if path.Join(path.Dir(named[1]), literal[1]) == rel {
				return true
			}
		}
	}
	return false
}
