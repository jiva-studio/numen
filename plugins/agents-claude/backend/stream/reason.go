package stream

import (
	"fmt"
	"strings"
)

// ReasonMaxTurns is the reason of a run that reached its limit of turns.
const ReasonMaxTurns = "the agent reached the limit of steps without finishing"

func getUnreachableReason(response event) string {
	for _, skipped := range response.ServerErrors {
		if skipped.Name == serverName {
			return fmt.Sprintf("the agent was not given this vault: %s", skipped.Message)
		}
	}
	if response.Servers == nil {
		return ""
	}
	for _, server := range *response.Servers {
		if server.Name != serverName {
			continue
		}
		if server.Status == "failed" {
			return "the agent could not reach this vault"
		}
		return ""
	}
	return "the agent was started without this vault"
}

func getResultError(response event) string {
	if text := strings.TrimSpace(response.Result); text != "" && text != response.Subtype {
		return text
	}
	if response.Subtype != "" {
		return formatSubtype(response.Subtype)
	}
	return "the agent stopped without finishing"
}

func formatSubtype(subtype string) string {
	switch subtype {
	case "error_max_turns", "max_turns":
		return ReasonMaxTurns
	case "error_context_length_exceeded", "error_context_length", "context_length_exceeded":
		return "the conversation is too long for the model"
	case "error_rate_limit", "rate_limit":
		return "the rate limit was reached"
	case "error_authentication", "error_auth", "authentication_error":
		return "authentication with the model failed"
	case "error_overloaded", "overloaded":
		return "the model is overloaded"
	case "error_interrupted", "error_canceled", "interrupted":
		return "the agent was interrupted"
	default:
		if rest, ok := strings.CutPrefix(subtype, "error_"); ok {
			return strings.ReplaceAll(rest, "_", " ")
		}
		return subtype
	}
}
