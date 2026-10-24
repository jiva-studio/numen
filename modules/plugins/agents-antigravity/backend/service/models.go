package service

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"slices"
	"strings"
	"sync"
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

type modelCache[M any] struct {
	mu        sync.Mutex
	key       string
	models    []M
	isCached  bool
	failedKey string
	failedAt  time.Time
	now       func() time.Time
}

func (c *modelCache[M]) getNow() time.Time {
	if c.now == nil {
		return time.Now()
	}
	return c.now()
}

func (c *modelCache[M]) get(ctx context.Context, key string, list func(context.Context) ([]M, error)) []M {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.isCached && c.key == key {
		return c.models
	}
	if c.failedKey == key && c.getNow().Sub(c.failedAt) < failureBackoff {
		return nil
	}
	asking, stop := context.WithTimeout(ctx, modelListLimit)
	defer stop()
	models, err := list(asking)
	if err != nil {
		if ctx.Err() == nil {
			c.failedKey, c.failedAt = key, c.getNow()
		}
		return nil
	}
	c.key, c.models, c.isCached = key, models, true
	c.failedKey = ""
	return c.models
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
