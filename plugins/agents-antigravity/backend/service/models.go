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

	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/internal/adapter/process"
)

const (
	modelListLimit = 15 * time.Second
	failureBackoff = 30 * time.Second
)

// antigravityModel is one model the program lists.
type antigravityModel struct {
	Name  string `json:"name"`
	Title string `json:"title"`
}

func listAntigravityModels(ctx context.Context, command []string) ([]antigravityModel, error) {
	name, args := resolveCommand(command)
	cmd := exec.CommandContext(ctx, name, slices.Concat(args, []string{"models"})...)
	// gomutants:disable-next-line STATEMENT_REMOVE reason="process.Supervise sets OS process-group attributes for child cleanup."
	process.Supervise(cmd)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("list the models of %s: %w", name, err)
	}
	return parseAntigravityModels(out), nil
}

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
