package settings_test

import (
	"regexp"
	"testing"

	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
)

func TestCreateSecretIDReturnsAnIDThatDiffersFromTheLastOne(t *testing.T) {
	first, err := settingsadapter.CreateSecretID()
	if err != nil {
		t.Fatal(err)
	}
	second, err := settingsadapter.CreateSecretID()
	if err != nil {
		t.Fatal(err)
	}

	if !regexp.MustCompile(`^s[0-9a-f]{8}$`).MatchString(first) || first == second {
		t.Errorf("ids %q and %q", first, second)
	}
}
