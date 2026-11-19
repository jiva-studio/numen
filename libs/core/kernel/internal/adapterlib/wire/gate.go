package wire

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/logs"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/broadcast"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
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
}

func (g *Gate) SetCallTrace(clock CallClock, record func(logs.Entry)) {
	g.trace.Store(&callTrace{clock: clock, record: record})
}

// Close rejects all new requests permanently. It returns after the requests
// already accepted have completed, so the index and embedder stay available
// for their whole duration.
func (g *Gate) Close() {
	g.closed.Store(true)
	<-g.questions.seal()
}

// IsClosed reports whether the window is shut.
func (g *Gate) IsClosed() bool { return g.closed.Load() }

// BeginQuestion counts a question being answered, and reports false once the window is shut.
func (g *Gate) BeginQuestion() bool { return g.questions.begin() }

// FinishQuestion ends a question that BeginQuestion counted.
func (g *Gate) FinishQuestion() { g.questions.finish() }

// CountQuestions counts each call to a service among the questions the window
// is answering, refuses it once the window is shut, and records a structured log entry when a
// call tracing is enabled.
func (g *Gate) CountQuestions() connect.HandlerOption {
	return connect.WithInterceptors(connect.UnaryInterceptorFunc(
		func(next connect.UnaryFunc) connect.UnaryFunc {
			return func(ctx context.Context, r connect.AnyRequest) (connect.AnyResponse, error) {
				if !g.questions.begin() {
					return nil, connect.NewError(connect.CodeUnavailable, ErrShut)
				}
				defer g.questions.finish()

				trace := g.trace.Load()
				if trace == nil || trace.clock == nil || trace.record == nil || isLogProcedure(r.Spec().Procedure) {
					return next(ctx, r)
				}
				start := trace.clock.Now()
				resp, err := next(ctx, r)
				trace.recordCall(ctx, start, r, resp, err)
				return resp, err
			}
		},
	))
}

func isLogProcedure(procedure string) bool {
	return strings.HasPrefix(procedure, "/numen.v1.LogService/")
}

func (t *callTrace) recordCall(
	ctx context.Context, start time.Time, req connect.AnyRequest, resp connect.AnyResponse, err error,
) {
	duration := t.clock.Now().Sub(start)
	service, method := splitProcedure(req.Spec().Procedure)
	level := getProcedureLevel(method, err)
	durationMs := strconv.FormatFloat(float64(duration.Microseconds())/1000.0, 'f', -1, 64)
	props := map[string]string{
		"service":    strconv.Quote(service),
		"method":     strconv.Quote(method),
		"request":    encodeMessageJSON(req.Any()),
		"durationMs": durationMs,
	}
	caller := describeCaller(plugindomain.GetCaller(ctx))
	if caller != "" {
		props["caller"] = strconv.Quote(caller)
	}
	if err != nil {
		props["error"] = strconv.Quote(err.Error())
	} else if resp != nil {
		props["response"] = encodeMessageJSON(resp.Any())
	}
	t.record(logs.Entry{
		Time:       start,
		Level:      level,
		Source:     logs.SourceRPC,
		Message:    formatRPCMessage(service, method, caller, durationMs),
		Properties: props,
	})
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

func encodeMessageJSON(msg any) string {
	if msg == nil {
		return "{}"
	}
	if pb, ok := msg.(proto.Message); ok {
		if raw, err := protojson.Marshal(pb); err == nil {
			return string(raw)
		}
	}
	if raw, err := json.Marshal(msg); err == nil {
		return string(raw)
	}
	return "{}"
}
