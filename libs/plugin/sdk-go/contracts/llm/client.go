package llm

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/protobuf/proto"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
)

// Complete asks the rows of the plugin, in order, to complete the messages. It returns the errors of
// connection.Frames.Err.
func Complete(ctx context.Context, caller connection.Caller, request *CompleteRequest, priority connection.Priority) (*CompleteResult, error) {
	result := &CompleteResult{}
	if err := callOnce(ctx, caller, OperationComplete, request, priority, result); err != nil {
		return nil, err
	}
	return result, nil
}

// Embed asks the rows of the plugin, in order, for one vector for each text. No text is no call. A result
// with another number of vectors than texts is an error.
func Embed(ctx context.Context, caller connection.Caller, request *EmbedRequest, priority connection.Priority) (*EmbedResult, error) {
	result := &EmbedResult{}
	if len(request.GetTexts()) == 0 {
		return result, nil
	}
	if err := callOnce(ctx, caller, OperationEmbed, request, priority, result); err != nil {
		return nil, err
	}
	if len(result.GetVectors()) != len(request.GetTexts()) {
		return nil, fmt.Errorf("llm: %s answered %d vectors for %d texts", OperationEmbed, len(result.GetVectors()), len(request.GetTexts()))
	}
	return result, nil
}

// Rerank asks the rows of the plugin, in order, for one score for each document. No document is no call. An
// empty query is an error with no call. A result with another number of scores than documents is an error.
func Rerank(ctx context.Context, caller connection.Caller, request *RerankRequest, priority connection.Priority) (*RerankResult, error) {
	result := &RerankResult{}
	if len(request.GetDocuments()) == 0 {
		return result, nil
	}
	if strings.TrimSpace(request.GetQuery()) == "" {
		return nil, fmt.Errorf("llm: %s needs a query", OperationRerank)
	}
	if err := callOnce(ctx, caller, OperationRerank, request, priority, result); err != nil {
		return nil, err
	}
	if len(result.GetScores()) != len(request.GetDocuments()) {
		return nil, fmt.Errorf("llm: %s answered %d scores for %d documents", OperationRerank, len(result.GetScores()), len(request.GetDocuments()))
	}
	return result, nil
}

// GetProfile asks the rows of the plugin, in order, for the profile of their embedding model. The call is
// background.
func GetProfile(ctx context.Context, caller connection.Caller) (*Profile, error) {
	result := &Profile{}
	if err := callOnce(ctx, caller, OperationProfile, &ProfileRequest{}, connection.PriorityBackground, result); err != nil {
		return nil, err
	}
	return result, nil
}

func callOnce(
	ctx context.Context, caller connection.Caller, operation string, request proto.Message, priority connection.Priority, result proto.Message,
) error {
	payload, err := proto.Marshal(request)
	if err != nil {
		return fmt.Errorf("llm: encode %s: %w", operation, err)
	}
	frames, err := caller.Call(ctx, operation, payload, priority)
	if err != nil {
		return err
	}
	_, done, err := connection.Collect(frames)
	if err != nil {
		return err
	}
	if err := proto.Unmarshal(done, result); err != nil {
		return fmt.Errorf("llm: decode %s: %w", operation, err)
	}
	return nil
}

// ChatStream is the frames of one turn. A reader calls Next until it returns false, then reads Err, and
// reads Finish where Err is nil.
type ChatStream struct {
	frames connection.Frames
	frame  *ChatFrame
	finish *ChatFinish
	err    error
}

// Chat asks the rows of the plugin, in order, for the next turn of a conversation. The core goes to the
// next row only when a row fails before its first frame. The caller closes the stream.
func Chat(ctx context.Context, caller connection.Caller, request *ChatRequest, priority connection.Priority) (*ChatStream, error) {
	payload, err := proto.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("llm: encode %s: %w", OperationChat, err)
	}
	frames, err := caller.Call(ctx, OperationChat, payload, priority)
	if err != nil {
		return nil, err
	}
	return &ChatStream{frames: frames}, nil
}

// Next moves to the next frame of the turn. It returns false at the end of the turn and on an error.
func (s *ChatStream) Next() bool {
	if s.err != nil || !s.frames.Next() {
		return false
	}
	frame := &ChatFrame{}
	if err := proto.Unmarshal(s.frames.Data(), frame); err != nil {
		s.err = fmt.Errorf("llm: decode a frame of %s: %w", OperationChat, err)
		return false
	}
	s.frame = frame
	return true
}

// Frame is the frame Next moved to.
func (s *ChatStream) Frame() *ChatFrame { return s.frame }

// Finish is why the turn ended, once Next returned false and Err is nil.
func (s *ChatStream) Finish() *ChatFinish {
	if s.finish == nil && s.err == nil && s.frames.Err() == nil {
		finish := &ChatFinish{}
		if err := proto.Unmarshal(s.frames.Done(), finish); err != nil {
			s.err = fmt.Errorf("llm: decode the finish of %s: %w", OperationChat, err)
			return nil
		}
		s.finish = finish
	}
	return s.finish
}

// Err returns why the turn ended: nil after the finish, the errors of connection.Frames.Err, or the error
// of a frame or a finish that is no message of the contract, which the TypeScript client throws.
func (s *ChatStream) Err() error { return errors.Join(s.err, s.frames.Err()) }

// Close ends the stream and cancels the call if the turn has not ended.
func (s *ChatStream) Close() error { return s.frames.Close() }
