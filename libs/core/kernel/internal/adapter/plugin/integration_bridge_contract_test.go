package plugin_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/pluginhost"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchsource"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/sourcebridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// bridgeHost is what every bridge is opened on.
type bridgeHost interface {
	bridge.Activator
	sourcebridge.Plugins
}

// contractPluginID is the plugin the bridges are opened on.
const contractPluginID = "md.numen.stub"

// bridgeCall makes one call through an open bridge and reports whether the
// returned value is the zero value.
type bridgeCall func(ctx context.Context) (isZero bool, err error)

// bridgeContract opens one bridge on a host and names a call through it.
type bridgeContract struct {
	name string
	open func(ctx context.Context, host bridgeHost) (io.Closer, bridgeCall, error)
}

func isZeroResult[T any](value T, err error) (bool, error) {
	return reflect.ValueOf(&value).Elem().IsZero(), err
}

func createContract[B io.Closer](bridge B, err error, call func(B, context.Context) (bool, error)) (io.Closer, bridgeCall, error) {
	if err != nil {
		return nil, nil, err
	}
	return bridge, func(ctx context.Context) (bool, error) { return call(bridge, ctx) }, nil
}

func getBridgeContracts() []bridgeContract {
	return []bridgeContract{
		{"search source", func(ctx context.Context, h bridgeHost) (io.Closer, bridgeCall, error) {
			b, err := sourcebridge.Open(ctx, h, h, nil)
			return createContract(b, err, func(b *sourcebridge.Source, ctx context.Context) (bool, error) {
				return isZeroResult(b.IndexTexts(ctx, searchsource.Place{}, []searchsource.Text{{ChunkID: "1", Text: "a"}}))
			})
		}},
	}
}

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

func isUnavailable(err error) bool { return errors.Is(err, domain.ErrUnavailable) }

func isNotInstalled(err error) bool {
	var unavailable *plugindomain.UnavailableError
	return errors.As(err, &unavailable) && unavailable.Reason == plugindomain.NotInstalled && isUnavailable(err)
}

func TestEveryBridgeAnswersAPluginThatIsDownWithAnErrorAndNoValue(t *testing.T) {
	for _, tc := range getBridgeContracts() {
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
	for _, tc := range getBridgeContracts() {
		t.Run(tc.name, func(t *testing.T) {
			host := container.NewPluginRoles(testsupport.GetMachine(), "desktop.editor")
			defer func() {
				if err := host.Teardown.TeardownAll(t.Context(), 50*time.Millisecond); err != nil {
					t.Errorf("TeardownAll: %v", err)
				}
			}()
			for _, problem := range listRefusalProblems(t.Context(), pluginhost.Join(host), tc.open, isNotInstalled) {
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
		want string
	}{
		{"an error with no value", func(context.Context, bridgeHost) (io.Closer, bridgeCall, error) {
			return nil, nil, &plugindomain.UnavailableError{Reason: plugindomain.NotInstalled}
		}, ""},
		{"a call that fails as it should", func(context.Context, bridgeHost) (io.Closer, bridgeCall, error) {
			return brokenBridge{}, func(context.Context) (bool, error) {
				return true, &plugindomain.UnavailableError{Reason: plugindomain.NotInstalled}
			}, nil
		}, ""},
		{"a bridge returned with the error", func(context.Context, bridgeHost) (io.Closer, bridgeCall, error) {
			return brokenBridge{}, nil, &plugindomain.UnavailableError{Reason: plugindomain.NotInstalled}
		}, "a bridge was returned"},
		{"a call that succeeds", func(context.Context, bridgeHost) (io.Closer, bridgeCall, error) {
			return brokenBridge{}, func(context.Context) (bool, error) { return false, nil }, nil
		}, "returned no error"},
		{"a value with the error", func(context.Context, bridgeHost) (io.Closer, bridgeCall, error) {
			return brokenBridge{}, func(context.Context) (bool, error) {
				return false, &plugindomain.UnavailableError{Reason: plugindomain.NotInstalled}
			}, nil
		}, "a value was returned"},
		{"a bridge that fails to close", func(context.Context, bridgeHost) (io.Closer, bridgeCall, error) {
			return failingBridge{}, func(context.Context) (bool, error) {
				return true, &plugindomain.UnavailableError{Reason: plugindomain.NotInstalled}
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

// A bridge takes its plugin only for the length of a call, so an open bridge lets the plugin go idle, and closing it
// twice gives back nothing more.
func TestABridgeReleasesThePluginAfterEachCall(t *testing.T) {
	for _, tc := range getBridgeContracts() {
		t.Run(tc.name, func(t *testing.T) {
			host := newLeaseHost(t)
			bridge, call, err := tc.open(t.Context(), host)
			if err != nil {
				t.Fatal(err)
			}
			// The stub serves no texts, so the call fails; it holds the plugin all the same.
			if _, err := call(t.Context()); err == nil {
				t.Fatal("the stub answered texts it does not serve")
			}
			if host.Acquired.Load() == 0 {
				t.Fatal("the call acquired no reference")
			}
			if acquired, released := host.Acquired.Load(), host.Released.Load(); acquired != released {
				t.Errorf("after the call %d references are held", acquired-released)
			}
			for range 2 {
				if err := bridge.Close(); err != nil {
					t.Fatal(err)
				}
			}
			if acquired, released := host.Acquired.Load(), host.Released.Load(); acquired != released {
				t.Errorf("acquired %d references, gave back %d", acquired, released)
			}
		})
	}
}
