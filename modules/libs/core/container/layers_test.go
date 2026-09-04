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
		"chunking", "embedding", "flashcards/format", "flashcards/review",
		"markdown", "proofread", "task",
	},
}

// driving are the adapters something outside comes in through. They call the
// scenarios; a driven adapter stands behind a port and calls none, so what a
// scenario is written over is a port and never an adapter's own answer.
var driving = map[string]bool{
	"adapter/cli":          true,
	"adapter/flashcardsui": true,
	"adapter/mcp":          true,
	"adapter/webui":        true,
}

// pure are the packages holding what is true of a note or a card, and the
// arithmetic over it. They name no port and no scenario, so nothing they answer
// waits on a disk, a database or a clock.
var pure = []string{"domain", "flashcards"}

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

// layers are the folders the tree is laid out in.
var layers = map[string]bool{
	"adapter": true, "container": true, "domain": true, "port": true, "usecase": true,
}

// A package is named at the call site for what it is. An alias naming a layer
// says only which folder the package sits in, which the import path already
// says, and the reader is left with usecase.Add for what is vaults.Add.
//
// The tests are read too: most of a package's call sites are in them.
func TestNoPackageIsImportedUnderItsLayer(t *testing.T) {
	var wrong []string
	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, one := range file.Imports {
			if one.Name == nil || !layers[one.Name.Name] {
				continue
			}
			held, err := strconv.Unquote(one.Path.Value)
			if err != nil {
				return err
			}
			wrong = append(wrong, path+" names "+held+" "+one.Name.Name)
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
		if strings.HasPrefix(to, "usecase/") && !driving[family(from)] {
			return "a driven adapter stands behind a port and runs no scenario"
		}
	case from == "container":
		if !assembling(to) {
			return "the composition root assembles the core and does none of its work"
		}
	// Fixtures build the real adapters, and only a test is compiled from them.
	case from == "testsupport", strings.HasPrefix(from, "testsupport/"):
	default:
		if strings.HasPrefix(to, "adapter/") || to == "container" {
			return "the core reaches no adapter and nothing that assembles one"
		}
		if holds(pure, from) && (to == "port" || strings.HasPrefix(to, "usecase/")) {
			return "what is true of a note or a card is worked out from neither"
		}
	}
	return ""
}

// holds says whether a package is one of these, or stands under one.
func holds(these []string, pkg string) bool {
	for _, one := range these {
		if pkg == one || strings.HasPrefix(pkg, one+"/") {
			return true
		}
	}
	return false
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

// public are the adapters an application names for itself: the four it serves
// something through, and the three it composes or configures.
var public = []string{
	"agent", "cli", "flashcardsui", "index", "mcp", "settings", "webui",
}

// The core's surface is these adapters and no others. A driven adapter nothing
// outside composes sits under internal/adapter, where the compiler holds it, so
// binding it to a port stays this package's work.
func TestTheCoresPublicAdaptersAreTheseAndNoOthers(t *testing.T) {
	held, err := os.ReadDir(filepath.Join("..", "adapter"))
	if err != nil {
		t.Fatal(err)
	}
	standing := make(map[string]bool, len(held))
	for _, one := range held {
		if !one.IsDir() {
			continue
		}
		standing[one.Name()] = true
		if !holds(public, one.Name()) {
			t.Errorf("adapter/%s is public and nothing outside composes it", one.Name())
		}
	}
	for _, one := range public {
		if !standing[one] {
			t.Errorf("adapter/%s is named here and is not there", one)
		}
	}
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
