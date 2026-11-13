package task

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	contract "github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/acp"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/domain/acp"
)

// conversation is a session of a program and the endpoint that serves it the tools, for the turns of one
// conversation. A turn holds the token of turn while it runs. lastUsed and isGone belong to the pool and
// are guarded by its lock. The first turn of a session is prompted with the distinct instruction sentences
// of the served tools, in the order of the listing, before the attached notes and the prompt. A forced
// close stops the program without ending the session, which also ends a turn that still runs.
type conversation struct {
	key  string
	turn chan struct{}

	mu       sync.Mutex
	session  *acp.Session
	served   ServedTools
	isClosed bool

	listed       map[string]*tools.ToolInfo
	instructions string
	isPrompted   bool

	scopeMu     sync.Mutex
	toolContext map[string]string
	vaultID     string
	connection  string

	lastUsed time.Time
	isGone   bool
}

func newConversation(key string, now time.Time) *conversation {
	c := &conversation{
		key: key, turn: make(chan struct{}, 1),
		mu: sync.Mutex{}, session: nil, served: nil, isClosed: false,
		listed: nil, instructions: "", isPrompted: false,
		scopeMu: sync.Mutex{}, toolContext: nil, vaultID: "", connection: "",
		lastUsed: now, isGone: false,
	}
	c.turn <- struct{}{}
	return c
}

func (c *conversation) setTools(listed []*tools.ToolInfo) {
	c.listed = make(map[string]*tools.ToolInfo, len(listed))
	var sentences []string
	for _, one := range listed {
		c.listed[one.GetName()] = one
		for _, sentence := range one.GetInstructions() {
			if !slices.Contains(sentences, sentence) {
				sentences = append(sentences, sentence)
			}
		}
	}
	c.instructions = strings.Join(sentences, " ")
}

func (c *conversation) listNames() []string {
	return slices.Sorted(maps.Keys(c.listed))
}

func (c *conversation) setSession(ctx context.Context, session *acp.Session, served ServedTools) error {
	c.mu.Lock()
	isClosed := c.isClosed
	if !isClosed {
		c.session, c.served = session, served
	}
	c.mu.Unlock()
	if isClosed {
		return errors.Join(errClosed, session.Close(ctx), closeServedTools(ctx, served))
	}
	return nil
}

func (c *conversation) setScope(toolContext map[string]string, vaultID, connectionID string) {
	c.scopeMu.Lock()
	defer c.scopeMu.Unlock()
	c.toolContext, c.vaultID, c.connection = maps.Clone(toolContext), vaultID, connectionID
}

func (c *conversation) invokeWith(host Tools) Invoke {
	return func(ctx context.Context, name string, arguments []byte) (*tools.CallResponse, error) {
		c.scopeMu.Lock()
		toolContext, vaultID, connectionID := maps.Clone(c.toolContext), c.vaultID, c.connection
		c.scopeMu.Unlock()
		return host.InvokeTool(ctx, tools.Call{Name: name, Arguments: arguments, Context: toolContext, VaultID: vaultID, ConnectionID: connectionID})
	}
}

func (c *conversation) writePrompt(request *contract.TaskRequest) []string {
	var texts []string
	if !c.isPrompted {
		texts = append(texts, c.instructions)
	}
	for _, note := range request.GetContextNotes() {
		texts = append(texts, writeNote(note))
	}
	return append(texts, request.GetPrompt())
}

func writeNote(note *contract.ContextNote) string {
	heading := "The note " + note.GetPath()
	if title := note.GetTitle(); title != "" {
		heading += " (" + title + ")"
	}
	return heading + ":\n\n" + note.GetText()
}

func (c *conversation) close(ctx context.Context, isForced bool) error {
	c.mu.Lock()
	if c.isClosed {
		c.mu.Unlock()
		return nil
	}
	c.isClosed = true
	session, served := c.session, c.served
	c.mu.Unlock()
	var err error
	switch {
	case session == nil:
	case isForced:
		err = session.Stop()
	default:
		err = session.Close(ctx)
	}
	return errors.Join(err, closeServedTools(ctx, served))
}

func closeServedTools(ctx context.Context, served ServedTools) error {
	if served == nil {
		return nil
	}
	return served.Close(context.WithoutCancel(ctx))
}
