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
		api.MethodGetTags: {CreateRequest: func() any { return &api.GetTagsRequest{} }, CreateResponse: func() any { return &api.TagList{} }},
		api.MethodSetTags: {CreateRequest: func() any { return &api.SetTagsRequest{} }, CreateResponse: func() any { return &api.SetTagsResult{} }},
	})
}

func TestTheTagsMethodNames(t *testing.T) {
	if api.MethodGetTagMap != "GetTagMap" || api.MethodGetTags != "GetTags" || api.MethodSetTags != "SetTags" {
		t.Errorf("methods %q, %q and %q, want GetTagMap, GetTags and SetTags", api.MethodGetTagMap, api.MethodGetTags, api.MethodSetTags)
	}
}
