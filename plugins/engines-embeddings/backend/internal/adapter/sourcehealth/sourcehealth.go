// Package sourcehealth keeps the health item source of the plugin in the list of the host: the text it
// reported last, so the host is called only when the text changes.
package sourcehealth

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
)

// key is the key of the health item that says the source of the vectors does not work.
const key = "source"

// settingSource is the setting a person changes to fix the source.
const settingSource = "source"

// reportLimit bounds one call to the host about the health of the plugin.
const reportLimit = 2 * time.Second

// OpenHealth connects to the list of health items of the host.
type OpenHealth func() (*health.Client, error)

// ReportFault writes a fault that nothing else handles.
type ReportFault func(what string, err error)

// Reporter reports the source of the vectors as the health item source of the plugin. A process started
// without a host reports nothing.
type Reporter struct {
	open   OpenHealth
	report ReportFault

	mu       sync.Mutex
	reported string
}

// New returns a Reporter that reaches the host through open and writes a failed call to report.
func New(open OpenHealth, report ReportFault) *Reporter {
	return &Reporter{open: open, report: report, mu: sync.Mutex{}, reported: ""}
}

// Set reports text as an error of the source, and an empty text clears the item.
func (r *Reporter) Set(ctx context.Context, text string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if text == r.reported {
		return
	}
	client, err := r.open()
	if errors.Is(err, hostcall.ErrNoHost) {
		return
	}
	if err != nil {
		r.report("embeddings: report the health of the source", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), reportLimit)
	defer cancel()
	if text == "" {
		err = client.Clear(ctx, key)
	} else {
		err = client.Report(ctx, health.Item{
			Key: key, Severity: health.Error, Text: text, Detail: "", Action: health.OpenSettings(settingSource),
		})
	}
	if err != nil {
		r.report("embeddings: report the health of the source", err)
		return
	}
	r.reported = text
}
