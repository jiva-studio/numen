package startup_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/startup"
)

type fakeIndex struct{ log *[]string }

func (i fakeIndex) Close() error {
	*i.log = append(*i.log, "close index")
	return nil
}

func newEditorStart(log *[]string) startup.OpenEditor[string, fakeIndex, string] {
	step := func(name string) { *log = append(*log, name) }
	return startup.OpenEditor[string, fakeIndex, string]{
		OpenRegistry: func() (string, error) { step("registry"); return "registry", nil },
		OpenIndex:    func(context.Context) (fakeIndex, error) { step("index"); return fakeIndex{log: log}, nil },
		OpenTexts:    func(fakeIndex) error { step("texts"); return nil },
		OpenWindow: func(context.Context, string, fakeIndex) (string, error) {
			step("window")
			return "window", nil
		},
		FollowVault:  func(string) { step("follow vault") },
		InstallVault: func(string, fakeIndex, string) error { step("install vault"); return nil },
		ShareWindow:  func(fakeIndex, string) { step("share window") },
	}
}

func TestOpenEditorTakesTheStepsInOrder(t *testing.T) {
	var log []string

	window, err := newEditorStart(&log).Execute(t.Context())

	want := []string{"registry", "index", "texts", "window", "follow vault", "install vault", "share window"}
	if err != nil || window != "window" || !slices.Equal(log, want) {
		t.Fatalf("window %q, error %v, steps %v; want steps %v", window, err, log, want)
	}
}

func TestOpenEditorClosesTheIndexWhereTheTextsFail(t *testing.T) {
	var log []string
	start := newEditorStart(&log)
	failure := errors.New("texts")
	start.OpenTexts = func(fakeIndex) error { return failure }

	_, err := start.Execute(t.Context())

	if !errors.Is(err, failure) || !slices.Equal(log, []string{"registry", "index", "close index"}) {
		t.Fatalf("error %v, steps %v", err, log)
	}
}

func TestOpenEditorStopsWhereThePluginVaultCannotBeInstalled(t *testing.T) {
	var log []string
	start := newEditorStart(&log)
	failure := errors.New("install")
	start.InstallVault = func(string, fakeIndex, string) error { return failure }

	_, err := start.Execute(t.Context())

	if !errors.Is(err, failure) || slices.Contains(log, "share window") {
		t.Fatalf("error %v, steps %v", err, log)
	}
}
