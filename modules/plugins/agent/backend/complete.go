package main

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/completion"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

var errNoKey = errors.New("no API key is configured")

// Complete answers one question from the chat API, with the model the host names, else the
// configured one, and with no tools, no vault and no conversation.
// The key is the one the host configured.
func (s *agentServer) Complete(
	ctx context.Context,
	req *connect.Request[numenv1.CompleteRequest],
) (*connect.Response[numenv1.CompleteResponse], error) {
	s.mu.RLock()
	key, base, model, client := s.apiKey, s.baseURL, s.modelName, s.httpClient
	s.mu.RUnlock()

	return completion.Answer(ctx, req.Msg.GetQuestion(), func(ctx context.Context, request completion.Request) (string, error) {
		if key == "" {
			return "", errNoKey
		}
		var messages []chatMessage
		if request.System != "" {
			messages = append(messages, chatMessage{Role: "system", Content: &request.System})
		}
		messages = append(messages, chatMessage{Role: "user", Content: &request.Prompt})

		reply, err := callChat(ctx, getHTTPClient(client), getBaseURL(base), key, getModel(req.Msg.GetModel(), model), messages, nil)
		if err != nil {
			return "", err
		}
		if reply.Content == nil {
			return "", nil
		}
		return *reply.Content, nil
	})
}

// ListCompletionModels lists the model Complete answers with when it is named none.
func (s *agentServer) ListCompletionModels(
	context.Context,
	*connect.Request[numenv1.ListCompletionModelsRequest],
) (*connect.Response[numenv1.ListCompletionModelsResponse], error) {
	s.mu.RLock()
	model := getModel("", s.modelName)
	s.mu.RUnlock()
	return connect.NewResponse(&numenv1.ListCompletionModelsResponse{
		Models: []*numenv1.CompletionModel{{Name: model, Title: model, IsDefault: true}},
	}), nil
}
