package service

import (
	"context"
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/ai"
)

// MockMark is appended by the mock proofreader to the end of every line in the prompt.
const MockMark = "[proofread]"

// completeMock responds to a request as a model would, with no model: every numbered line of the
// prompt comes back with MockMark after its words. It touches nothing outside the process.
func completeMock(ctx context.Context, question *ai.Question) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	messages := question.GetMessages()
	if len(messages) == 0 {
		return "", nil
	}
	var rows []string
	segments := strings.Split(messages[len(messages)-1].GetText(), Opens)
	for _, segment := range segments[1:] {
		number, text, ok := strings.Cut(segment, Closes)
		if _, err := strconv.Atoi(number); !ok || err != nil {
			continue
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		rows = append(rows, number+"|"+text+" "+MockMark)
	}
	return strings.Join(rows, "\n"), nil
}
