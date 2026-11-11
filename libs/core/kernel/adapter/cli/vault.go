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

// vaultCommand dispatches `vault` to the subcommand named by its first argument.
type vaultCommand struct {
	subcommands []command
}

func newVaultCommand(session Session) vaultCommand {
	return vaultCommand{subcommands: []command{
		addVaultCommand{session: session},
		listVaultsCommand{session: session},
		renameVaultCommand{session: session},
		removeVaultCommand{session: session},
		eraseVaultCommand{session: session},
		openVaultCommand{session: session},
	}}
}

func (vaultCommand) Name() string { return "vault" }

func (c vaultCommand) Synopsis() string {
	lines := make([]string, len(c.subcommands))
	for i, sub := range c.subcommands {
		lines[i] = sub.Synopsis()
	}
	return strings.Join(lines, "\n")
}

func (c vaultCommand) Run(ctx context.Context, term terminal, args []string) error {
	if len(args) == 0 {
		return errors.New(vaultUsage)
	}
	for _, sub := range c.subcommands {
		if sub.Name() == args[0] {
			return sub.Run(ctx, term, args[1:])
		}
	}
	return fmt.Errorf("unknown vault command %q", args[0])
}

type addVaultCommand struct {
	session Session
}

func (addVaultCommand) Name() string { return "add" }

func (addVaultCommand) Synopsis() string {
	return "  numen-cli vault add <path> [--name <name>]   add a folder to the vault list"
}

func (c addVaultCommand) Run(_ context.Context, term terminal, args []string) error {
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

	vaults, err := c.session.OpenVaults()
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

	r := newReporter(term.out)
	r.Printf("added %s\n  path %s\n  id   %s\n", v.Name, v.Path, v.ID)
	return r.Err()
}

type listVaultsCommand struct {
	session Session
}

func (listVaultsCommand) Name() string { return "list" }

func (listVaultsCommand) Synopsis() string {
	return "  numen-cli vault list                         show the vaults this installation knows"
}

func (c listVaultsCommand) Run(_ context.Context, term terminal, _ []string) error {
	vaults, err := c.session.OpenVaults()
	if err != nil {
		return err
	}
	// No vault is specified here, so the current one is the vault the next window
	// opens.
	recognized, err := vaults.Known.Execute("")
	if err != nil {
		return err
	}
	r := newReporter(term.out)
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

type renameVaultCommand struct {
	session Session
}

func (renameVaultCommand) Name() string { return "rename" }

func (renameVaultCommand) Synopsis() string {
	return "  numen-cli vault rename <vault> <new name>    change the name of a vault"
}

func (c renameVaultCommand) Run(ctx context.Context, term terminal, args []string) error {
	if len(args) != 2 {
		return errors.New("usage: numen-cli vault rename <vault> <new name>")
	}
	indexed, err := c.session.OpenIndexed(ctx, IndexOptions{})
	if err != nil {
		return err
	}

	renamed, err := indexed.Rename.ExecuteByName(ctx, args[0], args[1])
	if err != nil {
		return err
	}
	r := newReporter(term.out)
	r.Printf("renamed %s to %s\n  the folder is still %s\n", renamed.PreviousName, renamed.Vault.Name, renamed.Vault.Path)
	return r.Err()
}

type removeVaultCommand struct {
	session Session
}

func (removeVaultCommand) Name() string { return "remove" }

func (removeVaultCommand) Synopsis() string {
	return "  numen-cli vault remove <vault>               take a vault off the list, leaving its folder"
}

func (c removeVaultCommand) Run(ctx context.Context, term terminal, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: numen-cli vault remove <vault>")
	}
	indexed, err := c.session.OpenIndexed(ctx, IndexOptions{})
	if err != nil {
		return err
	}

	v, err := indexed.Unregister.ExecuteByName(ctx, args[0])
	if err != nil {
		return err
	}
	r := newReporter(term.out)
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

type eraseVaultCommand struct {
	session Session
}

func (eraseVaultCommand) Name() string { return "erase" }

func (eraseVaultCommand) Synopsis() string {
	return "  numen-cli vault erase <vault> [--yes]        remove it and move its folder to the trash"
}

func (c eraseVaultCommand) Run(ctx context.Context, term terminal, args []string) error {
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
	indexed, err := c.session.OpenIndexed(ctx, IndexOptions{})
	if err != nil {
		return err
	}

	var confirm vault.EraseConfirmation
	if !*yes {
		confirm = eraseConfirmation{out: term.out}
	}
	res, err := indexed.Erase.ExecuteByName(ctx, rest[0], confirm)
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
	r := newReporter(term.out)
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

type openVaultCommand struct {
	session Session
}

func (openVaultCommand) Name() string { return "open" }

func (openVaultCommand) Synopsis() string {
	return "  numen-cli vault open <vault>                 set the vault the next window opens"
}

func (c openVaultCommand) Run(_ context.Context, term terminal, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: numen-cli vault open <vault>")
	}
	vaults, err := c.session.OpenVaults()
	if err != nil {
		return err
	}
	v, err := vaults.RecordOpen.ExecuteByName(args[0])
	if err != nil {
		return err
	}
	r := newReporter(term.out)
	r.Printf("the next window opens %s\n  path %s\n", v.Name, v.Path)
	return r.Err()
}
