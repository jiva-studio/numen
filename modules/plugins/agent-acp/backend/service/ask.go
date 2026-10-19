package service

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// reasonDone is why a task stops when the agent has answered.
const reasonDone = "done"

// errNoTools is why a task naming tools is refused: the client opens sessions with no MCP server.
var errNoTools = errors.New("this agent answers single questions and runs no tools")

// answer asks the agent one question in a session of its own, with the model the agent reports by
// the id, and returns the text of its answer.
func (s *agentServer) answer(ctx context.Context, agent acpAgent, modelID, system, prompt string) (string, error) {
	session, err := openACPSession(ctx, agent)
	if err != nil {
		return "", err
	}
	defer session.close()
	if err := session.setModel(modelID); err != nil {
		return "", err
	}
	return session.ask(system, prompt)
}

// AskAgent answers a task that names no tool with the ACP agent the task names, in a session of its
// own with no MCP servers, no file system or terminal and every permission refused. A task naming a
// tool is stopped.
func (s *agentServer) AskAgent(
	ctx context.Context,
	req *connect.Request[numenv1.AskAgentRequest],
	stream *connect.ServerStream[numenv1.AskAgentResponse],
) error {
	msg := req.Msg
	if err := stream.Send(&numenv1.AskAgentResponse{
		Step: &numenv1.AskAgentResponse_ModelStarted{ModelStarted: &numenv1.ModelStarted{}},
	}); err != nil {
		return err
	}
	if agentrun.HasTools(msg) {
		return sendStopped(stream, errNoTools.Error())
	}
	agent, err := s.findAgent(msg.GetAgent(), msg.GetCommand())
	if err != nil {
		return sendStopped(stream, err.Error())
	}
	text, err := s.answer(ctx, agent, msg.GetModel(), msg.GetInstructions(), msg.GetPrompt())
	if ctx.Err() != nil {
		return nil
	}
	if err != nil {
		return sendStopped(stream, err.Error())
	}
	if err := stream.Send(&numenv1.AskAgentResponse{Step: &numenv1.AskAgentResponse_Said{Said: text}}); err != nil {
		return err
	}
	return sendStopped(stream, reasonDone)
}

func sendStopped(stream *connect.ServerStream[numenv1.AskAgentResponse], reason string) error {
	return stream.Send(&numenv1.AskAgentResponse{Step: &numenv1.AskAgentResponse_Stopped{Stopped: reason}})
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

// GetAgentPrograms lists, for each agent the command lines of the settings name, the program it is
// asked for by and the models it reports, and keeps the command lines.
func (s *agentServer) GetAgentPrograms(
	ctx context.Context,
	req *connect.Request[numenv1.GetAgentProgramsRequest],
) (*connect.Response[numenv1.GetAgentProgramsResponse], error) {
	commands := make(map[string][]string, len(req.Msg.GetCommands()))
	for _, one := range req.Msg.GetCommands() {
		commands[one.GetProgram()] = one.GetCommand()
	}
	s.mu.Lock()
	s.commands = commands
	s.mu.Unlock()

	res := &numenv1.GetAgentProgramsResponse{}
	for _, agent := range s.getACPAgents() {
		listing := s.acpModels.get(ctx, agent)
		program := &numenv1.AgentProgram{
			Name:        acpProgramPrefix + agent.Program,
			Title:       agent.Title,
			IsInstalled: true,
			LimitsTools: true,
		}
		for _, one := range listing.models {
			title := one.Name
			if title == "" {
				title = one.ID
			}
			program.Models = append(program.Models, &numenv1.AgentModel{
				Name: one.ID, Title: title, IsDefault: one.ID == listing.current,
			})
		}
		res.Programs = append(res.Programs, program)
	}
	return connect.NewResponse(res), nil
}
