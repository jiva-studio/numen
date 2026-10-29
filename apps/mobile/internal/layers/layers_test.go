// Package layers holds nothing. What is here is the rule this application is
// read against, over the Go the phone is built from.
package layers

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// named are files the walk has to have reached: the binding the phone starts
// the core through. A count says how much was read and never what.
var named = []string{"bind/mobile.go"}

// standing is a file of this application, parsed.
type standing struct {
	path string
	file *ast.File
}

// readSources is every hand-written Go file of this application, parsed.
//
// A test file is left out: a test stands outside the package it exercises and
func readSources(t *testing.T) []standing {
	t.Helper()

	var found []standing
	root := filepath.Join("..", "..")
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			if entry != nil && entry.IsDir() && entry.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return err
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		found = append(found, standing{
			path: filepath.ToSlash(strings.TrimPrefix(path, root+string(filepath.Separator))),
			file: file,
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, one := range named {
		if !slices.ContainsFunc(found, func(target standing) bool { return target.path == one }) {
			t.Fatalf("the walk did not read %s, so the rule stops before it", one)
		}
	}
	return found
}

// logging are the words a package that keeps a record is named by, read off the
// last element of an import path. The standard library holds two of them, and
// the rest are the libraries reached for when one of those will not do.
var logging = map[string]bool{
	"log": true, "slog": true, "logrus": true, "zap": true,
	"zerolog": true, "logr": true, "glog": true, "klog": true,
}

// logs answers whether an import is a logger.
func logs(to string) bool { return logging[to[strings.LastIndex(to, "/")+1:]] }

// Nothing here logs. The binding is where the process's own streams are named,
// and what it writes to standard error is for whatever the platform collects;
// everything a person has to act on is said in the window they are looking at.
func TestNoApplicationLogs(t *testing.T) {
	var wrong []string
	for _, target := range readSources(t) {
		for _, one := range target.file.Imports {
			to, err := strconv.Unquote(one.Path.Value)
			if err != nil {
				t.Fatal(err)
			}
			if logs(to) {
				wrong = append(wrong, target.path+" imports "+to)
			}
		}
	}
	for _, one := range wrong {
		t.Error(one + ": nothing here logs — a person is told where they are")
	}
}

// find both of the standard library's and a third party's, and let through the
// packages whose names only begin the same way.
func TestLoggingRule_RejectsLoggingImports(t *testing.T) {
	var violations []string
	for _, to := range []string{
		"log", "log/slog", "go.uber.org/zap", "github.com/rs/zerolog",
		"logic", "text/template",
		"github.com/jiva-studio/numen/libs/notes/domain",
	} {
		if logs(to) {
			violations = append(violations, to)
		}
	}
	want := []string{"log", "log/slog", "go.uber.org/zap", "github.com/rs/zerolog"}
	if !slices.Equal(violations, want) {
		t.Errorf("the rule rejects %v, want %v", violations, want)
	}
}
