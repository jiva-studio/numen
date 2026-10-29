package service

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"slices"
	"strings"
	"time"
)

const (
	modelListLimit = 15 * time.Second
	failureBackoff = 30 * time.Second
)

// antigravityModel is one model the program lists.
type antigravityModel struct {
	Name  string
	Title string
}

// listAntigravityModels queries the program for its models: the output of "models", one model per
// line as its name, a tab and its title.
func listAntigravityModels(ctx context.Context, command []string) ([]antigravityModel, error) {
	name, args := resolveCommand(command)
	cmd := exec.CommandContext(ctx, name, slices.Concat(args, []string{"models"})...)
	supervise(cmd)
	out, err := cmd.Output()
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
		name, title, found := cutAntigravityModelLine(lines.Text())
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

func cutAntigravityModelLine(line string) (string, string, bool) {
	if idx := strings.LastIndexByte(line, '\r'); idx >= 0 {
		line = line[idx+1:]
	}
	if name, title, found := strings.Cut(line, "\t"); found {
		return strings.TrimSpace(name), strings.TrimSpace(title), true
	}
	if name, title, found := strings.Cut(line, "  "); found {
		name = strings.TrimSpace(name)
		if name != "" && !strings.Contains(name, " ") {
			return name, strings.TrimSpace(title), true
		}
	}
	return "", "", false
}
