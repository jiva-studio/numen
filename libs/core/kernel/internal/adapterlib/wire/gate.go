package wire

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/logs"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/broadcast"
)

// ErrShut is returned for a request to a window that has been shut; the window answers no further requests.
var ErrShut = errors.New("this window is shut")

// CallClock provides the current instant for measuring RPC duration.
type CallClock interface {
	Now() time.Time
}

// callTrace records structured log entries for unary RPC calls passing through a Gate.
type callTrace struct {
	clock  CallClock
	record func(logs.Entry)
}

type inflight struct {
	in broadcast.Inflight
}

func (w *inflight) begin() bool           { return w.in.Begin() }
func (w *inflight) finish()               { w.in.Finish() }
func (w *inflight) seal() <-chan struct{} { return w.in.Seal() }

// Gate is the door on every question of the window. It counts the questions being answered and refuses
// new ones once the window is shut.
type Gate struct {
	closed    atomic.Bool
	questions inflight
	trace     atomic.Pointer[callTrace]

	mu      sync.Mutex
	streams map[*context.CancelFunc]struct{}
}

func (g *Gate) SetCallTrace(clock CallClock, record func(logs.Entry)) {
	g.trace.Store(&callTrace{clock: clock, record: record})
}

// Close rejects all new requests permanently and cancels the streams that are open. It returns after the
// requests already accepted have completed, so the index and embedder stay available for their whole
// duration.
func (g *Gate) Close() {
	g.mu.Lock()
	g.closed.Store(true)
	for cancel := range g.streams {
		(*cancel)()
	}
	g.mu.Unlock()
	<-g.questions.seal()
}

func (g *Gate) openStream(ctx context.Context) (context.Context, func()) {
	ctx, cancel := context.WithCancel(ctx)
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed.Load() {
		cancel()
		return ctx, func() {}
	}
	if g.streams == nil {
		g.streams = make(map[*context.CancelFunc]struct{})
	}
	g.streams[&cancel] = struct{}{}
	return ctx, func() {
		g.mu.Lock()
		delete(g.streams, &cancel)
		g.mu.Unlock()
		cancel()
	}
}

// IsClosed reports whether the window is shut.
func (g *Gate) IsClosed() bool { return g.closed.Load() }

// BeginQuestion counts a question being answered, and reports false once the window is shut.
func (g *Gate) BeginQuestion() bool { return g.questions.begin() }

// FinishQuestion ends a question that BeginQuestion counted.
func (g *Gate) FinishQuestion() { g.questions.finish() }

// CountQuestions counts each call to a service among the questions the window
// is answering, refuses it once the window is shut, and records a structured log entry when a
// call tracing is enabled. A stream is counted until it ends, and Close cancels it.
func (g *Gate) CountQuestions() connect.HandlerOption {
	return connect.WithInterceptors(counter{gate: g})
}

type counter struct {
	gate *Gate
}

func (c counter) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, r connect.AnyRequest) (connect.AnyResponse, error) {
		var resp connect.AnyResponse
		err := c.gate.answer(ctx, r.Spec().Procedure, func(ctx context.Context) error {
			var err error
			resp, err = next(ctx, r)
			return err
		})
		return resp, err
	}
}

func (c counter) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (c counter) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		return c.gate.answer(ctx, conn.Spec().Procedure, func(ctx context.Context) error {
			ctx, release := c.gate.openStream(ctx)
			defer release()
			return next(ctx, conn)
		})
	}
}

func (g *Gate) answer(ctx context.Context, procedure string, serve func(context.Context) error) error {
	if !g.questions.begin() {
		return connect.NewError(connect.CodeUnavailable, ErrShut)
	}
	defer g.questions.finish()

	trace := g.trace.Load()
	if trace == nil || trace.clock == nil || trace.record == nil || isLogProcedure(procedure) {
		return serve(ctx)
	}
	start := trace.clock.Now()
	err := serve(ctx)
	trace.recordCall(ctx, start, procedure, err)
	return err
}

func isLogProcedure(procedure string) bool {
	return strings.HasPrefix(procedure, "/numen.v1.LogService/")
}

func (t *callTrace) recordCall(
	ctx context.Context, start time.Time, procedure string, err error,
) {
	duration := t.clock.Now().Sub(start)
	service, method := splitProcedure(procedure)
	level := getProcedureLevel(method, err)
	durationMs := strconv.FormatFloat(float64(duration.Microseconds())/1000.0, 'f', -1, 64)
	props := map[string]string{
		"service":    strconv.Quote(service),
		"method":     strconv.Quote(method),
		"durationMs": durationMs,
	}
	caller := describeCaller(plugindomain.GetCaller(ctx))
	if caller != "" {
		props["caller"] = strconv.Quote(caller)
	}
	if err != nil {
		props["error"] = strconv.Quote(connect.CodeOf(err).String())
		if isDetailed(service, connect.CodeOf(err)) {
			props["detail"] = strconv.Quote(truncateDetail(err.Error()))
		}
	}
	t.record(logs.Entry{
		Time:       start,
		Level:      level,
		Source:     logs.SourceRPC,
		Message:    formatRPCMessage(service, method, caller, durationMs),
		Properties: props,
	})
}

func isDetailed(service string, code connect.Code) bool {
	return service == "IdentityService" && (code == connect.CodeInternal || code == connect.CodeUnknown)
}

// maxDetailRunes bounds the error text of a failed call in the log.
const maxDetailRunes = 400

func truncateDetail(text string) string {
	runes := []rune(text)
	if len(runes) > maxDetailRunes {
		return string(runes[:maxDetailRunes])
	}
	return text
}

func formatRPCMessage(service, method, caller, durationMs string) string {
	msg := "RPC: " + service + "/" + method
	if caller != "" {
		msg += " [" + caller + "]"
	}
	return msg + " (" + durationMs + " ms)"
}

func splitProcedure(procedure string) (service, method string) {
	trimmed := strings.TrimPrefix(procedure, "/")
	fullService, methodName, ok := strings.Cut(trimmed, "/")
	if !ok {
		return trimmed, ""
	}
	if idx := strings.LastIndex(fullService, "."); idx >= 0 {
		fullService = fullService[idx+1:]
	}
	return fullService, methodName
}

func getProcedureLevel(method string, err error) logs.Level {
	if err != nil {
		return logs.LevelError
	}
	if method == "WriteOpenTabs" || method == "WriteTabLinks" {
		return logs.LevelDebug
	}
	for _, prefix := range []string{
		"Create", "Rename", "Remove", "Delete", "Execute", "Run", "Write", "Save", "Join",
		"Add", "Open", "Move", "Empty", "Update", "Retry", "Install", "Uninstall", "Choose",
	} {
		if strings.HasPrefix(method, prefix) {
			return logs.LevelInfo
		}
	}
	return logs.LevelDebug
}

func describeCaller(caller plugindomain.Caller, ok bool) string {
	if !ok {
		return ""
	}
	if caller.PluginID != "" {
		return caller.PluginID
	}
	return caller.Who
}
