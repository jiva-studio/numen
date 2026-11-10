package stream

import "encoding/json"

type event struct {
	Type    string          `json:"type"`
	Subtype string          `json:"subtype"`
	Session string          `json:"session_id"`
	Message json.RawMessage `json:"message"`
	Event   streamEvent     `json:"event"`
	Result  string          `json:"result"`
	IsError bool            `json:"is_error"`
	Status  string          `json:"status"`

	Servers *[]struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	} `json:"mcp_servers"`
	ServerErrors []struct {
		Name    string `json:"name"`
		Message string `json:"message"`
	} `json:"mcp_server_errors"`
}

func (e event) hasToolAnswer() bool {
	var msg struct {
		Content []struct {
			Type string `json:"type"`
		} `json:"content"`
	}
	if err := json.Unmarshal(e.Message, &msg); err != nil {
		return false
	}
	for _, c := range msg.Content {
		if c.Type == "tool_result" {
			return true
		}
	}
	return false
}

type streamEvent struct {
	Type  string `json:"type"`
	Delta struct {
		Type    string `json:"type"`
		Text    string `json:"text"`
		Partial string `json:"partial_json"`
	} `json:"delta"`
	Block struct {
		Type  string          `json:"type"`
		Text  string          `json:"text"`
		Name  string          `json:"name"`
		ID    string          `json:"id"`
		Input json.RawMessage `json:"input"`
	} `json:"content_block"`
}
