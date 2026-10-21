package service

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"slices"
	"strings"
)

// antigravityModel is one model the program lists.
type antigravityModel struct {
	Name  string
	Title string
}

// listAntigravityModels asks the program for its models: the output of "models", one model to a
// line as its name, a tab and its title.
func listAntigravityModels(ctx context.Context, command []string) ([]antigravityModel, error) {
	name, args := resolveCommand(command)
	out, err := exec.CommandContext(ctx, name, slices.Concat(args, []string{"models"})...).Output()
	if err != nil {
		return nil, fmt.Errorf("list the models of %s: %w", name, err)
	}
	return parseAntigravityModels(out), nil
}

// parseAntigravityModels reads the lines of "models" that hold a name and a title. Any other line,
// such as the notice that the models are being fetched, is skipped.
func parseAntigravityModels(output []byte) []antigravityModel {
	var models []antigravityModel
	lines := bufio.NewScanner(bytes.NewReader(output))
	for lines.Scan() {
		name, title, found := strings.Cut(lines.Text(), "\t")
		name, title = strings.TrimSpace(name), strings.TrimSpace(title)
		if !found || name == "" {
			continue
		}
		if title == "" {
			title = name
		}
		models = append(models, antigravityModel{Name: name, Title: title})
	}
	return models
}
