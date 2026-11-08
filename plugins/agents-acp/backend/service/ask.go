// Package service implements the ACP agent plugin backend.
package service

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/reasoning"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/toolcall"
)

// reasonDone is why a task stops when the agent has answered.
const reasonDone = "done"

var errAgentNotInstalled = errors.New("the ACP agent is not installed")

// question is one task as it is posed to a session.
type question struct {
	model     string
	reasoning string
	system    string
	prompt    string
	tools     toolEndpoint
	words     map[string]toolcall.Declaration
	report    func(*agentrun.Response) error
}

func (s *agentServer) answer(ctx context.Context, agent acpAgent, q question) (text string, err error) {
	session, err := openACPSession(ctx, agent, q.tools)
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, session.close()) }()
	session.words, session.report = q.words, q.report
	if err := session.setModel(q.model); err != nil {
		return "", err
	}
	if err := session.setReasoning(q.reasoning); err != nil {
		return "", err
	}
	return session.ask(q.system, q.prompt)
}

// Ask runs the task on the ACP agent the task names, in a new session that
// connects to the tools of the task at their endpoint.
func (s *agentServer) Ask(ctx context.Context, msg *agentrun.Request, send func(*agentrun.Response) error) error {
	if err := send(&agentrun.Response{Step: &agentrun.ModelStartedStep{}}); err != nil {
		return err
	}
	agent, err := s.findAgent(msg.Agent, msg.Command)
	if err != nil {
		return sendStopped(send, err.Error())
	}
	text, err := s.answer(ctx, agent, question{
		model: msg.Model, reasoning: msg.Reasoning, system: msg.Instructions, prompt: msg.Prompt,
		tools: readToolEndpoint(msg), words: toolcall.IndexDeclarations(msg.ToolDeclarations), report: send,
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
	return sendStopped(send, reasonDone)
}

func sendStopped(send func(*agentrun.Response) error, reason string) error {
	return send(&agentrun.Response{Step: &agentrun.StoppedStep{Stopped: reason}})
}

func (s *agentServer) findAgent(program string, command []string) (acpAgent, error) {
	name, isACP := strings.CutPrefix(program, acpProgramPrefix)
	if !isACP || name == "" {
		return acpAgent{}, errAgentNotInstalled
	}
	if len(command) > 0 {
		return acpAgent{Program: name, Title: name, Command: command}, nil
	}
	for _, agent := range s.getACPAgents() {
		if agent.Program == name {
			return agent, nil
		}
	}
	return acpAgent{}, errAgentNotInstalled
}

// GetPrograms lists, for each agent named by the command lines in the
// settings, its program name and the models it reports, and keeps the command
// lines.
func (s *agentServer) GetPrograms(ctx context.Context, requested []agentrun.ProgramCommand) ([]*agentrun.Program, error) {
	commands := make(map[string][]string, len(requested))
	for _, one := range requested {
		commands[one.Program] = one.Command
	}
	s.mu.Lock()
	s.commands = commands
	s.mu.Unlock()

	agents := s.getACPAgents()
	listings := make([]acpListing, len(agents))
	var group sync.WaitGroup
	for idx, agent := range agents {
		group.Add(1)
		go func(i int, target acpAgent) {
			defer group.Done()
			listings[i] = s.getACPListing(ctx, target)
		}(idx, agent)
	}
	group.Wait()

	var programs []*agentrun.Program
	for idx, agent := range agents {
		listing := listings[idx]
		program := &agentrun.Program{
			Name:        acpProgramPrefix + agent.Program,
			Title:       agent.Title,
			IsInstalled: true,
			LimitsTools: true,
			IsDefault:   false,
			Models:      nil,
		}
		for _, one := range listing.Models {
			title := one.Name
			if title == "" {
				title = one.ID
			}
			model := &reasoning.Model{Name: one.ID, Title: title, IsDefault: one.ID == listing.Current, ReasoningLevels: nil, DefaultReasoning: ""}
			if one.ID == listing.Current {
				model.ReasoningLevels = listing.Levels
				model.DefaultReasoning = listing.Level
			}
			program.Models = append(program.Models, model)
		}
		programs = append(programs, program)
	}
	return programs, nil
}
