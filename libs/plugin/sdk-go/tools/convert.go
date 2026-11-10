package tools

import (
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
)

func readErrorCode(code v1.ToolErrorCode) ErrorCode {
	switch code {
	case v1.ToolErrorCode_TOOL_ERROR_CODE_UNKNOWN_TOOL:
		return ErrorCodeUnknownTool
	case v1.ToolErrorCode_TOOL_ERROR_CODE_BAD_ARGUMENTS:
		return ErrorCodeBadArguments
	default:
		return ErrorCodeUnspecified
	}
}

func writeErrorCode(code ErrorCode) v1.ToolErrorCode {
	switch code {
	case ErrorCodeUnknownTool:
		return v1.ToolErrorCode_TOOL_ERROR_CODE_UNKNOWN_TOOL
	case ErrorCodeBadArguments:
		return v1.ToolErrorCode_TOOL_ERROR_CODE_BAD_ARGUMENTS
	default:
		// gomutants:disable-next-line reason="the returned value is the zero value of its type, so returning zero is the same value"
		return v1.ToolErrorCode_TOOL_ERROR_CODE_UNSPECIFIED
	}
}

func writeEffect(effect Effect) v1.ToolEffect {
	switch effect {
	case ReadOnly:
		return v1.ToolEffect_TOOL_EFFECT_READ_ONLY
	case ReadWrite:
		return v1.ToolEffect_TOOL_EFFECT_READ_WRITE
	}
	// gomutants:disable-next-line reason="the returned value is the zero value of its type, so returning zero is the same value"
	return v1.ToolEffect_TOOL_EFFECT_UNSPECIFIED
}

func writeStep(step Step) v1.ToolStep {
	switch step {
	case Search:
		return v1.ToolStep_TOOL_STEP_SEARCH
	case Remove:
		return v1.ToolStep_TOOL_STEP_REMOVE
	case Move:
		return v1.ToolStep_TOOL_STEP_MOVE
	case ByEffect:
	}
	// gomutants:disable-next-line reason="the returned value is the zero value of its type, so returning zero is the same value"
	return v1.ToolStep_TOOL_STEP_UNSPECIFIED
}

func writeServers(servers []Server) []v1.ToolServer {
	wire := make([]v1.ToolServer, 0, len(servers))
	for _, one := range servers {
		switch one {
		case Full:
			wire = append(wire, v1.ToolServer_TOOL_SERVER_FULL)
		case Reading:
			wire = append(wire, v1.ToolServer_TOOL_SERVER_READING)
		}
	}
	return wire
}

func readEffect(effect v1.ToolEffect) Effect {
	switch effect {
	case v1.ToolEffect_TOOL_EFFECT_READ_ONLY:
		return ReadOnly
	case v1.ToolEffect_TOOL_EFFECT_READ_WRITE:
		return ReadWrite
	default:
		return 0
	}
}

func readStep(step v1.ToolStep) Step {
	switch step {
	case v1.ToolStep_TOOL_STEP_SEARCH:
		return Search
	case v1.ToolStep_TOOL_STEP_REMOVE:
		return Remove
	case v1.ToolStep_TOOL_STEP_MOVE:
		return Move
	default:
		// gomutants:disable-next-line reason="the returned value is the zero value of its type, so returning zero is the same value"
		return ByEffect
	}
}

func readServers(servers []v1.ToolServer) []Server {
	var out []Server
	for _, s := range servers {
		switch s {
		case v1.ToolServer_TOOL_SERVER_UNSPECIFIED:
			continue
		case v1.ToolServer_TOOL_SERVER_FULL:
			out = append(out, Full)
		case v1.ToolServer_TOOL_SERVER_READING:
			out = append(out, Reading)
		}
	}
	return out
}

func readToolInfo(t *v1.PluginTool) *ToolInfo {
	if t == nil {
		return nil
	}
	return &ToolInfo{
		Name:         t.GetName(),
		Title:        t.GetTitle(),
		Description:  t.GetDescription(),
		InputSchema:  t.GetInputSchema(),
		OutputSchema: t.GetOutputSchema(),
		Effect:       readEffect(t.GetEffect()),
		Step:         readStep(t.GetStep()),
		Servers:      readServers(t.GetServers()),
		Instructions: t.GetInstructions(),
		Requires:     t.GetRequires(),
		Hints:        readHints(t.GetHints()),
	}
}

func writeToolInfo(t *ToolInfo) *v1.PluginTool {
	if t == nil {
		return nil
	}
	return &v1.PluginTool{
		Name:         t.Name,
		Title:        t.Title,
		Description:  t.Description,
		InputSchema:  t.InputSchema,
		OutputSchema: t.OutputSchema,
		Effect:       writeEffect(t.Effect),
		Step:         writeStep(t.Step),
		Servers:      writeServers(t.Servers),
		Instructions: t.Instructions,
		Requires:     t.Requires,
		Hints:        writeHints(t.Hints),
	}
}

func readCallResponse(res *v1.CallToolResponse) *CallResponse {
	if res == nil {
		return nil
	}
	out := &CallResponse{Error: readErrorCode(res.GetError())}
	if r := res.GetResult(); r != nil {
		out.Result = &CallResult{
			Text:       r.GetText(),
			Structured: r.GetStructured(),
			IsError:    r.GetIsError(),
		}
	}
	return out
}

func writeCallResponse(res *CallResponse) *v1.CallToolResponse {
	if res == nil {
		return &v1.CallToolResponse{}
	}
	out := &v1.CallToolResponse{Error: writeErrorCode(res.Error)}
	if res.Result != nil {
		out.Result = &v1.ToolResult{
			Text:       res.Result.Text,
			Structured: res.Result.Structured,
			IsError:    res.Result.IsError,
		}
	}
	return out
}

func readHints(hints *v1.ToolHints) Hints {
	return Hints{
		ReadOnly:    hints.GetReadOnly(),
		Destructive: hints.GetDestructive(),
		Idempotent:  hints.GetIdempotent(),
		OpenWorld:   hints.GetOpenWorld(),
	}
}

func writeHints(hints Hints) *v1.ToolHints {
	return &v1.ToolHints{
		ReadOnly:    hints.ReadOnly,
		Destructive: hints.Destructive,
		Idempotent:  hints.Idempotent,
		OpenWorld:   hints.OpenWorld,
	}
}
