package service

import "testing"

func TestAnswerLengthIsReadFromTheCamelCaseKey(t *testing.T) {
	s := &Server{}
	s.Configure(map[string]string{"maxTokens": "1024"})

	if s.maxTokens == nil || *s.maxTokens != 1024 {
		t.Errorf("max tokens = %v, want 1024", s.maxTokens)
	}
}

func TestAnswerLengthIsNotReadFromTheSnakeCaseKey(t *testing.T) {
	s := &Server{}
	s.Configure(map[string]string{"max_tokens": "1024"})

	if s.maxTokens != nil {
		t.Errorf("max tokens = %d, want none", *s.maxTokens)
	}
}
