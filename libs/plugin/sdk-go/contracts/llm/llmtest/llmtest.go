// Package llmtest carries out the operations of the llm contract for the tests of a plugin, with the
// messages of the contract in place of their encoding: it asks the operations a plugin provides, and it
// is a Caller that answers from the operations a test provides.
package llmtest

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
)

// Complete asks the llm.complete operation of provider to complete the messages of request.
func Complete(ctx context.Context, provider connection.Provider, call connection.Call, request *llm.CompleteRequest) (*llm.CompleteResult, error) {
	result := &llm.CompleteResult{}
	if err := serveInto(ctx, provider, llm.OperationComplete, call, request, result); err != nil {
		return nil, err
	}
	return result, nil
}

// Chat asks the llm.chat operation of provider for the next turn of request. It hands each frame to send,
// in order, and returns the finish.
func Chat(
	ctx context.Context, provider connection.Provider, call connection.Call, request *llm.ChatRequest, send func(*llm.ChatFrame) error,
) (*llm.ChatFinish, error) {
	done, err := serve(ctx, provider, llm.OperationChat, call, request, func(data []byte) error {
		frame := &llm.ChatFrame{}
		if err := proto.Unmarshal(data, frame); err != nil {
			return err
		}
		return send(frame)
	})
	if err != nil {
		return nil, err
	}
	finish := &llm.ChatFinish{}
	if err := proto.Unmarshal(done, finish); err != nil {
		return nil, err
	}
	return finish, nil
}

// Embed asks the llm.embed operation of provider for the vectors of the texts of request.
func Embed(ctx context.Context, provider connection.Provider, call connection.Call, request *llm.EmbedRequest) (*llm.EmbedResult, error) {
	result := &llm.EmbedResult{}
	if err := serveInto(ctx, provider, llm.OperationEmbed, call, request, result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetProfile asks the llm.profile operation of provider for the profile of the model of call.
func GetProfile(ctx context.Context, provider connection.Provider, call connection.Call) (*llm.Profile, error) {
	profile := &llm.Profile{}
	if err := serveInto(ctx, provider, llm.OperationProfile, call, &llm.ProfileRequest{}, profile); err != nil {
		return nil, err
	}
	return profile, nil
}

func serveInto(
	ctx context.Context, provider connection.Provider, operation string, call connection.Call, request, result proto.Message,
) error {
	done, err := serve(ctx, provider, operation, call, request, nil)
	if err != nil {
		return err
	}
	return proto.Unmarshal(done, result)
}

func serve(
	ctx context.Context, provider connection.Provider, operation string, call connection.Call, request proto.Message,
	send func([]byte) error,
) ([]byte, error) {
	served, isServed := provider[operation]
	if !isServed {
		return nil, fmt.Errorf("llmtest: the provider does not serve %s", operation)
	}
	payload, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}
	return served(ctx, call, payload, send)
}
