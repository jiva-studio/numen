package container

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// module is what every package of the core is named under.
const module = "github.com/jiva-studio/numen/modules/libs/core/"

// owed are the edges this installation still has. Each is a package reaching a
// sibling it should be given instead, and the list only shrinks.
var owed = map[string][]string{
	// The terminal opens a vault and reads the settings file itself.
	"adapter/cli": {"adapter/filesystem", "adapter/settings", "container"},
	// The window assembles what it serves.
	"adapter/webui": {"container"},
	// One settings file is the union of every adapter's section.
	"adapter/settings": {
		"adapter/agent", "adapter/embed", "adapter/proofreading",
		"adapter/recognition", "adapter/transcription",
	},
	// The two source queues and the deck writer stand here, so the words for a
	// piece of work, a card, a schedule, a cut and a vector are read in place.
	"container": {
		"cards", "cutting", "embedding", "flashcards", "markdown", "proofread", "task",
	},
}

// The core is reached by the adapters and reaches none of them, an adapter is
// given what it needs rather than taking another, and what is assembled is
// assembled in one place.
//
// A test file is left out: a test stands outside the package it exercises and
// builds the adapters that stand in for the real ones.
func TestTheLayersAreWhatTheyAre(t *testing.T) {
	var wrong []string
	root := ".."
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
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
		from := within(root, path)
		for _, one := range file.Imports {
			to, err := strconv.Unquote(one.Path.Value)
			if err != nil || !strings.HasPrefix(to, module) {
				continue
			}
			if why := refused(from, strings.TrimPrefix(to, module)); why != "" {
				wrong = append(wrong, from+" reaches "+strings.TrimPrefix(to, module)+": "+why)
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
}

// within is the package a file belongs to, as the rules name it.
func within(root, path string) string {
	held, err := filepath.Rel(root, filepath.Dir(path))
	if err != nil {
		return path
	}
	return strings.TrimPrefix(filepath.ToSlash(held), "internal/")
}

// refused says why one package may not reach another, and nothing where it may.
func refused(from, to string) string {
	to = strings.TrimPrefix(to, "internal/")
	for _, held := range owed[from] {
		if to == held || strings.HasPrefix(to, held+"/") {
			return ""
		}
	}

	switch {
	case strings.HasPrefix(from, "adapter/"):
		if to == "container" {
			return "an adapter is given what it needs and assembles nothing"
		}
		if strings.HasPrefix(to, "adapter/") && !sibling(from, to) {
			return "an adapter is given what it needs and takes no other adapter"
		}
	case from == "container":
		if !assembling(to) {
			return "the composition root assembles the core and does none of its work"
		}
	// Fixtures build the real adapters, and only a test is compiled from them.
	case from == "testsupport":
	default:
		if strings.HasPrefix(to, "adapter/") || to == "container" {
			return "the core reaches no adapter and nothing that assembles one"
		}
	}
	return ""
}

// assembling says whether a package is one the composition root puts together:
// an adapter, a scenario, and the two languages the two are named in.
func assembling(to string) bool {
	return to == "container" || to == "domain" || to == "port" ||
		strings.HasPrefix(to, "adapter/") || strings.HasPrefix(to, "usecase/")
}

// sibling says whether two packages are one adapter: its own folder and every
// package under it, which share the settings section they are built from.
func sibling(from, to string) bool { return family(from) == family(to) }

// family is the adapter a package belongs to.
func family(pkg string) string {
	held := strings.Split(pkg, "/")
	if len(held) < 2 {
		return pkg
	}
	return held[0] + "/" + held[1]
}

// Reading the tree is what the rules are checked against, so the tree has to be
// where this expects it.
func TestTheTreeIsWhereTheLayersAreRead(t *testing.T) {
	for _, at := range []string{"domain", "port", "usecase", "adapter", "internal/adapter"} {
		if _, err := os.Stat(filepath.Join("..", at)); err != nil {
			t.Fatalf("%s: %v", at, err)
		}
	}
}
