package container

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// useCaseEdges are the use cases each use case may build on. A use case listed
// with nothing depends on no other.
var useCaseEdges = map[string][]string{
	"note":       nil,
	"source":     nil,
	"embedders":  nil,
	"file":       nil,
	"search":     nil,
	"cards":      {"note"},
	"flashcards": {"cards", "note"},
	"vault":      {"note", "source"},
}

// getUseCase is the use case a package belongs to, and whether it is one.
func getUseCase(pkg string) (name string, isUseCase bool) {
	rest, found := strings.CutPrefix(pkg, "usecase/")
	if !found {
		return "", false
	}
	name, _, _ = strings.Cut(rest, "/")
	return name, true
}

// getUseCaseRefusal says why one package may not reach another, and nothing
// where it may.
func getUseCaseRefusal(from, to string) string {
	source, isSource := getUseCase(from)
	target, isTarget := getUseCase(to)
	if !isSource || !isTarget || source == target {
		return ""
	}
	if slices.Contains(useCaseEdges[source], target) {
		return ""
	}
	return "a use case builds only on the use cases useCaseEdges lists for it"
}

// A use case reaches another only along an edge useCaseEdges declares.
//
// A test file is left out: a test stands outside the package it exercises.
func TestNoUseCaseReachesAnotherExceptAsDeclared(t *testing.T) {
	var wrong []string
	var read, between int
	reached := map[string]bool{}
	err := filepath.WalkDir("../usecase", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		from := getPackage("..", path)
		if name, ok := getUseCase(from); ok {
			reached[name] = true
		}
		for _, one := range file.Imports {
			to, err := strconv.Unquote(one.Path.Value)
			if err != nil || !strings.HasPrefix(to, module) {
				continue
			}
			to = strings.TrimPrefix(to, module)
			read++
			if _, ok := getUseCase(to); ok {
				between++
			}
			if why := getUseCaseRefusal(from, to); why != "" {
				wrong = append(wrong, from+" reaches "+to+": "+why)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range wrong {
		t.Error(one)
	}

	// A walk that read no edge between use cases is a rule checked against
	// nothing, and it passes. The floors are under what the tree holds, and
	// each use case is named because a count cannot say which of them the walk
	// reached.
	if read < 50 {
		t.Fatalf("%d edges of this module read: the walk is not reading the use cases", read)
	}
	if between < 5 {
		t.Fatalf("%d edges between use cases read: the walk is not reading them", between)
	}
	for name := range useCaseEdges {
		if !reached[name] {
			t.Errorf("usecase/%s has no file the walk read: the rule stops at that use case", name)
		}
	}
}

// The declared edges are asked directly: a walk that refuses nothing passes,
// and an entry admitting one edge could admit every edge.
func TestWhatTheUseCaseEdgeRuleRefuses(t *testing.T) {
	for _, one := range []struct {
		from, to  string
		isRefused bool
	}{
		// The declared edges.
		{"usecase/cards", "usecase/note", false},
		{"usecase/flashcards", "usecase/cards", false},
		{"usecase/flashcards", "usecase/note", false},
		{"usecase/vault", "usecase/note", false},
		{"usecase/vault", "usecase/source", false},

		// The same edges reversed, and the edges skipping the declaration.
		{"usecase/note", "usecase/cards", true},
		{"usecase/cards", "usecase/flashcards", true},
		{"usecase/note", "usecase/vault", true},
		{"usecase/source", "usecase/vault", true},
		{"usecase/cards", "usecase/vault", true},
		{"usecase/cards", "usecase/source", true},

		// A use case that builds on none.
		{"usecase/embedders", "usecase/note", true},
		{"usecase/file", "usecase/note", true},
		{"usecase/search", "usecase/note", true},
		{"usecase/note", "usecase/search", true},
		{"usecase/note", "usecase/embedders", true},

		// A package under a use case is that use case.
		{"usecase/note/sub", "usecase/vault", true},
		{"usecase/vault/sub", "usecase/note", false},
		{"usecase/note/sub", "usecase/note", false},

		// What is a use case on neither side is another rule's.
		{"usecase/note", "domain", false},
		{"usecase/note", "port", false},
		{"adapter/mcp", "usecase/note", false},
		{"container", "usecase/vault", false},
	} {
		why := getUseCaseRefusal(one.from, one.to)
		if one.isRefused && why == "" {
			t.Errorf("%s reaches %s and is not refused", one.from, one.to)
		}
		if !one.isRefused && why != "" {
			t.Errorf("%s reaches %s and is refused: %s", one.from, one.to, why)
		}
	}
}
