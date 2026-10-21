package vault_test

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

type fakeHome struct {
	path string
	err  error
}

func (h fakeHome) GetHomeFolder() (string, error) { return h.path, h.err }

// linked names the folders it is given by a different path, as a link would.
type linked struct {
	port.VaultIdentity
	to map[string]string
}

func (l linked) GetName(path string) string {
	if to, ok := l.to[path]; ok {
		return to
	}
	return path
}

func TestCheckRootRefusesWhatIsNotAFolderForNotes(t *testing.T) {
	t.Parallel()
	sep := string(filepath.Separator)
	home := filepath.Join(sep, "people", "ann")
	shortcut := filepath.Join(sep, "shortcut")
	check := vaults.NewCheckRoot(linked{to: map[string]string{shortcut: home}}, fakeHome{path: home})

	for root, response := range map[string]string{
		"notes":                          "not an absolute path",
		sep:                              "root of this filesystem",
		home:                             "home directory",
		filepath.Join(home, "..", "ann"): "home directory",
		shortcut:                         "home directory",
	} {
		if err := check.Execute(root); err == nil || !strings.Contains(err.Error(), response) {
			t.Errorf("%s was answered with %v, want an error naming %q", root, err, response)
		}
	}
}

func TestCheckRootLetsAFolderInsideTheHomeDirectoryThrough(t *testing.T) {
	t.Parallel()
	home := filepath.Join(string(filepath.Separator), "people", "ann")
	check := vaults.NewCheckRoot(filesystem.VaultIdentity{}, fakeHome{path: home})

	if err := check.Execute(filepath.Join(home, "notes")); err != nil {
		t.Errorf("a folder inside the home directory failed: %v", err)
	}
}

func TestCheckRootLetsAFolderThroughWhereNoHomeFolderIsKnown(t *testing.T) {
	t.Parallel()
	check := vaults.NewCheckRoot(filesystem.VaultIdentity{}, fakeHome{err: errors.New("no home")})

	if err := check.Execute(filepath.Join(string(filepath.Separator), "people", "ann")); err != nil {
		t.Errorf("a folder failed where no home folder is known: %v", err)
	}
}

func TestCheckRootNotMadeRefusesEveryRoot(t *testing.T) {
	t.Parallel()
	root := filepath.Join(string(filepath.Separator), "notes")
	var unmade *vaults.CheckRoot
	if err := unmade.Execute(root); err == nil {
		t.Error("a use case not made let a root through")
	}
	if err := vaults.NewCheckRoot(nil, nil).Execute(root); err == nil {
		t.Error("a use case with nothing to ask let a root through")
	}
}
