package api

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/plugintest"
)

// fixtureDir holds the request and response fixtures the frontend contract test reads as well.
const fixtureDir = "../../testdata/api"

func getContracts() map[string]plugintest.MessageContract {
	return map[string]plugintest.MessageContract{
		MethodGetLink:     {CreateRequest: func() any { return &GetLinkRequest{} }, CreateResponse: func() any { return &GetLinkResponse{} }},
		MethodCopyMedia:   {CreateRequest: func() any { return &CopyMediaRequest{} }, CreateResponse: func() any { return &CopyMediaResponse{} }},
		MethodCreateLinks: {CreateRequest: func() any { return &CreateLinksRequest{} }, CreateResponse: func() any { return &CreateLinksResponse{} }},
	}
}

func TestMessagesMatchFixtures(t *testing.T) {
	plugintest.CheckMessageFixtures(t, fixtureDir, getContracts())
}
