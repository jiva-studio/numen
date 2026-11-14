// Package model holds the models the Antigravity program lists, read from its output and grouped
// by reasoning level.
package model

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/domain/reasoning"
)

// DefaultName is the model name that selects the program's own default model.
const DefaultName = "default"

// Model is one model the program lists.
type Model struct {
	Name  string `json:"name"`
	Title string `json:"title"`
}

// Parse reads the models from the output of the program's "models" command.
func Parse(output []byte) []Model {
	var models []Model
	lines := bufio.NewScanner(bytes.NewReader(output))
	for lines.Scan() {
		name, title := cutLine(lines.Text())
		if name == "" {
			continue
		}
		if title == "" {
			title = name
		}
		models = append(models, Model{Name: name, Title: title})
	}
	return models
}

// Fold groups the models that differ only in their reasoning level into one family.
func Fold(items []Model) []reasoning.Family {
	entries := make([]reasoning.Listed, 0, len(items))
	for _, one := range items {
		entries = append(entries, reasoning.Listed{Name: one.Name, Title: one.Title})
	}
	return reasoning.Fold(entries)
}

func cutLine(line string) (string, string) {
	parts := strings.Split(line, "\r")
	line = parts[len(parts)-1]
	if name, title, found := strings.Cut(line, "\t"); found {
		return strings.TrimSpace(name), strings.TrimSpace(title)
	}
	if name, title, found := strings.Cut(line, "  "); found {
		name = strings.TrimSpace(name)
		if !strings.Contains(name, " ") {
			return name, strings.TrimSpace(title)
		}
	}
	return "", ""
}
