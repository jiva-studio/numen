package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/usecase/savedsearch"
)

const (
	commandCreate = "search.create"
	argumentTitle = "title"
)

func newCommands(built *parts) (*command.Registry, error) {
	commands := command.New()
	if err := commands.Add(commandCreate, newCreateCommand(built)); err != nil {
		return nil, err
	}
	return commands, nil
}

func newCreateCommand(built *parts) command.Run {
	return func(ctx context.Context, req command.Request) (command.Result, error) {
		vault, err := built.hosts.Window.GetVault(ctx)
		if err != nil {
			return command.Result{}, err
		}
		saved, err := built.notes.Create(ctx, vault.ID, savedsearch.CreateInput{
			Title:  req.Arguments.GetString(argumentTitle),
			Folder: req.Path,
			Query:  "",
		})
		if err != nil {
			return command.Result{}, err
		}
		if len(saved.Errors) > 0 {
			return command.Result{}, errors.New(formatProblems(saved))
		}
		return command.Result{Message: "Created " + saved.Path, OpenedPath: saved.Path}, nil
	}
}

func formatProblems(saved savedsearch.SavedSearch) string {
	lines := make([]string, len(saved.Errors))
	for i, problem := range saved.Errors {
		lines[i] = fmt.Sprintf("%d:%d %s", problem.Line, problem.Column, problem.Message)
	}
	return "the query is not valid: " + strings.Join(lines, "; ")
}
