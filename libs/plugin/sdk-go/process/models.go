package process

import (
	"cmp"
	"context"
	"slices"
	"strconv"
	"sync"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/performance"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

// ModelSessions tracks the execution providers of the model stages open in one plugin process.
type ModelSessions struct {
	mu       sync.Mutex
	counts   map[modelProvider]int
	retry    *time.Timer
	revision uint64
	isClosed bool
	server   *server.Server
	report   func(string, error)
}

type modelProvider struct{ stage, provider string }

// NewModelSessions returns the model stages of the process served at endpoint.
func NewModelSessions(s *server.Server, report func(string, error)) *ModelSessions {
	m := &ModelSessions{counts: make(map[modelProvider]int), server: s, report: report}
	s.OnClose(m.close)
	return m
}

// Track adds the stages of one successfully opened model. The returned function removes them on close.
func (m *ModelSessions) Track(ctx context.Context, stages map[string]string) func() {
	rows := make([]modelProvider, 0, len(stages))
	for stage, provider := range stages {
		rows = append(rows, modelProvider{stage: stage, provider: provider})
	}
	reportCtx := context.WithoutCancel(ctx)
	m.update(reportCtx, rows, 1)
	return sync.OnceFunc(func() { m.update(reportCtx, rows, -1) })
}

func (m *ModelSessions) update(ctx context.Context, rows []modelProvider, delta int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.isClosed {
		return
	}
	changed := false
	for _, row := range rows {
		count := m.counts[row]
		next := count + delta
		if count == 0 || next == 0 {
			changed = true
		}
		if next == 0 {
			delete(m.counts, row)
		} else {
			m.counts[row] = next
		}
	}
	if changed {
		m.revision++
		m.publishLocked(ctx)
	}
}

const (
	reportTimeout = 2 * time.Second
	retryInterval = time.Second
)

func (m *ModelSessions) close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.isClosed = true
	if m.retry != nil {
		m.retry.Stop()
	}
	return nil
}

func (m *ModelSessions) retryReport(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.isClosed {
		m.publishLocked(ctx)
	}
}

func (m *ModelSessions) publishLocked(ctx context.Context) {
	if m.retry != nil {
		m.retry.Stop()
		m.retry = nil
	}
	if err := m.publish(ctx); err != nil {
		m.report("report the execution providers of open models", err)
		switch connect.CodeOf(err) {
		case connect.CodeUnknown, connect.CodeDeadlineExceeded, connect.CodeUnavailable, connect.CodeResourceExhausted:
			m.retry = time.AfterFunc(retryInterval, func() { m.retryReport(ctx) })
		}
	}
}

func (m *ModelSessions) publish(ctx context.Context) error {
	endpoint, err := m.server.GetHostEndpoint()
	if err != nil {
		return err
	}
	rows := make([]*v1.ModelProvider, 0, len(m.counts))
	for row := range m.counts {
		rows = append(rows, &v1.ModelProvider{Stage: row.stage, Provider: row.provider})
	}
	slices.SortFunc(rows, func(a, b *v1.ModelProvider) int {
		if a.GetStage() != b.GetStage() {
			return cmp.Compare(a.GetStage(), b.GetStage())
		}
		return cmp.Compare(a.GetProvider(), b.GetProvider())
	})
	rpc := numenv1connect.NewPluginServiceClient(hostcall.HTTPClient(), endpoint.BaseURL, endpoint.Options()...)
	ctx, cancel := context.WithTimeout(ctx, reportTimeout)
	defer cancel()
	_, err = rpc.ReportModelProviders(ctx, connect.NewRequest(&v1.ReportModelProvidersRequest{Providers: rows, Revision: m.revision}))
	return err
}

// GetCompute reads the host policy in force when a model opens.
func (m *ModelSessions) GetCompute() performance.Compute {
	config := m.server.Lifecycle().AllConfigs()
	threads, _ := strconv.Atoi(config[performance.ThreadsKey])
	return performance.Compute{Provider: config[performance.ProviderKey], ThreadLimit: threads}
}
