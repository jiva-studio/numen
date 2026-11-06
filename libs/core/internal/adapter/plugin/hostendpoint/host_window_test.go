package hostendpoint

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// GetPluginWindow is no window: the fake host of the other tests lends none.
func (c *callers) GetPluginWindow() pluginport.PluginWindow { return nil }

// windowCallers is the fake host that lends its plugins the window.
type windowCallers struct {
	*callers
	window pluginport.PluginWindow
}

func (c windowCallers) GetPluginWindow() pluginport.PluginWindow { return c.window }

// recordingWindow is a window that records what it is asked to show and answers with fixed values.
type recordingWindow struct {
	mu        sync.Mutex
	notes     []string
	passages  []domain.Place
	reads     int
	activated []string
	err       error
}

func (w *recordingWindow) FocusNote(_ context.Context, path string) (domain.NoteRef, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.err != nil {
		return domain.NoteRef{}, w.err
	}
	w.notes = append(w.notes, path)
	return domain.NoteRef{Path: path, Title: "Grey Heron", ID: "01J8HERON0000000000000000"}, nil
}

func (w *recordingWindow) FocusPassages(_ context.Context, path string, passages []domain.ByteRange) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.err != nil {
		return w.err
	}
	w.passages = append(w.passages, domain.Place{Path: path, Ranges: passages})
	return nil
}

func (w *recordingWindow) ReadOpenTabs(context.Context) (domain.OpenTabs, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.reads++
	if w.err != nil {
		return domain.OpenTabs{}, w.err
	}
	return domain.OpenTabs{
		ActiveID: "tab-2",
		Tabs: []domain.Tab{
			{ID: "tab-1", Kind: "note", Path: "notes/Heron.md", Title: "Grey Heron"},
			{ID: "tab-2", Kind: "reader:reader-pdf", Path: "library/Birds.pdf", Title: "Birds", State: `{"page":4}`},
		},
	}, nil
}

// ActivateTab records the id and answers the fixed open tabs, whose active id stays tab-2. An id none of
// them has is fs.ErrNotExist.
func (w *recordingWindow) ActivateTab(ctx context.Context, id string) (domain.OpenTabs, error) {
	open, err := w.ReadOpenTabs(ctx)
	if err != nil {
		return domain.OpenTabs{}, err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, tab := range open.Tabs {
		if tab.ID == id {
			w.activated = append(w.activated, id)
			return open, nil
		}
	}
	return domain.OpenTabs{}, fmt.Errorf("no open tab has id %q: %w", id, fs.ErrNotExist)
}

func (w *recordingWindow) getActivated() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.activated)
}

func (w *recordingWindow) getAsked() ([]string, []domain.Place, int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.notes), slices.Clone(w.passages), w.reads
}

// newWindowClient opens the host endpoint over fake callers that lend the window, nil for none, and
// returns a client calling HostWindowService with the token.
func newWindowClient(t *testing.T, window pluginport.PluginWindow, token string) (numenv1connect.HostWindowServiceClient, *callers) {
	t.Helper()
	fakeCallers := newCallers(nil)
	endpoint, err := Open(t.Context(), Address, windowCallers{callers: fakeCallers, window: window})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := endpoint.Close(context.WithoutCancel(t.Context())); err != nil {
			t.Error(err)
		}
	})
	url := fmt.Sprintf("http://127.0.0.1:%d", endpoint.GetPort())
	return numenv1connect.NewHostWindowServiceClient(http.DefaultClient, url, connect.WithInterceptors(transport.AuthInterceptor(token))), fakeCallers
}

func TestHostWindowFocusesANoteAndChecksThePermission(t *testing.T) {
	window := &recordingWindow{}
	client, fakeCallers := newWindowClient(t, window, "token-a")

	got, err := client.FocusNote(t.Context(), connect.NewRequest(&v1.FocusNoteRequest{Path: "notes/Heron.md"}))
	if err != nil {
		t.Fatal(err)
	}
	if got.Msg.GetError() != v1.HostWindowErrorCode_HOST_WINDOW_ERROR_CODE_UNSPECIFIED || got.Msg.GetPath() != "notes/Heron.md" ||
		got.Msg.GetTitle() != "Grey Heron" || got.Msg.GetId() != "01J8HERON0000000000000000" {
		t.Errorf("answered %v", got.Msg)
	}
	if notes, _, _ := window.getAsked(); !slices.Equal(notes, []string{"notes/Heron.md"}) {
		t.Errorf("the window was asked to focus %v", notes)
	}
	if !slices.Equal(fakeCallers.checked, []string{"md.numen.a window:focus"}) {
		t.Errorf("checked %v, want window:focus asked once", fakeCallers.checked)
	}
}

func TestHostWindowFocusesThePassagesInTheirOrder(t *testing.T) {
	window := &recordingWindow{}
	client, fakeCallers := newWindowClient(t, window, "token-a")

	got, err := client.FocusPassages(t.Context(), connect.NewRequest(&v1.FocusPassagesRequest{
		Path:     "library/Birds.pdf",
		Passages: []*v1.PassageSpan{{Start: 1500, Length: 80}, {Start: 400, Length: 700}},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if got.Msg.GetError() != v1.HostWindowErrorCode_HOST_WINDOW_ERROR_CODE_UNSPECIFIED {
		t.Errorf("answered %v", got.Msg)
	}
	want := []domain.Place{{Path: "library/Birds.pdf", Ranges: []domain.ByteRange{{Start: 1500, End: 1580}, {Start: 400, End: 1100}}}}
	if _, places, _ := window.getAsked(); !reflect.DeepEqual(places, want) {
		t.Errorf("the window was asked %+v, want %+v", places, want)
	}
	if !slices.Equal(fakeCallers.checked, []string{"md.numen.a window:focus"}) {
		t.Errorf("checked %v, want window:focus asked once", fakeCallers.checked)
	}
}

func TestHostWindowOpensAFileWithNoPassagesAtItsBeginning(t *testing.T) {
	window := &recordingWindow{}
	client, _ := newWindowClient(t, window, "token-a")

	if _, err := client.FocusPassages(t.Context(), connect.NewRequest(&v1.FocusPassagesRequest{Path: "library/Birds.pdf"})); err != nil {
		t.Fatal(err)
	}
	_, places, _ := window.getAsked()
	if len(places) != 1 || places[0].Path != "library/Birds.pdf" || len(places[0].Ranges) != 0 {
		t.Errorf("the window was asked %+v, want the file with no passages", places)
	}
}

func TestHostWindowAnswersTheOpenTabsAndTheActiveTab(t *testing.T) {
	window := &recordingWindow{}
	client, fakeCallers := newWindowClient(t, window, "token-a")

	got, err := client.ReadOpenTabs(t.Context(), connect.NewRequest(&v1.HostWindowServiceReadOpenTabsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if got.Msg.GetError() != v1.HostWindowErrorCode_HOST_WINDOW_ERROR_CODE_UNSPECIFIED || got.Msg.GetActiveId() != "tab-2" {
		t.Fatalf("answered %v", got.Msg)
	}
	tabs := got.Msg.GetTabs()
	if len(tabs) != 2 {
		t.Fatalf("answered %d tabs, want 2: %v", len(tabs), tabs)
	}
	if tabs[0].GetId() != "tab-1" || tabs[0].GetKind() != "note" || tabs[0].GetPath() != "notes/Heron.md" || tabs[0].GetTitle() != "Grey Heron" {
		t.Errorf("the first tab is %v", tabs[0])
	}
	if tabs[1].GetId() != "tab-2" || tabs[1].GetKind() != "reader:reader-pdf" || tabs[1].GetPath() != "library/Birds.pdf" ||
		tabs[1].GetTitle() != "Birds" || tabs[1].GetState() != `{"page":4}` {
		t.Errorf("the second tab is %v", tabs[1])
	}
	if !slices.Equal(fakeCallers.checked, []string{"md.numen.a window:focus"}) {
		t.Errorf("checked %v, want window:focus asked once", fakeCallers.checked)
	}
}

func TestHostWindowRefusesEveryCallFromAPluginWithoutThePermission(t *testing.T) {
	window := &recordingWindow{}
	client, _ := newWindowClient(t, window, "token-b")
	ctx := t.Context()

	_, noteErr := client.FocusNote(ctx, connect.NewRequest(&v1.FocusNoteRequest{Path: "notes/Heron.md"}))
	_, passagesErr := client.FocusPassages(ctx, connect.NewRequest(&v1.FocusPassagesRequest{Path: "library/Birds.pdf"}))
	_, tabsErr := client.ReadOpenTabs(ctx, connect.NewRequest(&v1.HostWindowServiceReadOpenTabsRequest{}))
	for name, err := range map[string]error{"FocusNote": noteErr, "FocusPassages": passagesErr, "ReadOpenTabs": tabsErr} {
		if connect.CodeOf(err) != connect.CodePermissionDenied {
			t.Errorf("%s from a plugin without window:focus got %v, want permission denied", name, err)
		}
	}
	if notes, places, reads := window.getAsked(); len(notes) != 0 || len(places) != 0 || reads != 0 {
		t.Errorf("the window was asked %v, %v and %d reads for calls that were refused", notes, places, reads)
	}
}

func TestHostWindowWithoutAWindowAnswersNoWindow(t *testing.T) {
	for name, window := range map[string]pluginport.PluginWindow{
		"no window bound":                 nil,
		"a window that reports no window": &recordingWindow{err: fmt.Errorf("focus: %w", pluginport.ErrNoWindow)},
	} {
		t.Run(name, func(t *testing.T) {
			client, _ := newWindowClient(t, window, "token-a")
			ctx := t.Context()
			const noWindow = v1.HostWindowErrorCode_HOST_WINDOW_ERROR_CODE_NO_WINDOW

			if got := readWindowCode(client.FocusNote(ctx, connect.NewRequest(&v1.FocusNoteRequest{Path: "notes/Heron.md"}))); got != noWindow.String() {
				t.Errorf("FocusNote answered %s, want no window", got)
			}
			if got := readWindowCode(client.FocusPassages(ctx, connect.NewRequest(&v1.FocusPassagesRequest{Path: "library/Birds.pdf"}))); got != noWindow.String() {
				t.Errorf("FocusPassages answered %s, want no window", got)
			}
			if got := readWindowCode(client.ReadOpenTabs(ctx, connect.NewRequest(&v1.HostWindowServiceReadOpenTabsRequest{}))); got != noWindow.String() {
				t.Errorf("ReadOpenTabs answered %s, want no window", got)
			}
		})
	}
}

func TestHostWindowAnswersNotFoundForAPathTheVaultDoesNotHold(t *testing.T) {
	window := &recordingWindow{err: fmt.Errorf("this vault holds nothing at library/Gone.pdf: %w", fs.ErrNotExist)}
	client, _ := newWindowClient(t, window, "token-a")
	ctx := t.Context()
	const notFound = v1.HostWindowErrorCode_HOST_WINDOW_ERROR_CODE_NOT_FOUND

	if got := readWindowCode(client.FocusNote(ctx, connect.NewRequest(&v1.FocusNoteRequest{Path: "notes/Gone.md"}))); got != notFound.String() {
		t.Errorf("FocusNote answered %s, want not found", got)
	}
	if got := readWindowCode(client.FocusPassages(ctx, connect.NewRequest(&v1.FocusPassagesRequest{Path: "library/Gone.pdf"}))); got != notFound.String() {
		t.Errorf("FocusPassages answered %s, want not found", got)
	}
}

func TestHostWindowRejectsAPlaceTheUseCaseRefusesAsAnInvalidArgument(t *testing.T) {
	window := &recordingWindow{err: fmt.Errorf("highlight at most 8 places of one document: %w", pluginport.ErrInvalidFocus)}
	client, _ := newWindowClient(t, window, "token-a")
	ctx := t.Context()

	_, noteErr := client.FocusNote(ctx, connect.NewRequest(&v1.FocusNoteRequest{}))
	_, passagesErr := client.FocusPassages(ctx, connect.NewRequest(&v1.FocusPassagesRequest{
		Path: "library/Birds.pdf", Passages: []*v1.PassageSpan{{Start: -1, Length: 4}},
	}))
	for name, err := range map[string]error{"FocusNote": noteErr, "FocusPassages": passagesErr} {
		if connect.CodeOf(err) != connect.CodeInvalidArgument || !strings.Contains(err.Error(), "highlight at most 8 places") {
			t.Errorf("%s got %v, want invalid argument with the use case's message", name, err)
		}
	}
}

func TestHostWindowReportsAnyOtherFailureAsInternal(t *testing.T) {
	window := &recordingWindow{err: errors.New("the window stopped answering")}
	client, _ := newWindowClient(t, window, "token-a")

	_, err := client.FocusPassages(t.Context(), connect.NewRequest(&v1.FocusPassagesRequest{Path: "library/Birds.pdf"}))
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Errorf("a failing window gave %v, want internal", err)
	}
}

// readWindowCode is the error code a served call answered with, or the status of a call that was
// not served.
func readWindowCode[T any, M interface {
	*T
	GetError() v1.HostWindowErrorCode
}](res *connect.Response[T], err error) string {
	if err != nil {
		return err.Error()
	}
	return M(res.Msg).GetError().String()
}
