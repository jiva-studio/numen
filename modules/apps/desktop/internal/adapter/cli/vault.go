package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/vault"
)

func vaultCommand(ctx context.Context, out io.Writer, cfg container.Config, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: numen vault add <path> | numen vault list")
	}
	switch args[0] {
	case "add":
		return vaultAdd(out, cfg, args[1:])
	case "list":
		return vaultList(out, cfg)
	default:
		return fmt.Errorf("unknown vault command %q", args[0])
	}
}

func vaultAdd(out io.Writer, cfg container.Config, args []string) error {
	fs := flag.NewFlagSet("vault add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	name := fs.String("name", "", "name for this vault (default: the folder name)")
	rest, err := parseInterspersed(fs, args)
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return errors.New("usage: numen vault add <path> [--name <name>]")
	}

	registry, err := cfg.Registry()
	if err != nil {
		return err
	}
	root, err := filepath.Abs(rest[0])
	if err != nil {
		return err
	}
	// Reached through a symlink, the same folder has two names, and the second
	// one would look like a copy of itself.
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	}

	v, err := vault.Add{
		Identity: cfg.VaultIdentity(),
		Registry: registry,
		Now:      time.Now,
	}.Execute(root, *name)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "added %s\n  path %s\n  id   %s\n", v.Name, v.Path, v.ID)
	return nil
}

func vaultList(out io.Writer, cfg container.Config) error {
	registry, err := cfg.Registry()
	if err != nil {
		return err
	}
	known, err := vault.List{Registry: registry}.Execute()
	if err != nil {
		return err
	}
	if len(known) == 0 {
		fmt.Fprintln(out, "no vaults yet — add one with: numen vault add <path>")
		return nil
	}
	for _, v := range known {
		marker := " "
		if _, err := os.Stat(v.Path); err != nil {
			// The registry remembers where a vault was last seen; the vault
			// carries the identity. A folder that is not there is worth saying
			// out loud rather than failing on later.
			marker = "?"
		}
		fmt.Fprintf(out, "%s %-20s %s\n  %s\n", marker, v.Name, v.ID, v.Path)
	}
	return nil
}
