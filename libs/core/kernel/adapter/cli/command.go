package cli

import (
	"context"
	"io"
	"slices"
	"strings"
)

// terminal is where a command writes: out carries its result, errOut what it reports besides.
type terminal struct {
	out    io.Writer
	errOut io.Writer
}

// command is one name the terminal answers to. Synopsis is the text help prints for it, indented as
// printed and without a trailing newline.
type command interface {
	Name() string
	Synopsis() string
	Run(ctx context.Context, term terminal, args []string) error
}

// group is commands that help lists under one heading. The commands are listed when first needed, so
// the plugins are not readied for a built-in command.
type group struct {
	heading string
	list    func() []command
}

// registry holds the built-in commands and those the plugins declare, in the order help lists them.
type registry struct {
	groups []group
}

func newRegistry(deps Deps) registry {
	return registry{groups: []group{
		{list: func() []command { return getBuiltinCommands(deps) }},
		{heading: "commands of the plugins:", list: func() []command { return listDeclaredCommands(deps) }},
		{heading: "shared commands, run by the handler of a plugin:", list: func() []command { return listSharedCommands(deps) }},
	}}
}

func getBuiltinCommands(deps Deps) []command {
	return []command{
		newVaultCommand(deps),
		scanCommand{scan: deps.Scan},
		searchCommand{search: deps.Search},
		linksCommand{links: deps.Links},
		problemsCommand{problems: deps.Problems},
		helpCommand{deps: deps},
	}
}

func listDeclaredCommands(deps Deps) []command {
	if deps.PluginCommands == nil {
		return nil
	}
	var commands []command
	for _, entry := range deps.PluginCommands().List.Execute() {
		commands = append(commands, declaredCommand{plugins: deps.PluginCommands, entry: entry})
	}
	return commands
}

func listSharedCommands(deps Deps) []command {
	if deps.PluginCommands == nil {
		return nil
	}
	list := deps.PluginCommands().List
	var ids []string
	for _, one := range list.ListBackendHandlers() {
		if !slices.Contains(ids, one.Command.ID) {
			ids = append(ids, one.Command.ID)
		}
	}
	var commands []command
	for _, id := range ids {
		commands = append(commands, sharedCommand{plugins: deps.PluginCommands, handlers: list.ListSharedHandlers(id)})
	}
	return commands
}

// find returns the command of the name, searching the groups in order.
func (r registry) find(name string) (command, bool) {
	if name == "-h" || name == "--help" {
		name = "help"
	}
	for _, g := range r.groups {
		for _, one := range g.list() {
			if one.Name() == name {
				return one, true
			}
		}
	}
	return nil, false
}

// helpCommand prints the usage text, built from the commands of the registry.
type helpCommand struct {
	deps Deps
}

func (helpCommand) Name() string { return "help" }

func (helpCommand) Synopsis() string { return "" }

func (c helpCommand) Run(_ context.Context, term terminal, _ []string) error {
	return writeUsage(term.out, newRegistry(c.deps))
}

func writeUsage(out io.Writer, r registry) error {
	w := newReporter(out)
	for i, g := range r.groups {
		var lines []string
		for _, one := range g.list() {
			if synopsis := one.Synopsis(); synopsis != "" {
				lines = append(lines, synopsis)
			}
		}
		switch {
		case i == 0:
			w.Print(usageHead + strings.Join(lines, "\n") + "\n" + usageTail)
		case len(lines) > 0:
			w.Print("\n" + g.heading + "\n" + strings.Join(lines, "\n") + "\n")
		}
	}
	return w.Err()
}
