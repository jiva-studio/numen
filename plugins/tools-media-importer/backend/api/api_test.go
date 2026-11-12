package api

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/plugintest"
)

// fixtureDir holds the request and response fixtures the frontend contract test reads as well.
const fixtureDir = "../../testdata/api"

func getContracts() map[string]plugintest.MessageContract {
	return map[string]plugintest.MessageContract{
		MethodGetLink:     {Request: func() any { return &GetLinkRequest{} }, Response: func() any { return &GetLinkResponse{} }},
		MethodCopyMedia:   {Request: func() any { return &CopyMediaRequest{} }, Response: func() any { return &CopyMediaResponse{} }},
		MethodCreateLinks: {Request: func() any { return &CreateLinksRequest{} }, Response: func() any { return &CreateLinksResponse{} }},
	}
}

func TestMessagesMatchFixtures(t *testing.T) {
	plugintest.CheckMessageFixtures(t, fixtureDir, getContracts())
}
