// Package ask runs a task on the ACP agent the task names.
package ask

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/toolcall"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/domain/acp"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/domain/agents"
)

// ReasonDone is why a task stops when the agent has answered.
const ReasonDone = "done"

// Finder returns the agent a program names.
type Finder interface {
	Find(program string, command []string) (agents.Agent, error)
}

// Asker runs tasks, each in a new session.
type Asker struct {
	finder   Finder
	launcher acp.Launcher
}

// New returns an Asker that finds agents with finder and starts them with launcher.
func New(finder Finder, launcher acp.Launcher) (*Asker, error) {
	if finder == nil || launcher == nil {
		return nil, errors.New("ask: a finder and a launcher are required")
	}
	return &Asker{finder: finder, launcher: launcher}, nil
}

// question is one task as it is posed to a session.
type question struct {
	model     string
	reasoning string
	system    string
	prompt    string
	tools     acp.Tools
	words     map[string]toolcall.Declaration
	report    func(*agentrun.Response) error
}

// Execute runs the task on the ACP agent the task names, in a new session that connects to the
// tools of the task at their endpoint, and reports each step to send. A task the agent cannot
// answer ends with a stopped step that holds the reason.
func (a *Asker) Execute(ctx context.Context, task *agentrun.Request, send func(*agentrun.Response) error) error {
	if err := send(&agentrun.Response{Step: &agentrun.ModelStartedStep{}}); err != nil {
		return err
	}
	agent, err := a.finder.Find(task.Agent, task.Command)
	if err != nil {
		return sendStopped(send, err.Error())
	}
	text, err := a.answer(ctx, agent, question{
		model: task.Model, reasoning: task.Reasoning, system: task.Instructions, prompt: task.Prompt,
		tools: getTools(task), words: toolcall.IndexDeclarations(task.ToolDeclarations), report: send,
	})
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		return sendStopped(send, err.Error())
	}
	if err := send(&agentrun.Response{Step: &agentrun.TextStep{Text: text}}); err != nil {
		return err
	}
	return sendStopped(send, ReasonDone)
}

func (a *Asker) answer(ctx context.Context, agent agents.Agent, q question) (text string, err error) {
	session, err := acp.Open(ctx, a.launcher, agent, q.tools)
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, session.Close()) }()
	session.SetReporter(q.words, q.report)
	if err := session.SetModel(q.model); err != nil {
		return "", err
	}
	if err := session.SetReasoning(q.reasoning); err != nil {
		return "", err
	}
	return session.Ask(q.system, q.prompt)
}

func sendStopped(send func(*agentrun.Response) error, reason string) error {
	return send(&agentrun.Response{Step: &agentrun.StoppedStep{Stopped: reason}})
}

func getTools(task *agentrun.Request) acp.Tools {
	if !agentrun.HasTools(task) || task.ToolsURL == "" {
		return acp.Tools{URL: "", Token: "", Names: nil}
	}
	return acp.Tools{URL: task.ToolsURL, Token: task.ToolsToken, Names: task.Tools}
}
