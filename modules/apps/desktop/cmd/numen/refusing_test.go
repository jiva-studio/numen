package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/index"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/modules/libs/core/container"
)

// What stopped the application is said where a person is: in a window, with
// what this build knows about the state it found.
func TestARefusalSaysWhatItFound(t *testing.T) {
	at := filepath.Join(t.TempDir(), "index.db")
	cfg := container.Config{IndexPath: at}

	page, err := refusal{}.page(cfg, settings.TextScaleBounds.Check("appearance.text_scale", 4))
	if err != nil {
		t.Fatal(err)
	}
	said := string(page)

	for _, want := range []string{
		readingTheSettings,        // what could not be read
		"appearance.text_scale",   // which field
		">4<",                     // what was written there
		"as far as the size goes", // what it may be
		"Write a number",          // what to do
	} {
		if !strings.Contains(said, want) {
			t.Errorf("the page does not say %q", want)
		}
	}
}

// An error this build has nothing further to say about is still said.
func TestARefusalSaysAnythingElseTooRatherThanNothing(t *testing.T) {
	page, err := refusal{}.page(container.Config{IndexPath: filepath.Join(t.TempDir(), "i.db")},
		errors.New("the settings could not be read"))
	if err != nil {
		t.Fatal(err)
	}
	if said := string(page); !strings.Contains(said, "the settings could not be read") {
		t.Error("the page says nothing about what stopped it")
	}
}

// A message a person reads is not a place to put markup they did not write.
func TestARefusalDoesNotCarryMarkupOutOfAnError(t *testing.T) {
	page, err := refusal{}.page(container.Config{IndexPath: filepath.Join(t.TempDir(), "i.db")},
		errors.New(`<script>alert("x")</script>`))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(page), "<script>") {
		t.Error("an error was written into the page as markup")
	}
}

// stopping is one state the application can stop in, and the error it stops
// with. The error is the one the state itself produces: what a person reads is
// only as good as what this window can tell apart.
type stopping struct {
	name string
	cfg  container.Config
	why  error
}

func stoppings(t *testing.T) []stopping {
	t.Helper()

	corrupt := filepath.Join(t.TempDir(), "index.db")
	if err := os.WriteFile(corrupt, []byte("PK\x03\x04 not a database"), 0o644); err != nil {
		t.Fatal(err)
	}

	folder := filepath.Join(t.TempDir(), "index.db")
	if err := os.Mkdir(folder, 0o755); err != nil {
		t.Fatal(err)
	}

	shut := filepath.Join(t.TempDir(), "shut")
	if err := os.Mkdir(shut, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(shut, 0o755) })

	unreadable := filepath.Join(t.TempDir(), "numen.json")
	if err := os.WriteFile(unreadable, []byte("{ this is not JSON"), 0o644); err != nil {
		t.Fatal(err)
	}

	return []stopping{
		{
			name: "an index that is not a database",
			cfg:  container.Config{IndexPath: corrupt},
			why:  openIndex(t, corrupt),
		},
		{
			name: "an index path that is a folder",
			cfg:  container.Config{IndexPath: folder},
			why:  openIndex(t, folder),
		},
		{
			name: "an index folder nobody may write in",
			cfg:  container.Config{IndexPath: filepath.Join(shut, "index.db")},
			why:  openIndex(t, filepath.Join(shut, "index.db")),
		},
		{
			name: "a settings file that is not JSON",
			cfg:  container.Config{SettingsPath: unreadable},
			why:  read(t, unreadable),
		},
		{
			name: "a size the settings file does not take",
			cfg:  container.Config{SettingsPath: unreadable},
			why:  settings.TextScaleBounds.Check("appearance.text_scale", 4),
		},
		{
			name: "a size the command line does not take",
			cfg:  container.Config{},
			why:  settings.TextScaleBounds.Check("-text-scale", 4),
		},
	}
}

// openIndex is the trouble an index at a path comes back with.
func openIndex(t *testing.T, path string) error {
	t.Helper()
	db, err := index.Open(t.Context(), path)
	if err == nil {
		db.Close()
		t.Fatalf("%s opened as an index", path)
	}
	return err
}

// read is the trouble a settings file comes back with.
func read(t *testing.T, path string) error {
	t.Helper()
	held := container.DefaultSettings()
	err := settings.OpenAt(path, &held)
	if err == nil {
		t.Fatalf("%s was read as settings", path)
	}
	return err
}

// Every state this window draws says all three things: what could not be
// opened, what was found, and what a person can do about it.
func TestEveryRefusalSaysWhatToDoAboutIt(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes in a folder whatever its permissions say")
	}
	for _, one := range stoppings(t) {
		t.Run(one.name, func(t *testing.T) {
			said := getRefusal(one.cfg, one.why)
			if said.Heading == "" {
				t.Error("the page is drawn under no heading")
			}
			if said.Sentence == "" {
				t.Error("the page says nothing about what stopped it")
			}
			if len(said.Facts) == 0 {
				t.Error("the page holds no facts about the state it found")
			}
			if said.Remedy == "" {
				t.Error("the page offers nothing to do about it")
			}
			for _, held := range said.Facts {
				if held.Content == "" {
					t.Errorf("the page holds %q with nothing beside it", held.Name)
				}
			}
		})
	}
}

// The heading names what could not be opened. The index and a person's own
// settings file are two different things to have to put right, and neither of
// them is a vault.
func TestARefusalNamesWhatCouldNotBeOpened(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes in a folder whatever its permissions say")
	}
	want := map[string]string{
		"an index that is not a database":        openingTheIndex,
		"an index path that is a folder":         openingTheIndex,
		"an index folder nobody may write in":    openingTheIndex,
		"a settings file that is not JSON":       readingTheSettings,
		"a size the settings file does not take": readingTheSettings,
		"a size the command line does not take":  startingAtAll,
	}
	for _, one := range stoppings(t) {
		said := getRefusal(one.cfg, one.why)
		if said.Heading != want[one.name] {
			t.Errorf("%s is drawn under %q, want %q", one.name, said.Heading, want[one.name])
		}
		if strings.Contains(said.Heading, "vault") {
			t.Errorf("%s is drawn under a heading naming a vault", one.name)
		}
	}
}

// The two faults sqlite answers for with one sentence are two states here: a
// path that is a folder is not a folder nobody may write in.
func TestAnIndexThatWillNotOpenIsSaidByWhatIsWrong(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes in a folder whatever its permissions say")
	}
	by := map[string]refusal{}
	for _, one := range stoppings(t) {
		by[one.name] = getRefusal(one.cfg, one.why)
	}

	folder, shut := by["an index path that is a folder"], by["an index folder nobody may write in"]
	if folder.Sentence == shut.Sentence {
		t.Errorf("two faults are said in one sentence: %q", folder.Sentence)
	}
	if !strings.Contains(folder.Remedy, "-index") {
		t.Errorf("a path that is a folder is answered with %q", folder.Remedy)
	}
	if !strings.Contains(shut.Remedy, "permission") {
		t.Errorf("a folder nobody may write in is answered with %q", shut.Remedy)
	}

	corrupt := by["an index that is not a database"]
	if strings.Contains(corrupt.Sentence, "(26)") || strings.Contains(corrupt.Sentence, "not a database") {
		t.Errorf("the page speaks sqlite's language: %q", corrupt.Sentence)
	}
}

// A number a size does not take is answered where it was written: the field in
// the file, or the flag on the command line.
func TestASizeIsAnsweredWhereItWasWritten(t *testing.T) {
	file := getRefusal(container.Config{}, settings.TextScaleBounds.Check("appearance.text_scale", 4))
	if !strings.Contains(file.Remedy, "appearance.text_scale") {
		t.Errorf("a size in the file is answered with %q", file.Remedy)
	}

	line := getRefusal(container.Config{}, settings.TextScaleBounds.Check("-text-scale", 4))
	if !strings.Contains(line.Remedy, "-text-scale") {
		t.Errorf("a size on the command line is answered with %q", line.Remedy)
	}
}
