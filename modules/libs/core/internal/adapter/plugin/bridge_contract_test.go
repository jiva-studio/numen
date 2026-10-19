package plugin

import (
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// bridgeHost is what every bridge is opened on.
type bridgeHost interface {
	EmbedderPluginHost
	RecogniserPluginHost
	TranscriberPluginHost
	DownloaderPluginHost
	DocumentPluginHost
	BookPluginHost
	AgentPluginHost
	ProofreaderPluginHost
	CompletionPluginHost
}

// bridgeCall makes one call through an open bridge and says whether the value it
// returned is the zero value.
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

// bridgeContracts is every bridge the host opens, each with a call that reaches
// its plugin.
var bridgeContracts = []bridgeContract{
	{"embedder", func(ctx context.Context, h bridgeHost) (io.Closer, bridgeCall, error) {
		b, err := OpenEmbedderBridge(ctx, h)
		return createContract(b, err, func(b *EmbedderBridge, ctx context.Context) (bool, error) {
			return isZeroResult(b.Embed(ctx, []string{"a"}))
		})
	}},
	{"recogniser", func(ctx context.Context, h bridgeHost) (io.Closer, bridgeCall, error) {
		b, err := OpenRecogniserBridge(ctx, h)
		return createContract(b, err, func(b *RecogniserBridge, ctx context.Context) (bool, error) {
			return isZeroResult(b.Recognise(ctx, image.NewRGBA(image.Rect(0, 0, 1, 1))))
		})
	}},
	{"transcriber", func(ctx context.Context, h bridgeHost) (io.Closer, bridgeCall, error) {
		b, err := OpenTranscriberBridge(ctx, h)
		return createContract(b, err, func(b *TranscriberBridge, ctx context.Context) (bool, error) {
			return isZeroResult(b.Transcribe(ctx, port.Audio{}))
		})
	}},
	{"downloader", func(ctx context.Context, h bridgeHost) (io.Closer, bridgeCall, error) {
		b, err := OpenDownloaderBridge(ctx, h)
		return createContract(b, err, func(b *DownloaderBridge, ctx context.Context) (bool, error) {
			return isZeroResult(b.GetMetadata(ctx, "https://example.org"))
		})
	}},
	{"document", func(ctx context.Context, h bridgeHost) (io.Closer, bridgeCall, error) {
		b, err := OpenDocumentBridge(ctx, h)
		return createContract(b, err, func(b *DocumentBridge, ctx context.Context) (bool, error) {
			return isZeroResult(b.Read(ctx, []byte("%PDF")))
		})
	}},
	{"book", func(ctx context.Context, h bridgeHost) (io.Closer, bridgeCall, error) {
		b, err := OpenBookBridge(ctx, h)
		return createContract(b, err, func(b *BookBridge, ctx context.Context) (bool, error) {
			return isZeroResult(b.ReadBook(ctx, []byte("PK")))
		})
	}},
	{"agent", func(ctx context.Context, h bridgeHost) (io.Closer, bridgeCall, error) {
		b, err := OpenAgentBridgeForPlugin(ctx, h, contractPluginID)
		return createContract(b, err, func(b *AgentBridge, ctx context.Context) (bool, error) {
			return isZeroResult(b.GetPrograms(ctx, nil))
		})
	}},
	{"proofreader", func(_ context.Context, h bridgeHost) (io.Closer, bridgeCall, error) {
		b, err := OpenProofreaderBridge(h)
		return createContract(b, err, func(b *ProofreaderBridge, ctx context.Context) (bool, error) {
			return isZeroResult(b.ProofreadScan(ctx, []domain.ProofreadPage{{Number: 1}}, 0))
		})
	}},
	{"completion", func(ctx context.Context, h bridgeHost) (io.Closer, bridgeCall, error) {
		b, err := OpenCompletionBridge(ctx, h, contractPluginID)
		return createContract(b, err, func(b *CompletionBridge, ctx context.Context) (bool, error) {
			return isZeroResult(b.Complete(ctx, "", port.Question{}))
		})
	}},
}

// contractPluginID is the plugin the bridges that are opened by id are opened on.
const contractPluginID = "md.numen.stub"

// listRefusalProblems opens a bridge on a plugin that cannot answer and lists
// what is wrong with the way it refuses. A plugin that is down, and one that is
// not installed, fail the bridge with port.ErrUnavailable, at the open or at the
// call, and never with a value.
func listRefusalProblems(ctx context.Context, host bridgeHost, open func(context.Context, bridgeHost) (io.Closer, bridgeCall, error), isRefusal func(error) bool) []string {
	var problems []string
	bridge, call, err := open(ctx, host)
	if err == nil {
		defer func() { _ = bridge.Close() }()
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
		problems = append(problems, fmt.Sprintf("the error is %v, which is not the refusal asked for", err))
	}
	return problems
}

// What the refusal of a bridge is judged by.
func isUnavailable(err error) bool { return errors.Is(err, port.ErrUnavailable) }

func isNotInstalled(err error) bool {
	var unavailable *port.UnavailableError
	return errors.As(err, &unavailable) && unavailable.Reason == port.PluginNotInstalled && isUnavailable(err)
}

func TestEveryBridgeAnswersAPluginThatIsDownWithAnErrorAndNoValue(t *testing.T) {
	for _, tc := range bridgeContracts {
		t.Run(tc.name, func(t *testing.T) {
			host := newLeaseHost(t)
			host.ensureErr = errors.New("the plugin is down")
			for _, problem := range listRefusalProblems(t.Context(), host, tc.open, isUnavailable) {
				t.Error(problem)
			}
			if host.acquired.Load() != host.released.Load() {
				t.Errorf("acquired %d references, gave back %d", host.acquired.Load(), host.released.Load())
			}
		})
	}
}

func TestEveryBridgeAnswersAPluginThatIsNotInstalledWithUnavailable(t *testing.T) {
	for _, tc := range bridgeContracts {
		t.Run(tc.name, func(t *testing.T) {
			for _, problem := range listRefusalProblems(t.Context(), NewHost("desktop.editor"), tc.open, isNotInstalled) {
				t.Error(problem)
			}
		})
	}
}

type brokenBridge struct{}

func (brokenBridge) Close() error { return nil }

// The check refuses each way a bridge can answer a plugin that cannot answer
// wrongly, and passes the way it should.
func TestTheRefusalCheckRefusesABridgeThatAnswersWrongly(t *testing.T) {
	cases := []struct {
		name string
		open func(context.Context, bridgeHost) (io.Closer, bridgeCall, error)
		want int
	}{
		{"a refusal with no value", func(context.Context, bridgeHost) (io.Closer, bridgeCall, error) {
			return nil, nil, &port.UnavailableError{Reason: port.PluginNotInstalled}
		}, 0},
		{"a call that fails as it should", func(context.Context, bridgeHost) (io.Closer, bridgeCall, error) {
			return brokenBridge{}, func(context.Context) (bool, error) {
				return true, &port.UnavailableError{Reason: port.PluginNotInstalled}
			}, nil
		}, 0},
		{"a bridge returned with the error", func(context.Context, bridgeHost) (io.Closer, bridgeCall, error) {
			return brokenBridge{}, nil, &port.UnavailableError{Reason: port.PluginNotInstalled}
		}, 1},
		{"a call that succeeds", func(context.Context, bridgeHost) (io.Closer, bridgeCall, error) {
			return brokenBridge{}, func(context.Context) (bool, error) { return false, nil }, nil
		}, 1},
		{"a value with the error", func(context.Context, bridgeHost) (io.Closer, bridgeCall, error) {
			return brokenBridge{}, func(context.Context) (bool, error) {
				return false, &port.UnavailableError{Reason: port.PluginNotInstalled}
			}, nil
		}, 1},
		{"an error that is not unavailable", func(context.Context, bridgeHost) (io.Closer, bridgeCall, error) {
			return brokenBridge{}, func(context.Context) (bool, error) { return true, errors.New("boom") }, nil
		}, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := listRefusalProblems(t.Context(), nil, tc.open, isNotInstalled)
			if len(got) != tc.want {
				t.Errorf("%d problems found, want %d: %v", len(got), tc.want, got)
			}
		})
	}
}

func TestADoubleCloseGivesTheReferenceBackOnce(t *testing.T) {
	for _, tc := range bridgeContracts {
		t.Run(tc.name, func(t *testing.T) {
			host := newLeaseHost(t)
			bridge, _, err := tc.open(t.Context(), host)
			if err != nil {
				t.Fatal(err)
			}
			held := host.acquired.Load()
			if got := host.released.Load(); got != 0 {
				t.Fatalf("%d references were given back before Close", got)
			}
			for range 2 {
				if err := bridge.Close(); err != nil {
					t.Fatal(err)
				}
			}
			if got := host.released.Load(); got != held {
				t.Errorf("held %d references, gave back %d", held, got)
			}
		})
	}
}
