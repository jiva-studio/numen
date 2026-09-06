package cli

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

const vaultUsage = `usage:
  numen-cli vault add <path> [--name <name>]
  numen-cli vault list
  numen-cli vault rename <vault> <new name>
  numen-cli vault forget <vault>
  numen-cli vault erase <vault> [--yes]
  numen-cli vault open <vault>`

func vaultCommand(ctx context.Context, out io.Writer, deps Deps, args []string) error {
	if len(args) == 0 {
		return errors.New(vaultUsage)
	}
	switch args[0] {
	case "add":
		return vaultAdd(out, deps, args[1:])
	case "list":
		return vaultList(out, deps)
	case "rename":
		return vaultRename(ctx, out, deps, args[1:])
	case "forget":
		return vaultForget(ctx, out, deps, args[1:])
	case "erase":
		return vaultErase(ctx, out, deps, args[1:])
	case "open":
		return vaultOpen(out, deps, args[1:])
	default:
		return fmt.Errorf("unknown vault command %q", args[0])
	}
}

func vaultAdd(out io.Writer, deps Deps, args []string) error {
	fs := flag.NewFlagSet("vault add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	name := fs.String("name", "", "name for this vault (default: the folder name)")
	rest, err := parseInterspersed(fs, args)
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return errors.New("usage: numen-cli vault add <path> [--name <name>]")
	}

	vaults, err := deps.Vaults()
	if err != nil {
		return err
	}
	root, err := filepath.Abs(rest[0])
	if err != nil {
		return err
	}
	v, err := vault.NewAdd(vaults.Identity, vaults.Registry, vaults.Now).Execute(root, *name)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "added %s\n  path %s\n  id   %s\n", v.Name, v.Path, v.ID)
	return nil
}

func vaultList(out io.Writer, deps Deps) error {
	vaults, err := deps.Vaults()
	if err != nil {
		return err
	}
	// Nobody is sitting in front of a vault here, so the current one is the
	// vault the next window opens.
	known, err := vault.NewKnownVaults(vaults.Registry, vaults.Readers).Execute("")
	if err != nil {
		return err
	}
	if len(known) == 0 {
		fmt.Fprintln(out, "no vaults yet — add one with: numen-cli vault add <path>")
		return nil
	}
	for _, one := range known {
		last := " "
		if one.Current {
			last = "*"
		}
		there := " "
		if one.Missing {
			there = "?"
		}
		fmt.Fprintf(out, "%s%s %-20s %s\n  %s\n",
			last, there, one.Vault.Name, one.Vault.ID, one.Vault.Path)
	}
	return nil
}

func vaultRename(ctx context.Context, out io.Writer, deps Deps, args []string) error {
	if len(args) != 2 {
		return errors.New("usage: numen-cli vault rename <vault> <new name>")
	}
	vaults, err := deps.Vaults()
	if err != nil {
		return err
	}
	v, err := vaults.find(args[0])
	if err != nil {
		return err
	}
	rows, err := vaults.Rows(ctx)
	if err != nil {
		return err
	}
	defer closing(rows.Close)

	renamed, err := vault.NewRename(vaults.Registry, rows.Vaults).Execute(ctx, v, args[1])
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "renamed %s to %s\n  the folder is still %s\n", v.Name, renamed.Name, renamed.Path)
	return nil
}

func vaultForget(ctx context.Context, out io.Writer, deps Deps, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: numen-cli vault forget <vault>")
	}
	vaults, err := deps.Vaults()
	if err != nil {
		return err
	}
	v, err := vaults.find(args[0])
	if err != nil {
		return err
	}
	rows, err := vaults.Rows(ctx)
	if err != nil {
		return err
	}
	defer closing(rows.Close)

	forget := vault.NewForget(vaults.Registry, rows.Vaults)
	if err := forget.Execute(ctx, v); err != nil {
		return err
	}
	fmt.Fprintf(out, "forgot %s\n  the folder is still %s, and adding it again brings back the same vault\n",
		v.Name, v.Path)
	return nil
}

func vaultErase(ctx context.Context, out io.Writer, deps Deps, args []string) error {
	fs := flag.NewFlagSet("vault erase", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	yes := fs.Bool("yes", false, "erase without asking")
	rest, err := parseInterspersed(fs, args)
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return errors.New("usage: numen-cli vault erase <vault> [--yes]")
	}
	vaults, err := deps.Vaults()
	if err != nil {
		return err
	}
	v, err := vaults.find(rest[0])
	if err != nil {
		return err
	}
	if !*yes && !agreed(out, v) {
		return fmt.Errorf("%s was not erased", v.Name)
	}

	rows, err := vaults.Rows(ctx)
	if err != nil {
		return err
	}
	defer closing(rows.Close)

	erase := vault.NewErase(
		vaults.Identity, vaults.Trash, vault.NewForget(vaults.Registry, rows.Vaults),
	)
	res, err := erase.Execute(ctx, v)
	if errors.Is(err, port.ErrNoTrash) {
		return fmt.Errorf("%w — take it off the list with: numen-cli vault forget %s", err, v.Name)
	}
	if err != nil {
		return err
	}

	went := fmt.Sprintf("nothing was at %s", v.Path)
	if res.Trashed {
		went = fmt.Sprintf("%s went to the trash this machine keeps", v.Path)
	}
	fmt.Fprintf(out, "erased %s\n  %s\n", v.Name, went)
	return nil
}

// agreed says what erasing does and reads the answer from the terminal this was
// typed at. Only yes is a yes.
func agreed(out io.Writer, v domain.Vault) bool {
	fmt.Fprintf(out, "erase %s?\n  %s goes to the trash this machine keeps\n"+
		"  the vault leaves the list and the index\ntype yes to erase it: ", v.Name, v.Path)
	answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.EqualFold(strings.TrimSpace(answer), "yes")
}

func vaultOpen(out io.Writer, deps Deps, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: numen-cli vault open <vault>")
	}
	vaults, err := deps.Vaults()
	if err != nil {
		return err
	}
	v, err := vaults.find(args[0])
	if err != nil {
		return err
	}
	if err := vaults.Registry.Opened(v.ID); err != nil {
		return err
	}
	fmt.Fprintf(out, "the next window opens %s\n  path %s\n", v.Name, v.Path)
	return nil
}

// findVault resolves what the person typed, against the list opened for it.
func findVault(deps Deps, nameOrPath string) (domain.Vault, error) {
	vaults, err := deps.Vaults()
	if err != nil {
		return domain.Vault{}, err
	}
	return vaults.find(nameOrPath)
}
