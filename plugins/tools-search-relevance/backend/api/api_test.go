package api

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/plugintest"
)

// fixtureDir holds the request and response fixtures the frontend contract test reads as well.
const fixtureDir = "../../testdata/api"

func getContracts() map[string]plugintest.MessageContract {
	return map[string]plugintest.MessageContract{
		MethodGetFloor:       {Request: func() any { return &GetFloorRequest{} }, Response: func() any { return &GetFloorResponse{} }},
		MethodCalibrateFloor: {Request: func() any { return &CalibrateFloorRequest{} }, Response: func() any { return &CalibrateFloorResponse{} }},
		MethodSetFloor:       {Request: func() any { return &SetFloorRequest{} }, Response: func() any { return &SetFloorResponse{} }},
		MethodProbeFloor:     {Request: func() any { return &ProbeFloorRequest{} }, Response: func() any { return &ProbeFloorResponse{} }},
	}
}

func TestMessagesMatchFixtures(t *testing.T) {
	plugintest.CheckMessageFixtures(t, fixtureDir, getContracts())
}

func TestEveryMethodHasItsFixturePair(t *testing.T) {
	plugintest.CheckFixturePairs(t, fixtureDir, getContracts())
}
