package log_test

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/plugins/todos/backend/internal/adapter/log"
)

func TestSlogErrorWritesTheMessageAndTheError(t *testing.T) {
	var out bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&out, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	log.Slog{}.Error("start the index", errors.New("boom"))

	if got := out.String(); !strings.Contains(got, "start the index") || !strings.Contains(got, "boom") {
		t.Fatalf("log = %q, want the message and the error", got)
	}
}
