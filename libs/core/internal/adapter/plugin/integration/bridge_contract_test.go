package integration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/agentbridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/embedderbridge"
	"github.com/jiva-studio/numen/libs/core/port/completionport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// bridgeHost is what every bridge is opened on.
type bridgeHost interface {
	embedderbridge.Host
	agentbridge.Host
}

// bridgeCall makes one call through an open bridge and reports whether the
// returned value is the zero value.
type bridgeCall func(ctx context.Context) (isZero bool, err error)

// bridgeContract opens one bridge on a host and names a call through it.
type bridgeContract struct {
	name string
	open func(ctx context.Context, host bridgeHost) (io.Closer, bridgeCall, error)
}

// isZeroResult reports whether a call returned the zero value of its type.
func isZeroResult[T any](value T, err error) (bool, error) {
	return reflect.ValueOf(&value).Elem().IsZero(), err
}

// createContract pairs an open bridge with a call through it. An open that failed
// returns no bridge.
func createContract[B io.Closer](bridge B, err error, call func(B, context.Context) (bool, error)) (io.Closer, bridgeCall, error) {
	if err != nil {
		return nil, nil, err
	}
	return bridge, func(ctx context.Context) (bool, error) { return call(bridge, ctx) }, nil
}

// bridgeContracts is every bridge the host opens, each with a call that invokes
// its plugin.
var bridgeContracts = []bridgeContract{
	{"embedder", func(ctx context.Context, h bridgeHost) (io.Closer, bridgeCall, error) {
		b, err := embedderbridge.Open(ctx, h)
		return createContract(b, err, func(b *embedderbridge.Embedder, ctx context.Context) (bool, error) {
			return isZeroResult(b.Embed(ctx, []string{"a"}))
		})
	}},
	{"agent", func(ctx context.Context, h bridgeHost) (io.Closer, bridgeCall, error) {
		b, err := agentbridge.OpenForPlugin(ctx, h, contractPluginID)
		return createContract(b, err, func(b *agentbridge.Agent, ctx context.Context) (bool, error) {
			return isZeroResult(b.GetPrograms(ctx, nil))
		})
	}},
}

// contractPluginID is the plugin the bridges that are opened by id are opened on.
const contractPluginID = "md.numen.stub"

// listRefusalProblems opens a bridge on a plugin that cannot respond and lists
// what is wrong with its error behavior. A plugin that is down, and one that is
// not installed, fail the bridge with port.ErrUnavailable, at the open or at the
// call, and never with a value.
func listRefusalProblems(ctx context.Context, host bridgeHost, open func(context.Context, bridgeHost) (io.Closer, bridgeCall, error), isRefusal func(error) bool) (problems []string) {
	bridge, call, err := open(ctx, host)
	if err == nil {
		defer func() {
			if closeErr := bridge.Close(); closeErr != nil {
				problems = append(problems, fmt.Sprintf("closing the bridge failed: %v", closeErr))
			}
		}()
		var isZero bool
		isZero, err = call(ctx)
		if err == nil {
			return append(problems, "a call to a plugin that cannot answer returned no error")
		}
		if !isZero {
			problems = append(problems, "a value was returned with the error")
		}
	} else if bridge != nil {
		problems = append(problems, "a bridge was returned with the error")
	}
	if !isRefusal(err) {
		problems = append(problems, fmt.Sprintf("the error is %v, which is not the rejection asked for", err))
	}
	return problems
}

// What the rejection of a bridge is judged by.
func isUnavailable(err error) bool { return errors.Is(err, completionport.ErrUnavailable) }

func isNotInstalled(err error) bool {
	var unavailable *pluginport.UnavailableError
	return errors.As(err, &unavailable) && unavailable.Reason == pluginport.PluginNotInstalled && isUnavailable(err)
}

func TestEveryBridgeAnswersAPluginThatIsDownWithAnErrorAndNoValue(t *testing.T) {
	for _, tc := range bridgeContracts {
		t.Run(tc.name, func(t *testing.T) {
			host := newLeaseHost(t)
			host.EnsureErr = errors.New("the plugin is down")
			for _, problem := range listRefusalProblems(t.Context(), host, tc.open, isUnavailable) {
				t.Error(problem)
			}
			if host.Acquired.Load() != host.Released.Load() {
				t.Errorf("acquired %d references, gave back %d", host.Acquired.Load(), host.Released.Load())
			}
		})
	}
}

func TestBridges_UninstalledPluginReturnsUnavailable(t *testing.T) {
	for _, tc := range bridgeContracts {
		t.Run(tc.name, func(t *testing.T) {
			host := plugin.NewHost("desktop.editor")
			defer func() {
				if err := host.TeardownAll(t.Context(), 50*time.Millisecond); err != nil {
					t.Errorf("TeardownAll: %v", err)
				}
			}()
			for _, problem := range listRefusalProblems(t.Context(), host, tc.open, isNotInstalled) {
				t.Error(problem)
			}
		})
	}
}

type brokenBridge struct{}

func (brokenBridge) Close() error { return nil }

type failingBridge struct{}

func (failingBridge) Close() error { return errors.New("not closed") }

// The check rejects each way a bridge can respond incorrectly for a plugin that
// cannot respond, and accepts the correct way.
func TestErrorCheck_RejectsBridgeWithWrongResponse(t *testing.T) {
	cases := []struct {
		name string
		open func(context.Context, bridgeHost) (io.Closer, bridgeCall, error)
		// want is what the problems say, and empty where there are none.
		want string
	}{
		{"an error with no value", func(context.Context, bridgeHost) (io.Closer, bridgeCall, error) {
			return nil, nil, &pluginport.UnavailableError{Reason: pluginport.PluginNotInstalled}
		}, ""},
		{"a call that fails as it should", func(context.Context, bridgeHost) (io.Closer, bridgeCall, error) {
			return brokenBridge{}, func(context.Context) (bool, error) {
				return true, &pluginport.UnavailableError{Reason: pluginport.PluginNotInstalled}
			}, nil
		}, ""},
		{"a bridge returned with the error", func(context.Context, bridgeHost) (io.Closer, bridgeCall, error) {
			return brokenBridge{}, nil, &pluginport.UnavailableError{Reason: pluginport.PluginNotInstalled}
		}, "a bridge was returned"},
		{"a call that succeeds", func(context.Context, bridgeHost) (io.Closer, bridgeCall, error) {
			return brokenBridge{}, func(context.Context) (bool, error) { return false, nil }, nil
		}, "returned no error"},
		{"a value with the error", func(context.Context, bridgeHost) (io.Closer, bridgeCall, error) {
			return brokenBridge{}, func(context.Context) (bool, error) {
				return false, &pluginport.UnavailableError{Reason: pluginport.PluginNotInstalled}
			}, nil
		}, "a value was returned"},
		{"a bridge that fails to close", func(context.Context, bridgeHost) (io.Closer, bridgeCall, error) {
			return failingBridge{}, func(context.Context) (bool, error) {
				return true, &pluginport.UnavailableError{Reason: pluginport.PluginNotInstalled}
			}, nil
		}, "closing the bridge failed"},
		{"an error that is not unavailable", func(context.Context, bridgeHost) (io.Closer, bridgeCall, error) {
			return brokenBridge{}, func(context.Context) (bool, error) { return true, errors.New("boom") }, nil
		}, "not the rejection asked for"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := listRefusalProblems(t.Context(), nil, tc.open, isNotInstalled)
			joined := strings.Join(got, "; ")
			if tc.want == "" && len(got) != 0 || !strings.Contains(joined, tc.want) {
				t.Errorf("problems found: %q, want %q", joined, tc.want)
			}
		})
	}
}

func TestADoubleCloseProducesTheReferenceBackOnce(t *testing.T) {
	for _, tc := range bridgeContracts {
		t.Run(tc.name, func(t *testing.T) {
			host := newLeaseHost(t)
			bridge, _, err := tc.open(t.Context(), host)
			if err != nil {
				t.Fatal(err)
			}
			target := host.Acquired.Load()
			if target == 0 {
				t.Fatal("opening the bridge acquired no reference")
			}
			if got := host.Released.Load(); got != 0 {
				t.Fatalf("%d references were given back before Close", got)
			}
			for range 2 {
				if err := bridge.Close(); err != nil {
					t.Fatal(err)
				}
			}
			if got := host.Released.Load(); got != target {
				t.Errorf("held %d references, gave back %d", target, got)
			}
		})
	}
}
