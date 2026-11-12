// Package task carries out acp.task: a prompt turn of the agent program a connection names, in the
// session of its conversation, with the tools of the host served to the program.
package task

import (
	"context"
	"errors"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	contract "github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/acp"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/domain/acp"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/domain/agents"
)

const (
	fieldProgram = "program"
	// defaultOpenLimit is how long a program has to open a session where the settings give no limit.
	defaultOpenLimit = 120 * time.Second
	// listLimit is how long the host has to list the tools when a session opens.
	listLimit = 10 * time.Second
)

// Dependencies are what a Runner needs. Without Tools, a session is served no tool.
type Dependencies struct {
	Launcher   acp.Launcher
	LookPath   func(program string) (string, error)
	Tools      Tools
	ToolServer ToolServer
	Clock      Clock
	Settings   Settings
}

// Runner carries out the tasks of the connections and keeps the sessions of their conversations.
type Runner struct {
	deps Dependencies
	pool *pool
}

// New creates a Runner, and refuses dependencies that are missing.
func New(deps Dependencies) (*Runner, error) {
	if deps.Launcher == nil || deps.ToolServer == nil || deps.Clock == nil || deps.Settings == nil {
		return nil, errors.New("task: a launcher, a tool server, a clock and the settings are required")
	}
	return &Runner{deps: deps, pool: newPool(deps.Clock)}, nil
}

var _ contract.Runner = (*Runner)(nil)

// RunTask carries out one prompt turn in the session of the conversation of the request, opened for the
// turn where the plugin keeps none, and sends the session updates of the turn as frames. A program that
// cannot be found or does not open a session is a failure to try the next connection after; a turn that
// fails is a stop. A canceled turn is canceled in the program too, and the cancellation of ctx is the
// error of a task that could not run.
func (r *Runner) RunTask(
	ctx context.Context, call connection.Call, request *contract.TaskRequest, send func(*contract.TaskFrame) error,
) (*contract.TaskDone, error) {
	agent, err := agents.Find(call.Type, call.Fields[fieldProgram], r.deps.LookPath)
	if errors.Is(err, agents.ErrNoProgram) {
		return nil, connection.NewStopError(connection.ReasonInvalidRequest)
	}
	if err != nil {
		return nil, connection.NewTryNextError(connection.ReasonUnavailable, 0)
	}
	key := ""
	if request.GetConversation() != "" {
		key = request.GetConversation() + "\x00" + call.ConnectionID + "\x00" + agent.GetKey()
	}
	conv, isNew, isKept, err := r.pool.acquire(ctx, key)
	if err != nil {
		return nil, readFailure(ctx, err)
	}
	if isNew {
		if err := r.open(ctx, conv, agent); err != nil {
			r.pool.release(ctx, conv, isKept, true)
			return nil, readFailure(ctx, err)
		}
	}
	done, isBroken, err := r.runTurn(ctx, conv, call, request, send)
	r.pool.release(ctx, conv, isKept, isBroken)
	return done, err
}

func (r *Runner) runTurn(
	ctx context.Context, conv *conversation, call connection.Call, request *contract.TaskRequest, send func(*contract.TaskFrame) error,
) (*contract.TaskDone, bool, error) {
	conv.setScope(request.GetToolContext(), request.GetVaultId())
	session := conv.session
	if err := session.SetModel(ctx, call.Options[llm.OptionModel]); err != nil {
		return nil, true, readFailure(ctx, err)
	}
	if err := session.SetReasoning(ctx, call.Options[llm.OptionReasoning]); err != nil {
		return nil, true, readFailure(ctx, err)
	}
	reason, err := session.Prompt(ctx, conv.writePrompt(request), newTurn(conv.listed, send))
	conv.isPrompted = true
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, err != nil, ctxErr
	}
	if failed, isSendError := errors.AsType[*sendError](err); isSendError {
		return nil, true, failed.err
	}
	if err != nil {
		return nil, true, connection.NewStopError(connection.ReasonUnavailable)
	}
	return &contract.TaskDone{StopReason: writeStopReason(reason)}, false, nil
}

func (r *Runner) open(ctx context.Context, conv *conversation, agent agents.Agent) error {
	if r.deps.Tools != nil {
		listing, cancel := context.WithTimeout(ctx, listLimit)
		listed, err := r.deps.Tools.ListTools(listing)
		cancel()
		if err != nil {
			return connection.NewStopError(connection.ReasonUnavailable)
		}
		conv.setTools(listed.Tools)
	}
	var served ServedTools
	var servers []acp.Server
	if len(conv.listed) > 0 {
		var err error
		served, err = r.deps.ToolServer.Serve(ctx, listTools(conv), conv.invokeWith(r.deps.Tools))
		if err != nil {
			return connection.NewStopError(connection.ReasonUnavailable)
		}
		servers = []acp.Server{{Name: acp.ServerName, URL: served.GetURL(), Token: served.GetToken()}}
	}
	opening, cancel := context.WithTimeout(ctx, r.getOpenLimit())
	defer cancel()
	session, err := acp.Open(opening, r.deps.Launcher, acp.Start{
		Command: agent.Command, Servers: servers, Meta: agent.GetTaskMeta(conv.listNames()),
	})
	if err != nil {
		return errors.Join(connection.NewTryNextError(connection.ReasonUnavailable, 0), closeServedTools(ctx, served))
	}
	return conv.setSession(ctx, session, served)
}

func listTools(conv *conversation) []*tools.ToolInfo {
	listed := make([]*tools.ToolInfo, 0, len(conv.listed))
	for _, name := range conv.listNames() {
		listed = append(listed, conv.listed[name])
	}
	return listed
}

func readFailure(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	if _, isTryNext := connection.IsTryNext(err); isTryNext || connection.IsStop(err) {
		return err
	}
	if errors.Is(err, errClosed) {
		return connection.NewStopError(connection.ReasonUnavailable)
	}
	return connection.NewTryNextError(connection.ReasonUnavailable, 0)
}

func (r *Runner) getOpenLimit() time.Duration {
	if limit := r.deps.Settings.RequestTimeout(); limit > 0 {
		return limit
	}
	return defaultOpenLimit
}

// CloseIdle closes the sessions that have gone unused for 30 minutes, and returns the errors of closing
// sessions since the last call.
func (r *Runner) CloseIdle(ctx context.Context) error {
	return r.pool.closeIdleConversations(ctx)
}

// Close closes every session, and ends a task that still runs.
func (r *Runner) Close(ctx context.Context) error {
	return r.pool.close(ctx)
}
