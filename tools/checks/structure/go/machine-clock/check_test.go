// Package machineclock is the check `go/machine-clock`, run on the tree with the planted violations that prove it.
package machineclock

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// TestCorePackagesDoNotReadMachineClock checks that the domain has no access to the current time.
// A clock is a port: a scenario that stamps a note or evaluates what is due
// today receives one, and the composition root binds it to this machine's clock.
// A scenario reading the clock itself cannot be exercised at a chosen instant.
func TestCorePackagesDoNotReadMachineClock(t *testing.T) {
	var wrong []string
	var read int
	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		pkg := load.GetPackage("..", path)
		if strings.HasSuffix(path, "_test.go") || load.IsAdapter(pkg) || load.IsShared(pkg) || load.IsMachinery(pkg) {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		read++
		for _, one := range getClockReads(file) {
			wrong = append(wrong, path+" runs "+one+": a scenario is handed a clock and reads no other")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range wrong {
		t.Error(one)
	}

	// A walk that read no file of the core is a rule checked against nothing,
	// and it passes.
	if read < 100 {
		t.Fatalf("%d files of the core read: the walk is not reading it", read)
	}
}

// the clock reads, and lets through timers, the arithmetic over an instant and
// the constructors that read no clock.
func TestClockRule_RejectsClockReads(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "clock.go", `package p

import "time"

func run(origin time.Time) {
	_ = time.Now()
	_ = time.Since(origin)
	_ = time.Until(origin)
	<-time.After(time.Second)
	time.AfterFunc(time.Second, func() {})
	_ = time.NewTimer(time.Second)
	_ = time.NewTicker(time.Second)
	_ = time.Tick(time.Second)
	_ = origin.Add(time.Second)
	_ = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	_ = time.Duration(3)
}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	if reads, want := getClockReads(file), []string{"time.Now", "time.Since", "time.Until"}; !slices.Equal(reads, want) {
		t.Errorf("the rule refuses the reads %v, want %v", reads, want)
	}
}

// TestNothingOutsideAdaptersReadsTheMachineClock verifies this across libs/note, plugins, and apps/desktop.
func TestNothingOutsideAdaptersReadsTheMachineClock(t *testing.T) {
	roots := []string{"../../../note", "../../../../plugins", "../../../../apps/desktop"}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			slash := filepath.ToSlash(path)
			if strings.Contains(slash, "/cmd/") || strings.Contains(slash, "/internal/adapter/") || strings.Contains(slash, "/testsupport/") {
				return nil
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				return err
			}
			for _, one := range getClockReads(file) {
				t.Errorf("%s runs %s: inject a clock port instead of reading the machine clock", path, one)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
