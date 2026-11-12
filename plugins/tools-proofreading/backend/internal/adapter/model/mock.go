package model

import (
	"context"
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/plugins/tools-proofreading/backend/domain/numbered"
)

// MockMark is appended by the mock proofreader to the end of every line in the prompt.
const MockMark = "[proofread]"

// CompleteMock answers the last message of a request as a proofreader would, with every numbered
// line of the prompt returned marked.
func CompleteMock(ctx context.Context, request connection.CompleteRequest) (connection.CompleteResult, error) {
	if err := ctx.Err(); err != nil {
		return connection.CompleteResult{}, err
	}
	messages := request.Messages
	if len(messages) == 0 {
		return connection.CompleteResult{}, nil
	}
	var rows []string
	segments := strings.Split(messages[len(messages)-1].Text, numbered.Opens)
	for _, segment := range segments[1:] {
		number, text, ok := strings.Cut(segment, numbered.Closes)
		if _, err := strconv.Atoi(number); !ok || err != nil {
			continue
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		rows = append(rows, number+"|"+text+" "+MockMark)
	}
	return connection.CompleteResult{Text: strings.Join(rows, "\n")}, nil
}
