package api

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/plugintest"
)

// fixtureDir holds the request and response fixtures the frontend contract test reads as well.
const fixtureDir = "../../testdata/api"

func TestMessagesMatchFixtures(t *testing.T) {
	contracts := map[string]plugintest.MessageContract{
		MethodCreateNote: {Request: func() any { return &CreateNoteRequest{} }, Response: func() any { return &CreateNoteResponse{} }},
	}

	plugintest.CheckMessageFixtures(t, fixtureDir, contracts)
	plugintest.CheckFixturePairs(t, fixtureDir, contracts)
}
