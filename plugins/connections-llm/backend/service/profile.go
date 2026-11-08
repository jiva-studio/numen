package service

import (
	"context"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/connection"
)

type embeddingProfile struct {
	ids        []string
	dimensions int
	maxTokens  int
	pooling    string
}

var embeddingProfiles = []embeddingProfile{
	{[]string{"openai/text-embedding-3-small", "text-embedding-3-small"}, 1536, 8191, "cls"},
	{[]string{"openai/text-embedding-3-large", "text-embedding-3-large"}, 3072, 8191, "cls"},
	{[]string{"openai/text-embedding-ada-002", "text-embedding-ada-002"}, 1536, 8191, "cls"},
	{[]string{"baai/bge-m3"}, 1024, 8192, "cls"},
	{[]string{"qwen/qwen3-embedding-8b"}, 4096, 32768, "cls"},
}

// GetModelProfile returns what the plugin knows of an embedding model from its table, with no request.
// A model that is not in the table has an unknown profile.
func (s *Service) GetModelProfile(_ context.Context, call connection.Call) (connection.ModelProfile, error) {
	if !isKnownType(call.Type) {
		return connection.ModelProfile{}, connection.NewStopError(connection.ReasonBadRequest)
	}
	base, err := parseAddress(call.Fields[fieldBaseURL], false)
	if err != nil {
		return connection.ModelProfile{}, connection.NewStopError(connection.ReasonBadRequest)
	}
	profile := connection.ModelProfile{Source: pluginID + "/" + call.Type + " " + base.Scheme + "://" + base.Host + strings.TrimSuffix(base.Path, "/")}
	name := strings.ToLower(strings.TrimSpace(call.Model))
	for _, one := range embeddingProfiles {
		if slices.Contains(one.ids, name) {
			profile.IsKnown, profile.Dimensions, profile.MaxTokens, profile.Pooling = true, one.dimensions, one.maxTokens, one.pooling
			break
		}
	}
	return profile, nil
}
