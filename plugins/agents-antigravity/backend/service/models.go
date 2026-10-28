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

	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/internal/adapter/clock"
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
	inFlight  map[string]bool
	isAsync   bool
	now       func() time.Time
}

func (c *modelCache[M]) getNow() time.Time {
	if c.now == nil {
		return clock.Now()
	}
	return c.now()
}

func (c *modelCache[M]) get(ctx context.Context, key string, list func(context.Context) ([]M, error)) []M {
	c.mu.Lock()
	if c.isCached && c.key == key {
		models := c.models
		c.mu.Unlock()
		return models
	}
	if c.failedKey == key && c.getNow().Sub(c.failedAt) < failureBackoff {
		c.mu.Unlock()
		return nil
	}
	if c.isAsync {
		if !c.inFlight[key] {
			if c.inFlight == nil {
				c.inFlight = make(map[string]bool)
			}
			c.inFlight[key] = true
			go c.fetchInBackground(key, list)
		}
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()
	return c.fetchAndStore(ctx, key, list)
}

func (c *modelCache[M]) fetchInBackground(key string, list func(context.Context) ([]M, error)) {
	defer func() {
		c.mu.Lock()
		delete(c.inFlight, key)
		c.mu.Unlock()
	}()
	c.fetchAndStore(context.Background(), key, list)
}

func (c *modelCache[M]) fetchAndStore(ctx context.Context, key string, list func(context.Context) ([]M, error)) []M {
	asking, stop := context.WithTimeout(ctx, modelListLimit)
	defer stop()
	models, err := list(asking)
	c.mu.Lock()
	defer c.mu.Unlock()
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
