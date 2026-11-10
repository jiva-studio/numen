package startup_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/startup"
)

func newFlashcardsStart(log, messages *[]string) startup.OpenFlashcards[string, fakeIndex, string] {
	step := func(name string) { *log = append(*log, name) }
	return startup.OpenFlashcards[string, fakeIndex, string]{
		Label:        "numen-flashcards",
		OpenRegistry: func() (string, error) { step("registry"); return "registry", nil },
		OpenIndex:    func(context.Context) (fakeIndex, error) { step("index"); return fakeIndex{log: log}, nil },
		OpenVaults:   func(fakeIndex) string { step("vaults"); return "vaults" },
		BindWindow:   func(context.Context, string, fakeIndex, string) { step("bind window") },
		InstallVault: func(string, fakeIndex) error { step("install vault"); return nil },
		FollowChanges: func(context.Context) (<-chan struct{}, error) {
			step("follow changes")
			return make(chan struct{}), nil
		},
		ReportChanges: func(context.Context, <-chan struct{}) { step("report changes") },
		BindThemes: func(say func(string)) error {
			step("themes")
			say("")
			say("the themes folder is missing")
			return nil
		},
		Notify: func(message string) { *messages = append(*messages, message) },
	}
}

func TestOpenFlashcardsTakesTheStepsInOrderAndSaysWhatThemesAnswer(t *testing.T) {
	var log, messages []string

	opened, err := newFlashcardsStart(&log, &messages).Execute(t.Context())

	want := []string{"registry", "index", "vaults", "bind window", "install vault", "follow changes", "report changes", "themes"}
	if err != nil || opened.Vaults != "vaults" || !slices.Equal(log, want) {
		t.Fatalf("opened %+v, error %v, steps %v; want steps %v", opened, err, log, want)
	}
	if !slices.Equal(messages, []string{"themes: the themes folder is missing"}) {
		t.Errorf("messages = %v", messages)
	}
}

func TestOpenFlashcardsSaysThatTheIndexIsNotFollowedAndGoesOn(t *testing.T) {
	var log, messages []string
	start := newFlashcardsStart(&log, &messages)
	start.FollowChanges = func(context.Context) (<-chan struct{}, error) { return nil, errors.New("watch refused") }

	opened, err := start.Execute(t.Context())

	if err != nil || slices.Contains(log, "report changes") || !slices.Contains(log, "themes") {
		t.Fatalf("opened %+v, error %v, steps %v", opened, err, log)
	}
	if want := "numen-flashcards: the index is not being followed: watch refused"; messages[0] != want {
		t.Errorf("message = %q, want %q", messages[0], want)
	}
}

func TestOpenFlashcardsReturnsWhatItOpenedWhereThePluginVaultFails(t *testing.T) {
	var log, messages []string
	start := newFlashcardsStart(&log, &messages)
	failure := errors.New("install")
	start.InstallVault = func(string, fakeIndex) error { return failure }

	opened, err := start.Execute(t.Context())

	if !errors.Is(err, failure) || opened.Vaults != "vaults" || slices.Contains(log, "follow changes") {
		t.Fatalf("opened %+v, error %v, steps %v", opened, err, log)
	}
}
