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
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
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
	subcommand, ok := getVaultSubcommands()[args[0]]
	if !ok {
		return fmt.Errorf("unknown vault command %q", args[0])
	}
	return subcommand(ctx, out, deps, args[1:])
}

type vaultSubcommand func(ctx context.Context, out io.Writer, deps Deps, args []string) error

func getVaultSubcommands() map[string]vaultSubcommand {
	return map[string]vaultSubcommand{
		"add": func(_ context.Context, out io.Writer, deps Deps, args []string) error {
			return addVault(out, deps, args)
		},
		"list": func(_ context.Context, out io.Writer, deps Deps, _ []string) error {
			return listVaults(out, deps)
		},
		"rename": func(ctx context.Context, out io.Writer, deps Deps, args []string) error {
			return renameVault(ctx, deps, out, args)
		},
		"remove": func(ctx context.Context, out io.Writer, deps Deps, args []string) error {
			return removeVaultFromList(ctx, deps, out, args)
		},
		"erase": func(ctx context.Context, out io.Writer, deps Deps, args []string) error {
			return eraseVault(ctx, deps, out, args)
		},
		"open": func(_ context.Context, out io.Writer, deps Deps, args []string) error {
			return openVault(out, deps, args)
		},
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

func renameVault(ctx context.Context, deps Deps, out io.Writer, args []string) (err error) {
	if len(args) != 2 {
		return errors.New("usage: numen-cli vault rename <vault> <new name>")
	}
	vaults, err := deps.Vaults()
	if err != nil {
		return err
	}
	renaming, err := vaults.OpenRename(ctx)
	if err != nil {
		return err
	}
	defer closeInto(&err, renaming.Close)

	renamed, err := renaming.Rename.ExecuteByName(ctx, args[0], args[1])
	if err != nil {
		return err
	}
	r := newReporter(out)
	r.Printf("renamed %s to %s\n  the folder is still %s\n", renamed.PreviousName, renamed.Vault.Name, renamed.Vault.Path)
	return r.Err()
}

func removeVaultFromList(ctx context.Context, deps Deps, out io.Writer, args []string) (err error) {
	if len(args) != 1 {
		return errors.New("usage: numen-cli vault remove <vault>")
	}
	vaults, err := deps.Vaults()
	if err != nil {
		return err
	}
	forgetting, err := vaults.OpenUnregister(ctx)
	if err != nil {
		return err
	}
	defer closeInto(&err, forgetting.Close)

	v, err := forgetting.Unregister.ExecuteByName(ctx, args[0])
	if err != nil {
		return err
	}
	r := newReporter(out)
	r.Printf("forgot %s\n  the folder is still %s, and adding it again brings back the same vault\n",
		v.Name, v.Path)
	return r.Err()
}

type eraseConfirmation struct {
	out io.Writer
}

func (c eraseConfirmation) ConfirmErase(v domain.Vault) (bool, error) {
	return confirmErase(c.out, v)
}

func eraseVault(ctx context.Context, deps Deps, out io.Writer, args []string) (err error) {
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
	erasing, err := vaults.OpenErase(ctx)
	if err != nil {
		return err
	}
	defer closeInto(&err, erasing.Close)

	var confirm vault.EraseConfirmation
	if !*yes {
		confirm = eraseConfirmation{out: out}
	}
	res, err := erasing.Erase.ExecuteByName(ctx, rest[0], confirm)
	if errors.Is(err, machine.ErrNoTrash) {
		return fmt.Errorf("%w — take it off the list with: numen-cli vault remove %s", err, res.Vault.Name)
	}
	if err != nil {
		return err
	}

	went := fmt.Sprintf("nothing was at %s", res.Vault.Path)
	if res.IsTrashed {
		went = fmt.Sprintf("%s went to the trash", res.Vault.Path)
	}
	r := newReporter(out)
	r.Printf("erased %s\n  %s\n", res.Vault.Name, went)
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
	v, err := vaults.RecordOpen.ExecuteByName(args[0])
	if err != nil {
		return err
	}
	r := newReporter(out)
	r.Printf("the next window opens %s\n  path %s\n", v.Name, v.Path)
	return r.Err()
}
