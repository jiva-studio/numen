package service

import "testing"

func TestMinChunkLengthIsReadFromTheCamelCaseKey(t *testing.T) {
	if got := readMinChunkLength(map[string]string{"minChunkLength": "120"}); got != 120 {
		t.Errorf("readMinChunkLength = %d, want 120", got)
	}
}

func TestMinChunkLengthIsNotReadFromTheSnakeCaseKey(t *testing.T) {
	if got := readMinChunkLength(map[string]string{"min_chunk_length": "120"}); got != defaultMinChunkLength {
		t.Errorf("readMinChunkLength = %d, want the default %d", got, defaultMinChunkLength)
	}
}
