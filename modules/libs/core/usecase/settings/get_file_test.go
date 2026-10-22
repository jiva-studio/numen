package settings_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/settings"
)

func TestGetFileAnswersTheFileAndWhereItExists(t *testing.T) {
	t.Parallel()

	got, err := settings.NewGetFile(read).Execute()
	if err != nil {
		t.Fatal(err)
	}
	if got.Written != `{"v":1}` || got.Path != "/home/settings.json" {
		t.Errorf("the file is %q at %q", got.Written, got.Path)
	}
}

func TestGetFileRefusesWhenTheFileCannotBeRead(t *testing.T) {
	t.Parallel()
	get := settings.NewGetFile(func() (string, string, error) { return "", "", errRead })

	if _, err := get.Execute(); !errors.Is(err, errRead) {
		t.Fatalf("error is %v, want the read error", err)
	}
}
