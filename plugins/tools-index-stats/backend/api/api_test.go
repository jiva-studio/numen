package api

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/plugintest"
)

// fixtureDir holds the request and response fixtures the frontend contract test reads as well.
const fixtureDir = "../../testdata/api"

func TestMessagesMatchFixtures(t *testing.T) {
	plugintest.CheckMessageFixtures(t, fixtureDir, map[string]plugintest.MessageContract{
		MethodGetStats: {Request: func() any { return &GetStatsRequest{} }, Response: func() any { return &Stats{} }},
	})
}
