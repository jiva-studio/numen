// Package catalog lists the programs of the Antigravity plugin with the models the program offers,
// and chooses the model that a request names.
package catalog

import (
	"context"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/reasoning"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/model"
)

const (
	modelListLimit = 15 * time.Second
	failureBackoff = 30 * time.Second
	programName    = "antigravity"
)

// Lister lists the models of the program that command starts.
type Lister interface {
	List(ctx context.Context, command []string) ([]model.Model, error)
}

// Installation reports if the program is installed, at path when given.
type Installation interface {
	IsInstalled(path string) bool
}

// Catalog keeps the last list of models the program gave.
type Catalog struct {
	lister       Lister
	installation Installation
	models       *cache.Cache[[]model.Model]
}

// New returns a catalog. With isAsync, a list that is not cached is loaded in the background.
func New(lister Lister, installation Installation, isAsync bool) *Catalog {
	return &Catalog{
		lister:       lister,
		installation: installation,
		models: cache.New[[]model.Model](
			"models",
			cache.WithTimeout(modelListLimit),
			cache.WithFailureBackoff(failureBackoff),
			cache.WithAsync(isAsync),
		),
	}
}

// GetState reports if the program is reachable, with the reason when it is not.
func (c *Catalog) GetState(executablePath string) *agentrun.State {
	if !c.installation.IsInstalled(executablePath) {
		return &agentrun.State{Unreachable: "Antigravity was not found on this machine"}
	}
	return &agentrun.State{Unreachable: ""}
}

// GetPrograms lists Antigravity, whether it is installed, and the models it lists.
func (c *Catalog) GetPrograms(
	ctx context.Context, executablePath string, commands []agentrun.ProgramCommand,
) []*agentrun.Program {
	command := getProgramCommand(commands)
	if len(command) == 0 && executablePath != "" {
		command = []string{executablePath}
	}
	antigravity := &agentrun.Program{
		Name: programName, Title: "Antigravity", IsInstalled: c.installation.IsInstalled(executablePath) || len(command) > 0,
		IsDefault: false, Models: nil, LimitsTools: false,
	}
	if antigravity.IsInstalled {
		for _, one := range c.getFamilies(ctx, command) {
			antigravity.Models = append(antigravity.Models, one.GetModel())
		}
	}
	return []*agentrun.Program{antigravity}
}

// ResolveModel replaces the model of req with the listed model that its name and reasoning level choose.
func (c *Catalog) ResolveModel(ctx context.Context, req *agentrun.Request) {
	if req.Model == "" || req.Model == model.DefaultName {
		return
	}
	req.Model = reasoning.ResolveModel(c.getFamilies(ctx, req.Command), req.Model, req.Reasoning)
}

func (c *Catalog) getFamilies(ctx context.Context, command []string) []reasoning.Family {
	list := func(ctx context.Context) ([]model.Model, error) { return c.lister.List(ctx, command) }
	items, _, err := c.models.Get(ctx, strings.Join(command, " "), list)
	// gomutants:disable-next-line BRANCH_IF reason="A failed load returns the cached items folded identically to a succeeded load."
	if err != nil {
		// A failed load leaves the last list of the cache, or none, and the program is offered with it.
		return model.Fold(items)
	}
	return model.Fold(items)
}

func getProgramCommand(commands []agentrun.ProgramCommand) []string {
	for _, one := range commands {
		if one.Program == programName {
			return one.Command
		}
	}
	return nil
}
