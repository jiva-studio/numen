package api

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/plugintest"
)

// fixtureDir holds the request and response fixtures the frontend contract test reads as well.
const fixtureDir = "../../testdata/api"

func getContracts() map[string]plugintest.MessageContract {
	return map[string]plugintest.MessageContract{
		MethodGetFloor:       {CreateRequest: func() any { return &GetFloorRequest{} }, CreateResponse: func() any { return &GetFloorResponse{} }},
		MethodCalibrateFloor: {CreateRequest: func() any { return &CalibrateFloorRequest{} }, CreateResponse: func() any { return &CalibrateFloorResponse{} }},
		MethodSetFloor:       {CreateRequest: func() any { return &SetFloorRequest{} }, CreateResponse: func() any { return &SetFloorResponse{} }},
		MethodProbeFloor:     {CreateRequest: func() any { return &ProbeFloorRequest{} }, CreateResponse: func() any { return &ProbeFloorResponse{} }},
	}
}

func TestMessagesMatchFixtures(t *testing.T) {
	plugintest.CheckMessageFixtures(t, fixtureDir, getContracts())
}

func TestEveryMethodHasItsFixturePair(t *testing.T) {
	plugintest.CheckFixturePairs(t, fixtureDir, getContracts())
}
