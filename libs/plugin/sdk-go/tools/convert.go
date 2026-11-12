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
		Instructions: t.GetInstructions(),
		Requires:     t.GetRequires(),
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
		Instructions: t.Instructions,
		Requires:     t.Requires,
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
