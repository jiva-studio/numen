package acp

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func createChunk(text string) json.RawMessage {
	return json.RawMessage(`{"sessionId":"s1","update":{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"` + text + `"}}}`)
}

func TestAnAnswerMayFillTheLimitAndNotPassIt(t *testing.T) {
	session := &Session{id: "s1"}
	session.answer.WriteString(strings.Repeat("a", maxAnswerBytes-4))

	if err := session.readUpdate(createChunk("1234")); err != nil || session.answer.Len() != maxAnswerBytes {
		t.Errorf("readUpdate at the limit = %v (len %d)", err, session.answer.Len())
	}
	if err := session.readUpdate(createChunk("5")); !errors.Is(err, errAnswerTooLarge) {
		t.Errorf("readUpdate past the limit = %v, want %v", err, errAnswerTooLarge)
	}
}

func TestAChunkOfAnotherSessionIsNotPartOfTheAnswer(t *testing.T) {
	session := &Session{id: "s1"}

	if err := session.readUpdate(json.RawMessage(`{"sessionId":"other","update":{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"elsewhere"}}}`)); err != nil {
		t.Fatal(err)
	}
	if session.answer.Len() != 0 {
		t.Errorf("the answer holds %q", session.answer.String())
	}
}
