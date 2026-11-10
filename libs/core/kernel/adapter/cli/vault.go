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

	"github.com/jiva-studio/numen/libs/core/kernel/domain/machine"
	"github.com/jiva-studio/numen/libs/note/domain"
)

const vaultUsage = `usage:
  numen-cli vault add <path> [--name <name>]
  numen-cli vault list
  numen-cli vault rename <vault> <new name>
  numen-cli vault remove <vault>
  numen-cli vault erase <vault> [--yes]
  numen-cli vault open <vault>`

func runVaultCommand(ctx context.Context, out io.Writer, deps Deps, args []string) error {
	if len(args) == 0 {
		return errors.New(vaultUsage)
	}
	switch args[0] {
	case "add":
		return addVault(out, deps, args[1:])
	case "list":
		return listVaults(out, deps)
	case "rename":
		return renameVault(ctx, out, deps, args[1:])
	case "remove":
		return removeVaultFromList(ctx, out, deps, args[1:])
	case "erase":
		return eraseVault(ctx, out, deps, args[1:])
	case "open":
		return openVault(out, deps, args[1:])
	default:
		return fmt.Errorf("unknown vault command %q", args[0])
	}
}

func addVault(out io.Writer, deps Deps, args []string) error {
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
	v, err := vaults.Add.Execute(root, *name)
	if err != nil {
		return err
	}

	r := newReporter(out)
	r.Printf("added %s\n  path %s\n  id   %s\n", v.Name, v.Path, v.ID)
	return r.Err()
}

func listVaults(out io.Writer, deps Deps) error {
	vaults, err := deps.Vaults()
	if err != nil {
		return err
	}
	// No vault is specified here, so the current one is the vault the next window
	// opens.
	recognized, err := vaults.Known.Execute("")
	if err != nil {
		return err
	}
	r := newReporter(out)
	if len(recognized) == 0 {
		r.Println("no vaults yet — add one with: numen-cli vault add <path>")
		return r.Err()
	}
	for _, one := range recognized {
		last := " "
		if one.IsCurrent {
			last = "*"
		}
		there := " "
		if one.IsMissing {
			there = "?"
		}
		r.Printf("%s%s %-20s %s\n  %s\n",
			last, there, one.Vault.Name, one.Vault.ID, one.Vault.Path)
	}
	return r.Err()
}

func renameVault(ctx context.Context, out io.Writer, deps Deps, args []string) (err error) {
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
	renaming, err := vaults.OpenRename(ctx)
	if err != nil {
		return err
	}
	defer closeInto(&err, renaming.Close)

	renamed, err := renaming.Rename.Execute(ctx, v, args[1])
	if err != nil {
		return err
	}
	r := newReporter(out)
	r.Printf("renamed %s to %s\n  the folder is still %s\n", v.Name, renamed.Name, renamed.Path)
	return r.Err()
}

func removeVaultFromList(ctx context.Context, out io.Writer, deps Deps, args []string) (err error) {
	if len(args) != 1 {
		return errors.New("usage: numen-cli vault remove <vault>")
	}
	vaults, err := deps.Vaults()
	if err != nil {
		return err
	}
	v, err := vaults.find(args[0])
	if err != nil {
		return err
	}
	forgetting, err := vaults.OpenUnregister(ctx)
	if err != nil {
		return err
	}
	defer closeInto(&err, forgetting.Close)

	if err := forgetting.Unregister.Execute(ctx, v); err != nil {
		return err
	}
	r := newReporter(out)
	r.Printf("forgot %s\n  the folder is still %s, and adding it again brings back the same vault\n",
		v.Name, v.Path)
	return r.Err()
}

func eraseVault(ctx context.Context, out io.Writer, deps Deps, args []string) (err error) {
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
	if !*yes {
		isConfirmed, confirmErr := confirmErase(out, v)
		if confirmErr != nil {
			return confirmErr
		}
		if !isConfirmed {
			return fmt.Errorf("%s was not erased", v.Name)
		}
	}

	erasing, err := vaults.OpenErase(ctx)
	if err != nil {
		return err
	}
	defer closeInto(&err, erasing.Close)

	res, err := erasing.Erase.Execute(ctx, v)
	if errors.Is(err, machine.ErrNoTrash) {
		return fmt.Errorf("%w — take it off the list with: numen-cli vault remove %s", err, v.Name)
	}
	if err != nil {
		return err
	}

	went := fmt.Sprintf("nothing was at %s", v.Path)
	if res.IsTrashed {
		went = fmt.Sprintf("%s went to the trash", v.Path)
	}
	r := newReporter(out)
	r.Printf("erased %s\n  %s\n", v.Name, went)
	return r.Err()
}

func confirmErase(out io.Writer, v domain.Vault) (bool, error) {
	r := newReporter(out)
	r.Printf("erase %s?\n  %s goes to the trash\n"+
		"  the vault leaves the list and the index\ntype yes to erase it: ", v.Name, v.Path)
	if err := r.Err(); err != nil {
		return false, err
	}
	answer, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, fmt.Errorf("read confirmation: %w", err)
	}
	return strings.EqualFold(strings.TrimSpace(answer), "yes"), nil
}

func openVault(out io.Writer, deps Deps, args []string) error {
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
	if err := vaults.RecordOpen.Execute(v.ID); err != nil {
		return err
	}
	r := newReporter(out)
	r.Printf("the next window opens %s\n  path %s\n", v.Name, v.Path)
	return r.Err()
}

func findVault(deps Deps, nameOrPath string) (domain.Vault, error) {
	vaults, err := deps.Vaults()
	if err != nil {
		return domain.Vault{}, err
	}
	return vaults.find(nameOrPath)
}
