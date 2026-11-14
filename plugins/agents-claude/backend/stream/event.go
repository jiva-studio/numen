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
	raw, err := unmarshalObject(data, (*plain)(e))
	return errors.Join(
		err,
		readField(raw, "session_id", &e.Session),
		readField(raw, "is_error", &e.IsError),
		readField(raw, "mcp_servers", &e.Servers),
		readField(raw, "mcp_server_errors", &e.ServerErrors),
	)
}

func unmarshalObject(data []byte, target any) (map[string]json.RawMessage, error) {
	var raw map[string]json.RawMessage
	return raw, errors.Join(json.Unmarshal(data, target), json.Unmarshal(data, &raw))
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
		if err := errors.Join(
			readField(block, "type", &kind),
			readField(block, "tool_use_id", &id),
			readField(block, "is_error", &isError),
		); err != nil || kind != "tool_result" {
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
	Type     string `json:"type"`
	Text     string `json:"text"`
	Thinking string `json:"thinking"`
	Partial  string `json:"-"`
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
	raw, err := unmarshalObject(data, (*plain)(s))
	return errors.Join(err, readField(raw, "content_block", &s.Block))
}

// UnmarshalJSON reads the partial JSON of a call, which the program writes under a key that is not
// camelCase.
func (d *streamDelta) UnmarshalJSON(data []byte) error {
	type plain streamDelta
	raw, err := unmarshalObject(data, (*plain)(d))
	return errors.Join(err, readField(raw, "partial_json", &d.Partial))
}
