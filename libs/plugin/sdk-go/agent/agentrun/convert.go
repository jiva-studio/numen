package agentrun

import (
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/reasoning"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/toolcall"
)

// This file converts between the types of this package and the protocol messages. Every conversion
// has a round-trip test, so a protocol field that a conversion does not carry fails the test.

func requestFromProto(msg *v1.AskAgentRequest) *Request {
	if msg == nil {
		return nil
	}
	declarations := make([]toolcall.ToolDeclaration, 0, len(msg.GetToolDeclarations()))
	for _, one := range msg.GetToolDeclarations() {
		declarations = append(declarations, toolDeclarationFromProto(one))
	}
	return &Request{
		Prompt:           msg.GetPrompt(),
		Focus:            msg.GetFocus(),
		Conversation:     msg.GetConversation(),
		Context:          msg.GetContext(),
		Agent:            msg.GetAgent(),
		Model:            msg.GetModel(),
		Instructions:     msg.GetInstructions(),
		Tools:            msg.GetTools(),
		MaxSteps:         msg.GetMaxSteps(),
		ToolDeclarations: declarations,
		Root:             msg.GetRoot(),
		Command:          msg.GetCommand(),
		ToolsURL:         msg.GetToolsUrl(),
		ToolsToken:       msg.GetToolsToken(),
		Reasoning:        msg.GetReasoning(),
	}
}

func requestToProto(req *Request) *v1.AskAgentRequest {
	if req == nil {
		return nil
	}
	declarations := make([]*v1.ToolDeclaration, 0, len(req.ToolDeclarations))
	for _, one := range req.ToolDeclarations {
		declarations = append(declarations, toolDeclarationToProto(one))
	}
	return &v1.AskAgentRequest{
		Prompt:           req.Prompt,
		Focus:            req.Focus,
		Conversation:     req.Conversation,
		Context:          req.Context,
		Agent:            req.Agent,
		Model:            req.Model,
		Instructions:     req.Instructions,
		Tools:            req.Tools,
		MaxSteps:         req.MaxSteps,
		ToolDeclarations: declarations,
		Root:             req.Root,
		Command:          req.Command,
		ToolsUrl:         req.ToolsURL,
		ToolsToken:       req.ToolsToken,
		Reasoning:        req.Reasoning,
	}
}

func toolDeclarationFromProto(msg *v1.ToolDeclaration) toolcall.ToolDeclaration {
	return toolcall.ToolDeclaration{
		Name:    msg.GetName(),
		Title:   msg.GetTitle(),
		Kind:    msg.GetKind(),
		About:   msg.GetAbout(),
		Element: msg.GetElement(),
		Match:   msg.GetMatch(),
		Text:    msg.GetText(),
	}
}

func toolDeclarationToProto(declaration toolcall.ToolDeclaration) *v1.ToolDeclaration {
	return &v1.ToolDeclaration{
		Name:    declaration.Name,
		Title:   declaration.Title,
		Kind:    declaration.Kind,
		About:   declaration.About,
		Element: declaration.Element,
		Match:   declaration.Match,
		Text:    declaration.Text,
	}
}

func responseFromProto(msg *v1.AskAgentResponse) *Response {
	if msg == nil {
		return nil
	}
	switch step := msg.GetStep().(type) {
	case *v1.AskAgentResponse_Text:
		return &Response{Step: &TextStep{Text: step.Text}}
	case *v1.AskAgentResponse_ToolCall:
		return &Response{Step: &ToolCallStep{ToolCall: toolCallFromProto(step.ToolCall)}}
	case *v1.AskAgentResponse_Stopped:
		return &Response{Step: &StoppedStep{Stopped: step.Stopped}}
	case *v1.AskAgentResponse_ToolFinished:
		return &Response{Step: &ToolFinishedStep{}}
	case *v1.AskAgentResponse_ModelStarted:
		return &Response{Step: &ModelStartedStep{}}
	case *v1.AskAgentResponse_EditProgress:
		return &Response{Step: &EditProgressStep{EditProgress: editProgressFromProto(step.EditProgress)}}
	}
	return &Response{Step: nil}
}

func responseToProto(resp *Response) *v1.AskAgentResponse {
	if resp == nil {
		return nil
	}
	switch step := resp.Step.(type) {
	case *TextStep:
		return &v1.AskAgentResponse{Step: &v1.AskAgentResponse_Text{Text: step.Text}}
	case *ToolCallStep:
		return &v1.AskAgentResponse{Step: &v1.AskAgentResponse_ToolCall{ToolCall: toolCallToProto(step.ToolCall)}}
	case *StoppedStep:
		return &v1.AskAgentResponse{Step: &v1.AskAgentResponse_Stopped{Stopped: step.Stopped}}
	case *ToolFinishedStep:
		return &v1.AskAgentResponse{Step: &v1.AskAgentResponse_ToolFinished{ToolFinished: &v1.ToolFinished{}}}
	case *ModelStartedStep:
		return &v1.AskAgentResponse{Step: &v1.AskAgentResponse_ModelStarted{ModelStarted: &v1.ModelStarted{}}}
	case *EditProgressStep:
		return &v1.AskAgentResponse{Step: &v1.AskAgentResponse_EditProgress{EditProgress: editProgressToProto(step.EditProgress)}}
	}
	return &v1.AskAgentResponse{Step: nil}
}

func toolCallFromProto(msg *v1.ToolCall) *toolcall.ToolCall {
	if msg == nil {
		return nil
	}
	var span *toolcall.Range
	if msg.GetRange() != nil {
		span = &toolcall.Range{Start: msg.GetRange().GetStart(), End: msg.GetRange().GetEnd()}
	}
	return &toolcall.ToolCall{
		Tool:   msg.GetTool(),
		About:  msg.GetAbout(),
		Length: msg.GetLength(),
		Path:   msg.GetPath(),
		Range:  span,
		Call:   msg.GetCall(),
		Kind:   msg.GetKind(),
	}
}

func toolCallToProto(call *toolcall.ToolCall) *v1.ToolCall {
	if call == nil {
		return nil
	}
	var span *v1.Range
	if call.Range != nil {
		span = &v1.Range{Start: call.Range.Start, End: call.Range.End}
	}
	return &v1.ToolCall{
		Tool:   call.Tool,
		About:  call.About,
		Length: call.Length,
		Path:   call.Path,
		Range:  span,
		Call:   call.Call,
		Kind:   call.Kind,
	}
}

func editProgressFromProto(msg *v1.EditProgress) *EditProgress {
	if msg == nil {
		return nil
	}
	return &EditProgress{Call: msg.GetCall(), Path: msg.GetPath(), Match: msg.GetMatch(), Text: msg.GetText()}
}

func editProgressToProto(edit *EditProgress) *v1.EditProgress {
	if edit == nil {
		return nil
	}
	return &v1.EditProgress{Call: edit.Call, Path: edit.Path, Match: edit.Match, Text: edit.Text}
}

func stateFromProto(msg *v1.GetAgentStateResponse) *State {
	return &State{Unreachable: msg.GetUnreachable()}
}

func stateToProto(state *State) *v1.GetAgentStateResponse {
	return &v1.GetAgentStateResponse{Unreachable: state.Unreachable}
}

func programCommandsFromProto(msg *v1.GetAgentProgramsRequest) []ProgramCommand {
	commands := make([]ProgramCommand, 0, len(msg.GetCommands()))
	for _, one := range msg.GetCommands() {
		commands = append(commands, ProgramCommand{Program: one.GetProgram(), Command: one.GetCommand()})
	}
	return commands
}

func programCommandsToProto(commands []ProgramCommand) *v1.GetAgentProgramsRequest {
	msg := &v1.GetAgentProgramsRequest{Commands: make([]*v1.AgentProgramCommand, 0, len(commands))}
	for _, one := range commands {
		msg.Commands = append(msg.Commands, &v1.AgentProgramCommand{Program: one.Program, Command: one.Command})
	}
	return msg
}

func programsFromProto(msg *v1.GetAgentProgramsResponse) []*Program {
	programs := make([]*Program, 0, len(msg.GetPrograms()))
	for _, one := range msg.GetPrograms() {
		models := make([]*reasoning.Model, 0, len(one.GetModels()))
		for _, model := range one.GetModels() {
			models = append(models, modelFromProto(model))
		}
		programs = append(programs, &Program{
			Name:        one.GetName(),
			Title:       one.GetTitle(),
			IsInstalled: one.GetIsInstalled(),
			IsDefault:   one.GetIsDefault(),
			Models:      models,
			LimitsTools: one.GetLimitsTools(),
		})
	}
	return programs
}

func programsToProto(programs []*Program) *v1.GetAgentProgramsResponse {
	msg := &v1.GetAgentProgramsResponse{Programs: make([]*v1.AgentProgram, 0, len(programs))}
	for _, one := range programs {
		models := make([]*v1.AgentModel, 0, len(one.Models))
		for _, model := range one.Models {
			models = append(models, modelToProto(model))
		}
		msg.Programs = append(msg.Programs, &v1.AgentProgram{
			Name:        one.Name,
			Title:       one.Title,
			IsInstalled: one.IsInstalled,
			IsDefault:   one.IsDefault,
			Models:      models,
			LimitsTools: one.LimitsTools,
		})
	}
	return msg
}

func modelFromProto(msg *v1.AgentModel) *reasoning.Model {
	levels := make([]reasoning.Level, 0, len(msg.GetReasoningLevels()))
	for _, one := range msg.GetReasoningLevels() {
		levels = append(levels, reasoning.Level{Name: one.GetName(), Title: one.GetTitle()})
	}
	return &reasoning.Model{
		Name:             msg.GetName(),
		Title:            msg.GetTitle(),
		IsDefault:        msg.GetIsDefault(),
		ReasoningLevels:  levels,
		DefaultReasoning: msg.GetDefaultReasoning(),
	}
}

func modelToProto(model *reasoning.Model) *v1.AgentModel {
	levels := make([]*v1.ReasoningLevel, 0, len(model.ReasoningLevels))
	for _, one := range model.ReasoningLevels {
		levels = append(levels, &v1.ReasoningLevel{Name: one.Name, Title: one.Title})
	}
	return &v1.AgentModel{
		Name:             model.Name,
		Title:            model.Title,
		IsDefault:        model.IsDefault,
		ReasoningLevels:  levels,
		DefaultReasoning: model.DefaultReasoning,
	}
}

func toolDefinitionFromProto(msg *v1.AgentToolDefinition) *ToolDefinition {
	if msg == nil {
		return nil
	}
	return &ToolDefinition{
		Name:                 msg.GetName(),
		Description:          msg.GetDescription(),
		ParametersJSONSchema: msg.GetParametersJsonSchema(),
	}
}

func toolDefinitionToProto(tool *ToolDefinition) *v1.AgentToolDefinition {
	if tool == nil {
		return nil
	}
	return &v1.AgentToolDefinition{
		Name:                 tool.Name,
		Description:          tool.Description,
		ParametersJsonSchema: tool.ParametersJSONSchema,
	}
}

func toolSetFromProto(msg *v1.GetAgentToolsResponse) *ToolSet {
	tools := make([]*ToolDefinition, 0, len(msg.GetTools()))
	for _, one := range msg.GetTools() {
		tools = append(tools, toolDefinitionFromProto(one))
	}
	return &ToolSet{Tools: tools, SystemPromptExtension: msg.GetSystemPromptExtension()}
}

func toolSetToProto(set *ToolSet) *v1.GetAgentToolsResponse {
	msg := &v1.GetAgentToolsResponse{
		Tools:                 make([]*v1.AgentToolDefinition, 0, len(set.Tools)),
		SystemPromptExtension: set.SystemPromptExtension,
	}
	for _, one := range set.Tools {
		msg.Tools = append(msg.Tools, toolDefinitionToProto(one))
	}
	return msg
}
