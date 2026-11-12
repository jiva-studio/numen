package cli_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/cli"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/machine"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func isolate(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
}

// brokenRegistry is the list of a session with one call that fails.
type brokenRegistry struct {
	list   registry.Ports
	breaks string
}

func (b brokenRegistry) ports() registry.Ports {
	return registry.Ports{
		Vaults: b, Lookup: b.list.Lookup, Saver: b, Remover: b, Recorder: b, Last: b.list.Last,
	}
}

func (b brokenRegistry) List() ([]domain.Vault, error) {
	if b.breaks == "List" {
		return nil, errRefused
	}
	return b.list.Vaults.List()
}

func (b brokenRegistry) Save(v domain.Vault) error {
	if b.breaks == "Save" {
		return errRefused
	}
	return b.list.Saver.Save(v)
}

func (b brokenRegistry) Remove(id domain.VaultID) error {
	if b.breaks == "Remove" {
		return errRefused
	}
	return b.list.Remover.Remove(id)
}

func (b brokenRegistry) RecordOpened(id domain.VaultID) error {
	if b.breaks == "RecordOpened" {
		return errRefused
	}
	return b.list.Recorder.RecordOpened(id)
}

func (s *session) runBrokenVault(adjust func(*vaultParts), args ...string) (string, error) {
	s.t.Helper()
	var out bytes.Buffer
	err := cli.Run(s.t.Context(), &out, &s.reply, append(s.base, args...), func(where cli.Locations) cli.Session {
		parts := s.parts(where)
		adjust(&parts)
		return s.depsOver(where, parts).session()
	})
	return out.String(), err
}

func TestVaultCommand_RequiresSubcommand(t *testing.T) {
	t.Parallel()
	_, err := runWithDeps(stubSession{Vaults: createVaultList(getNotebook())}, "vault")
	if err == nil || !strings.Contains(err.Error(), "usage:\n  numen-cli vault add <path>") {
		t.Errorf("vault with no command answered %v", err)
	}
}

func TestAVaultCommandNeedsItsArguments(t *testing.T) {
	t.Parallel()
	for usage, args := range map[string][]string{
		"usage: numen-cli vault add <path> [--name <name>]": {"vault", "add"},
		"usage: numen-cli vault rename <vault> <new name>":  {"vault", "rename", "notebook"},
		"usage: numen-cli vault remove <vault>":             {"vault", "remove"},
		"usage: numen-cli vault erase <vault> [--yes]":      {"vault", "erase"},
		"usage: numen-cli vault open <vault>":               {"vault", "open"},
	} {
		out, err := runWithDeps(stubSession{Vaults: createVaultList(getNotebook())}, args...)
		if err == nil || err.Error() != usage {
			t.Errorf("%v answered %v, want %q", args, err, usage)
		}
		if out != "" {
			t.Errorf("%v said something before it failed:\n%s", args, out)
		}
	}
}

func TestAVaultCommandReturnsWhatWentWrongOpeningTheList(t *testing.T) {
	t.Parallel()
	provided := stubSession{
		Vaults:  func() (cli.Vaults, error) { return cli.Vaults{}, errRefused },
		Indexed: func(context.Context, cli.IndexOptions) (cli.Indexed, error) { return cli.Indexed{}, errRefused },
	}
	for _, args := range [][]string{
		{"vault", "add", "somewhere"},
		{"vault", "list"},
		{"vault", "rename", "notebook", "other"},
		{"vault", "remove", "notebook"},
		{"vault", "erase", "notebook", "--yes"},
		{"vault", "open", "notebook"},
	} {
		out, err := runWithDeps(provided, args...)
		if !errors.Is(err, errRefused) {
			t.Errorf("%v answered %v", args, err)
		}
		if out != "" {
			t.Errorf("%v said something before it failed:\n%s", args, out)
		}
	}
}

func TestAFlagMistypedInAVaultCommandIsReportedAndNothingIsSaved(t *testing.T) {
	// Not parallel: the standard flag package reports on the process's stderr.
	isolate(t)
	file, err := os.Create(filepath.Join(t.TempDir(), "stderr"))
	if err != nil {
		t.Fatal(err)
	}
	was := os.Stderr
	os.Stderr = file
	t.Cleanup(func() { os.Stderr = was })
	testsupport.CloseAfter(t, file)

	wasOpened := false
	provided := stubSession{Vaults: func() (cli.Vaults, error) { wasOpened = true; return cli.Vaults{}, errRefused }}
	for _, args := range [][]string{
		{"vault", "add", "--no-such-flag"},
		{"vault", "add", "somewhere", "--no-such-flag"},
		{"vault", "erase", "--no-such-flag"},
		{"vault", "erase", "notebook", "--no-such-flag"},
	} {
		_, err := runWithDeps(provided, args...)
		if err == nil || !strings.Contains(err.Error(), "no-such-flag") {
			t.Errorf("%v answered %v", args, err)
		}
	}
	if wasOpened {
		t.Error("the list was opened for a command line that did not parse")
	}
	response, readErr := os.ReadFile(file.Name())
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(response) != 0 {
		t.Errorf("the flag package wrote to the terminal:\n%s", response)
	}
}

func TestVaultCommand_ReturnsVaultListError(t *testing.T) {
	isolate(t)
	for name, c := range map[string]struct {
		breaks string
		args   []string
	}{
		"listing the vaults":            {"List", []string{"vault", "list"}},
		"renaming a vault":              {"Save", []string{"vault", "rename", "first", "renamed"}},
		"forgetting a vault":            {"Remove", []string{"vault", "remove", "first"}},
		"erasing a vault":               {"Remove", []string{"vault", "erase", "first", "--yes"}},
		"recording the vault to open":   {"RecordOpened", []string{"vault", "open", "first"}},
		"listing the vaults to rename":  {"List", []string{"vault", "rename", "first", "renamed"}},
		"listing the vaults to forget":  {"List", []string{"vault", "remove", "first"}},
		"listing the vaults to erase":   {"List", []string{"vault", "erase", "first", "--yes"}},
		"listing the vaults to add one": {"List", []string{"vault", "add", t.TempDir()}},
	} {
		s := newSession(t)
		s.mustRun("vault", "add", s.vault, "--name", "first")
		s.addVault("second")
		b := s.trash(nil)

		out, err := s.runBrokenVault(func(v *vaultParts) {
			v.ports = brokenRegistry{list: v.ports, breaks: c.breaks}.ports()
		}, c.args...)
		if !errors.Is(err, errRefused) {
			t.Errorf("%s answered %v", name, err)
		}
		if out != "" {
			t.Errorf("%s said something before it failed:\n%s", name, out)
		}
		if c.args[1] == "erase" && c.breaks == "List" && len(b.took) != 0 {
			t.Errorf("%s: a folder went to the trash: %v", name, b.took)
		}
	}
}

func TestVaultCommand_ReturnsIndexError(t *testing.T) {
	isolate(t)
	for _, args := range [][]string{
		{"vault", "rename", "first", "renamed"},
		{"vault", "remove", "first"},
		{"vault", "erase", "first", "--yes"},
	} {
		s := newSession(t)
		s.mustRun("vault", "add", s.vault, "--name", "first")
		s.addVault("second")
		b := s.trash(nil)

		out, err := s.runBrokenVault(func(v *vaultParts) {
			v.rows = func(context.Context) (vaultport.VaultRepository, func() error, error) { return nil, nil, errRefused }
		}, args...)
		if !errors.Is(err, errRefused) {
			t.Errorf("%v answered %v", args, err)
		}
		if out != "" {
			t.Errorf("%v said something before it failed:\n%s", args, out)
		}
		if len(b.took) != 0 {
			t.Errorf("%v: a folder went to the trash: %v", args, b.took)
		}
		if items := s.mustRun("vault", "list"); !strings.Contains(items, "first") {
			t.Errorf("%v: the vault left the list:\n%s", args, items)
		}
	}
}

func TestVaultRemove_IndexErrorKeepsListAndFolder(t *testing.T) {
	isolate(t)
	s := newSession(t)
	s.mustRun("vault", "add", s.vault, "--name", "first")
	s.addVault("second")
	s.trash(nil)

	for _, args := range [][]string{{"vault", "remove", "first"}, {"vault", "erase", "first", "--yes"}} {
		out, err := s.runBrokenVault(func(v *vaultParts) {
			open := v.rows
			v.rows = func(ctx context.Context) (vaultport.VaultRepository, func() error, error) {
				rows, closeRows, err := open(ctx)
				if err != nil {
					return rows, closeRows, err
				}
				return refusingToForget{rows}, closeRows, nil
			}
		}, args...)
		if !errors.Is(err, errRefused) {
			t.Errorf("%v answered %v", args, err)
		}
		if strings.Contains(out, "forgot") {
			t.Errorf("%v said it forgot:\n%s", args, out)
		}
	}
	if items := s.mustRun("vault", "list"); !strings.Contains(items, "first") {
		t.Errorf("the vault left the list:\n%s", items)
	}
}

// refusingToForget is the rows of an index that cannot take a vault out.
type refusingToForget struct{ vaultport.VaultRepository }

func (refusingToForget) Unregister(context.Context, domain.VaultID) error { return errRefused }

func TestErasingReturnsWhereTheFolderWent(t *testing.T) {
	isolate(t)
	s := newSession(t)
	s.mustRun("vault", "add", s.vault, "--name", "erased")
	s.addVault("other")
	s.trash(nil)

	out := s.mustRun("vault", "erase", "erased", "--yes")
	want := "erased erased\n  " + s.vault + " went to the trash\n"
	if out != want {
		t.Errorf("vault erase said\n%q\nwant\n%q", out, want)
	}
}

func TestErasingAVaultWhoseFolderIsMissingReturnsThereWasNothing(t *testing.T) {
	isolate(t)
	s := newSession(t)
	s.mustRun("vault", "add", s.vault, "--name", "vanished")
	s.addVault("other")
	b := s.trash(nil)
	if err := os.RemoveAll(s.vault); err != nil {
		t.Fatal(err)
	}

	out := s.mustRun("vault", "erase", "vanished", "--yes")
	want := "erased vanished\n  nothing was at " + s.vault + "\n"
	if out != want {
		t.Errorf("vault erase said\n%q\nwant\n%q", out, want)
	}
	if len(b.took) != 0 {
		t.Errorf("a folder went to the trash: %v", b.took)
	}
}

func TestAMachineWithNowhereToInsertItReturnsHowToForgetTheVault(t *testing.T) {
	isolate(t)
	s := newSession(t)
	s.mustRun("vault", "add", s.vault, "--name", "kept")
	s.addVault("other")
	s.trash(machine.ErrNoTrash)

	_, err := s.run("vault", "erase", "kept", "--yes")
	if err == nil || !strings.HasSuffix(err.Error(), " — take it off the list with: numen-cli vault remove kept") {
		t.Errorf("erasing answered %v", err)
	}
}
