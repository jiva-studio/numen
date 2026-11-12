package service

import (
	"context"
	"errors"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/console"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/usecase/serving"
)

// sourceKey is the key of the health item that says the source of the vectors does not work.
const sourceKey = "source"

// settingSource is the setting a person changes to fix the source.
const settingSource = "source"

// maxHealthText is the longest text the host takes for an item.
const maxHealthText = 120

// reportLimit bounds one call to the host about the health of the plugin.
const reportLimit = 2 * time.Second

// openHealth connects to the list of health items of the host.
type openHealth func() (*health.Client, error)

// sourceHealth reports the source of the vectors as the health item source of the plugin: an error
// while the settings name a source that cannot answer (no connection, a connection that fails, a local
// model that is missing), and no item once a call succeeds. The host is called only when that changes,
// and a process started without a host reports nothing.
type sourceHealth struct {
	open openHealth

	mu       sync.Mutex
	reported string
}

func newSourceHealth(open openHealth) *sourceHealth {
	return &sourceHealth{open: open, mu: sync.Mutex{}, reported: ""}
}

// observe takes the outcome of a call of the model: a precondition the settings do not meet reports
// the item, a call that succeeded clears it, and any other failure leaves it as it is.
func (s *sourceHealth) observe(ctx context.Context, err error) {
	failure, isPrecondition := errors.AsType[serving.PreconditionError](err)
	switch {
	case err == nil:
		s.update(ctx, "")
	case isPrecondition:
		s.update(ctx, getHealthText(failure.Error()))
	}
}

func (s *sourceHealth) update(ctx context.Context, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if text == s.reported {
		return
	}
	client, err := s.open()
	if errors.Is(err, hostcall.ErrNoHost) {
		return
	}
	if err != nil {
		console.Report("embeddings: report the health of the source", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), reportLimit)
	defer cancel()
	if text == "" {
		err = client.Clear(ctx, sourceKey)
	} else {
		err = client.Report(ctx, health.Item{
			Key: sourceKey, Severity: health.Error, Text: text, Detail: "", Action: health.OpenSettings(settingSource),
		})
	}
	if err != nil {
		console.Report("embeddings: report the health of the source", err)
		return
	}
	s.reported = text
}

// getHealthText is the message as a sentence that fits an item: its first letter upper case, cut at the
// limit of the host.
func getHealthText(message string) string {
	if message == "" {
		return ""
	}
	first, size := utf8.DecodeRuneInString(message)
	text := string(unicode.ToUpper(first)) + message[size:]
	if runes := []rune(text); len(runes) > maxHealthText {
		text = string(runes[:maxHealthText-1]) + "…"
	}
	return text
}
