package api

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/plugintest"
)

// fixtureDir holds the request and response fixtures the frontend contract test reads as well.
const fixtureDir = "../../testdata/api"

func getContracts() map[string]plugintest.MessageContract {
	return map[string]plugintest.MessageContract{
		MethodCheck:     {Request: func() any { return &CheckRequest{} }, Response: func() any { return &Status{} }},
		MethodGetStatus: {Request: func() any { return &GetStatusRequest{} }, Response: func() any { return &Status{} }},
	}
}

func TestMessagesMatchFixtures(t *testing.T) {
	plugintest.CheckMessageFixtures(t, fixtureDir, getContracts())
}
