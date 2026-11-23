package api

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/plugintest"
)

// fixtureDir holds the request and response fixtures the frontend contract test reads as well.
const fixtureDir = "../../testdata/api"

func TestMessagesMatchFixtures(t *testing.T) {
	contracts := map[string]plugintest.MessageContract{
		MethodListFields: {CreateRequest: func() any { return &ListFieldsRequest{} }, CreateResponse: func() any { return &ListFieldsResponse{} }},
		MethodCreateNote: {CreateRequest: func() any { return &CreateNoteRequest{} }, CreateResponse: func() any { return &CreateNoteResponse{} }},
	}

	plugintest.CheckMessageFixtures(t, fixtureDir, contracts)
	plugintest.CheckFixturePairs(t, fixtureDir, contracts)
}
