package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/note"
)

func linksCommand(ctx context.Context, out io.Writer, cfg container.Config, args []string) error {
	if len(args) != 2 {
		return errors.New("usage: numen links <vault> <note>")
	}
	v, err := findVault(cfg, args[0])
	if err != nil {
		return err
	}
	db, err := cfg.OpenIndex(ctx)
	if err != nil {
		return err
	}
	defer db.Close()

	connections, err := note.ShowConnections{Links: db.Links()}.Execute(ctx, v, args[1])
	if err != nil {
		return err
	}

	if len(connections.Links) == 0 && len(connections.Backlinks) == 0 {
		fmt.Fprintln(out, "no links either way")
		return nil
	}
	if len(connections.Links) > 0 {
		fmt.Fprintln(out, "points at:")
		for _, l := range connections.Links {
			fmt.Fprintf(out, "  %s\n", describeLink(v, l))
		}
	}
	if len(connections.Backlinks) > 0 {
		fmt.Fprintln(out, "pointed at by:")
		for _, l := range connections.Backlinks {
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
	case l.Ambiguous:
		where += "  (several notes answer to that name)"
	}
	if vault, crossed := l.InVault(from.ID); crossed {
		where += "  (in another vault: " + vault + ")"
	}
	line := fmt.Sprintf("%-10s %s", l.Role, where)
	if l.Type != "" {
		line += "  [" + l.Type + "]"
	}
	if l.Note != "" {
		line += "  — " + l.Note
	}
	return line
}
