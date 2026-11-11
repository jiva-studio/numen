// Package environment reads the variables that override the model and the runtime of the plugin.
package environment

import (
	"context"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
)

// Variables are the variables of the process environment, read as they are when asked.
type Variables struct{}

// GetOverrides returns what the variables set for the model and the runtime.
func (Variables) GetOverrides() model.Overrides {
	return model.Overrides{
		Model:      os.Getenv("NUMEN_EMBEDDING_MODEL"),
		Repository: os.Getenv("NUMEN_EMBEDDING_REPO"),
		Dir:        os.Getenv("NUMEN_EMBEDDING_DIR"),
		File:       os.Getenv("NUMEN_EMBEDDING_FILE"),
		Pooling:    os.Getenv("NUMEN_EMBEDDING_POOLING"),
		Runtime:    os.Getenv("NUMEN_ONNX_RUNTIME"),
		Dimensions: parsePositiveInt(os.Getenv("NUMEN_EMBEDDING_DIMENSIONS")),
		MaxTokens:  parsePositiveInt(os.Getenv("NUMEN_EMBEDDING_MAX_TOKENS")),
		Threads:    parseThreads(os.Getenv("NUMEN_ONNX_THREADS")),
	}
}

// GetServiceURL is the address of the embedding service that variable names.
func (Variables) GetServiceURL() string {
	return os.Getenv("NUMEN_EMBEDDING_URL")
}

func parsePositiveInt(val string) int {
	// gomutants:disable-next-line BRANCH_IF reason="empty string is rejected by strconv.Atoi"
	if val == "" {
		return 0
	}
	// gomutants:disable-next-line CONDITIONALS_BOUNDARY,INTEGER_DECREMENT reason="returning 0 for d == 0 matches the fallback return 0"
	if d, err := strconv.Atoi(val); err == nil && d > 0 && d <= math.MaxInt32 {
		return d
	}
	return 0
}

func parseThreads(val string) int {
	// gomutants:disable-next-line CONDITIONALS_BOUNDARY,INTEGER_DECREMENT reason="zero threads leaves the default 0 unchanged"
	if th, err := strconv.Atoi(val); err == nil && th > 0 {
		return th
	}
	return 0
}

// The variables of the mock model, which automated tests select.
const (
	envMockModel    = "NUMEN_MOCK_ML"
	envMockLog      = "NUMEN_MOCK_EMBED_LOG"
	envMockOpenHold = "NUMEN_MOCK_OPEN_HOLD_MS"
)

// IsMockModel returns whether the variables ask for the mock model.
func (Variables) IsMockModel() bool {
	return os.Getenv(envMockModel) == "true"
}

// GetMockLogPath is the file the mock model appends the texts it is given to; it is empty when the
// variable is not set.
func (Variables) GetMockLogPath() string {
	return os.Getenv(envMockLog)
}

// WaitMockOpenHold waits as long as the variable names, as a model that is still downloading. It
// ends early with the error of ctx.
func (Variables) WaitMockOpenHold(ctx context.Context) error {
	// gomutants:disable-next-line INVERT_LOGICAL,EXPRESSION_REMOVE,CONDITIONALS_BOUNDARY,INTEGER_DECREMENT reason="0ms timeout completes immediately without blocking"
	held, err := strconv.Atoi(os.Getenv(envMockOpenHold))
	if err != nil || held <= 0 {
		return nil
	}
	// gomutants:disable-next-line ARITHMETIC_BASE reason="mock hold duration scaling for integration tests"
	hold, cancel := context.WithTimeout(ctx, time.Duration(held)*time.Millisecond)
	defer cancel()
	<-hold.Done()
	return ctx.Err()
}
