package stream

import (
	"encoding/json"
	"errors"
)

type event struct {
	Type    string          `json:"type"`
	Subtype string          `json:"subtype"`
	Session string          `json:"-"`
	Message json.RawMessage `json:"message"`
	Event   streamEvent     `json:"event"`
	Result  string          `json:"result"`
	IsError bool            `json:"-"`

	Servers      *[]serverStatus `json:"-"`
	ServerErrors []serverError   `json:"-"`
}

type serverStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type serverError struct {
	Name    string `json:"name"`
	Message string `json:"message"`
}

// UnmarshalJSON reads the keys of the program that are not camelCase beside the ones that are.
func (e *event) UnmarshalJSON(data []byte) error {
	type plain event
	var read plain
	if err := json.Unmarshal(data, &read); err != nil {
		return err
	}
	raw, err := readFields(data)
	if err != nil {
		return err
	}
	err = errors.Join(
		readField(raw, "session_id", &read.Session),
		readField(raw, "is_error", &read.IsError),
		readField(raw, "mcp_servers", &read.Servers),
		readField(raw, "mcp_server_errors", &read.ServerErrors),
	)
	if err != nil {
		return err
	}
	*e = event(read)
	return nil
}

func readFields(data []byte) (map[string]json.RawMessage, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func readField(raw map[string]json.RawMessage, key string, target any) error {
	value, ok := raw[key]
	if !ok {
		return nil
	}
	return json.Unmarshal(value, target)
}

// toolAnswer is the end of a call the program reports in a user message.
type toolAnswer struct {
	ID      string
	IsError bool
}

func (e event) getToolAnswers() []toolAnswer {
	var msg struct {
		Content []map[string]json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(e.Message, &msg); err != nil {
		return nil
	}
	var answers []toolAnswer
	for _, block := range msg.Content {
		var kind, id string
		var isError bool
		if json.Unmarshal(block["type"], &kind) != nil || kind != "tool_result" {
			continue
		}
		if readField(block, "tool_use_id", &id) != nil || readField(block, "is_error", &isError) != nil {
			continue
		}
		answers = append(answers, toolAnswer{ID: id, IsError: isError})
	}
	return answers
}

type streamEvent struct {
	Type  string       `json:"type"`
	Delta streamDelta  `json:"delta"`
	Block contentBlock `json:"-"`
}

type streamDelta struct {
	Type    string `json:"type"`
	Text    string `json:"text"`
	Partial string `json:"-"`
}

type contentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text"`
	Name  string          `json:"name"`
	ID    string          `json:"id"`
	Input json.RawMessage `json:"input"`
}

// UnmarshalJSON reads the content block, which the program writes under a key that is not camelCase.
func (s *streamEvent) UnmarshalJSON(data []byte) error {
	type plain streamEvent
	var read plain
	if err := json.Unmarshal(data, &read); err != nil {
		return err
	}
	raw, err := readFields(data)
	if err != nil {
		return err
	}
	if err := readField(raw, "content_block", &read.Block); err != nil {
		return err
	}
	*s = streamEvent(read)
	return nil
}

// UnmarshalJSON reads the partial JSON of a call, which the program writes under a key that is not
// camelCase.
func (d *streamDelta) UnmarshalJSON(data []byte) error {
	type plain streamDelta
	var read plain
	if err := json.Unmarshal(data, &read); err != nil {
		return err
	}
	raw, err := readFields(data)
	if err != nil {
		return err
	}
	if err := readField(raw, "partial_json", &read.Partial); err != nil {
		return err
	}
	*d = streamDelta(read)
	return nil
}
