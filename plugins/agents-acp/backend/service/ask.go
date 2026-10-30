// Package service implements the ACP agent plugin backend.
package service

import (
	"context"
	"errors"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"strings"
	"sync"

	"connectrpc.com/connect"

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
	// words are the declarations of the run's tools, which a call is shown by.
	words map[string]toolcall.Declaration
	// report sends a step of the run as it happens.
	report func(*agentrun.AskAgentResponse) error
}

// answer poses one question to the agent in a new session, using the model the
// agent reports for the id and the tools of the question. It reports each tool
// call as it happens and returns the text of the answer.
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

// AskAgent runs the task on the ACP agent the task names, in a new session that
// connects to the tools of the task at their endpoint.
func (s *agentServer) AskAgent(
	ctx context.Context,
	req *connect.Request[agentrun.AskAgentRequest],
	stream *connect.ServerStream[agentrun.AskAgentResponse],
) error {
	msg := req.Msg
	if err := stream.Send(&agentrun.AskAgentResponse{
		Step: &agentrun.AskAgentResponse_ModelStarted{ModelStarted: &agentrun.ModelStarted{}},
	}); err != nil {
		return err
	}
	agent, err := s.findAgent(msg.GetAgent(), msg.GetCommand())
	if err != nil {
		return sendStopped(stream, err.Error())
	}
	text, err := s.answer(ctx, agent, question{
		model: msg.GetModel(), reasoning: msg.GetReasoning(), system: msg.GetInstructions(), prompt: msg.GetPrompt(),
		tools: readToolEndpoint(msg), words: toolcall.IndexDeclarations(msg.GetToolDeclarations()), report: stream.Send,
	})
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		return sendStopped(stream, err.Error())
	}
	if err := stream.Send(&agentrun.AskAgentResponse{Step: &agentrun.AskAgentResponse_Text{Text: text}}); err != nil {
		return err
	}
	return sendStopped(stream, reasonDone)
}

func sendStopped(stream *connect.ServerStream[agentrun.AskAgentResponse], reason string) error {
	return stream.Send(&agentrun.AskAgentResponse{Step: &agentrun.AskAgentResponse_Stopped{Stopped: reason}})
}

// findAgent returns the agent the program names, started by the command line given, else by the one
// the settings gave at the last program listing.
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

// GetAgentPrograms lists, for each agent named by the command lines in the
// settings, its program name and the models it reports, and keeps the command
// lines.
func (s *agentServer) GetAgentPrograms(
	ctx context.Context,
	req *connect.Request[agentrun.GetAgentProgramsRequest],
) (*connect.Response[agentrun.GetAgentProgramsResponse], error) {
	commands := make(map[string][]string, len(req.Msg.GetCommands()))
	for _, one := range req.Msg.GetCommands() {
		commands[one.GetProgram()] = one.GetCommand()
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

	res := &agentrun.GetAgentProgramsResponse{Programs: nil}
	for idx, agent := range agents {
		listing := listings[idx]
		program := &agentrun.AgentProgram{
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
			model := &agentrun.AgentModel{Name: one.ID, Title: title, IsDefault: one.ID == listing.Current, ReasoningLevels: nil, DefaultReasoning: ""}
			if one.ID == listing.Current {
				model.ReasoningLevels = reasoning.ToProto(listing.Levels)
				model.DefaultReasoning = listing.Level
			}
			program.Models = append(program.Models, model)
		}
		res.Programs = append(res.Programs, program)
	}
	return connect.NewResponse(res), nil
}
