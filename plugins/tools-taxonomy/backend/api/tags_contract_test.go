package api_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/plugintest"

	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/api"
)

// fixtureDir holds the request and response fixtures the frontend tests read as well.
const fixtureDir = "../../testdata/api"

func TestTheTagsMessagesMatchTheirFixtures(t *testing.T) {
	plugintest.CheckMessageFixtures(t, fixtureDir, map[string]plugintest.MessageContract{
		api.MethodGetTags: {Request: func() any { return &api.GetTagsRequest{} }, Response: func() any { return &api.TagList{} }},
		api.MethodSetTags: {Request: func() any { return &api.SetTagsRequest{} }, Response: func() any { return &api.SetTagsResult{} }},
	})
}

func TestTheTagsMethodNames(t *testing.T) {
	if api.MethodGetTags != "GetTags" || api.MethodSetTags != "SetTags" {
		t.Errorf("methods %q and %q, want GetTags and SetTags", api.MethodGetTags, api.MethodSetTags)
	}
}
