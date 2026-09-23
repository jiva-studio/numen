package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

func linksCommand(ctx context.Context, out io.Writer, deps Deps, args []string) error {
	if len(args) != 2 {
		return errors.New("usage: numen-cli links <vault> <note>")
	}
	v, err := findVault(deps, args[0])
	if err != nil {
		return err
	}
	open, err := deps.Links(ctx)
	if err != nil {
		return err
	}
	defer closeIfOpen(open.Close)

	links, err := open.Show.Execute(ctx, v, args[1])
	if err != nil {
		return err
	}

	if len(links.Links) == 0 && len(links.Backlinks) == 0 {
		fmt.Fprintln(out, "no links either way")
		return nil
	}
	if len(links.Links) > 0 {
		fmt.Fprintln(out, "points at:")
		for _, l := range links.Links {
			fmt.Fprintf(out, "  %s\n", describeLink(v, l))
		}
	}
	if len(links.Backlinks) > 0 {
		fmt.Fprintln(out, "pointed at by:")
		for _, l := range links.Backlinks {
			fmt.Fprintf(out, "  %-10s %s\n", l.Role, l.From)
		}
	}
	return nil
}

// describeLink says where a link goes and, when it goes nowhere, says that
// too — a dangling link is a fact about the vault, not an omission.
func describeLink(from domain.Vault, l domain.ResolvedLink) string {
	where := l.To
	switch {
	case where == "" && l.Target.Scheme == domain.SchemeName:
		where = l.Target.Value + "  (nothing by that name)"
	case where == "" && l.Target.Scheme == domain.SchemeNote:
		// Neither resolved nor broken: no connected vault holds it, and which
		// of the two it is cannot be known from here.
		where = l.Target.String() + "  (no connected vault holds this note)"
	case where == "":
		where = l.Target.String()
	case l.IsAmbiguous:
		where += "  (several notes answer to that name)"
	}
	if vault, crossed := l.InVault(from.ID); crossed {
		where += "  (in another vault: " + string(vault) + ")"
	}
	line := fmt.Sprintf("%-10s %s", l.Role, where)
	if l.Type != "" {
		line += "  [" + l.Type + "]"
	}
	if l.Why != "" {
		line += "  — " + l.Why
	}
	return line
}
