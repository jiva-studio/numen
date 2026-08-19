package main

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/index"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
)

// What stopped the application is said where a person is: in a window, with
// what this build knows about the state it found.
func TestARefusalSaysWhatItFound(t *testing.T) {
	at := filepath.Join(t.TempDir(), "index.db")
	cfg := container.Config{IndexPath: at}

	page, err := refusal{}.page(cfg, &index.Ahead{Held: 9, Known: 1})
	if err != nil {
		t.Fatal(err)
	}
	said := string(page)

	for _, want := range []string{
		"later version of numen", // what happened
		">9<",                    // the schema the index holds
		">1<",                    // the schema this build knows
		at,                       // which file
		"Update numen",           // what to do
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
