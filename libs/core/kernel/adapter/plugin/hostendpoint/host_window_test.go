package hostendpoint

import (
	"context"
	"errors"
	"fmt"
	windowdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"io/fs"
	"math"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/reflect/protoreflect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
)

// recordingWindow is a window that records the places it receives and returns fixed values.
type recordingWindow struct {
	mu        sync.Mutex
	notes     []string
	passages  []domain.Place
	reads     int
	activated []string
	err       error
	open      *windowdomain.OpenTabs
}

func (w *recordingWindow) FocusNote(_ context.Context, path string) (domain.NoteRef, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.err != nil {
		return domain.NoteRef{}, w.err
	}
	w.notes = append(w.notes, path)
	return domain.NoteRef{Path: path, Title: "Gray Heron", ID: "01J8HERON0000000000000000"}, nil
}

func (w *recordingWindow) FocusFile(_ context.Context, path string, passages []domain.ByteRange) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.err != nil {
		return w.err
	}
	w.passages = append(w.passages, domain.Place{Path: path, Ranges: passages})
	return nil
}

func (w *recordingWindow) ReadOpenTabs(context.Context) (windowdomain.OpenTabs, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.reads++
	if w.err != nil {
		return windowdomain.OpenTabs{}, w.err
	}
	if w.open != nil {
		return *w.open, nil
	}
	return windowdomain.OpenTabs{
		ActiveID: "tab-2",
		Tabs: []windowdomain.Tab{
			{ID: "tab-1", Kind: "note", Path: "notes/Heron.md", Title: "Gray Heron"},
			{ID: "tab-2", Kind: "reader:reader-pdf", Path: "library/Birds.pdf", Title: "Birds", State: `{"page":4,"zoom":2}`, Place: &windowdomain.TabPlace{Page: 4, PageCount: 80}},
		},
	}, nil
}

// ActivateTab records the id and returns the fixed open tabs, whose active id stays tab-2. An id none of
// them has is fs.ErrNotExist.
func (w *recordingWindow) ActivateTab(ctx context.Context, id string) (windowdomain.OpenTabs, error) {
	open, err := w.ReadOpenTabs(ctx)
	if err != nil {
		return windowdomain.OpenTabs{}, err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, tab := range open.Tabs {
		if tab.ID == id {
			w.activated = append(w.activated, id)
			return open, nil
		}
	}
	return windowdomain.OpenTabs{}, fmt.Errorf("no open tab has id %q: %w", id, fs.ErrNotExist)
}

func (w *recordingWindow) getActivatedIDs() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.activated)
}

func (w *recordingWindow) getRequests() ([]string, []domain.Place, int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.notes), slices.Clone(w.passages), w.reads
}

func newWindowClient(t *testing.T, window WindowControl, token string) (numenv1connect.HostWindowServiceClient, *callers) {
	t.Helper()
	fakeCallers := newCallers()
	fakeCallers.window = window
	endpoint, err := Open(t.Context(), Address, fakeCallers.asCallers())
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
		got.Msg.GetTitle() != "Gray Heron" || got.Msg.GetId() != "01J8HERON0000000000000000" {
		t.Errorf("returned %v", got.Msg)
	}
	if notes, _, _ := window.getRequests(); !slices.Equal(notes, []string{"notes/Heron.md"}) {
		t.Errorf("the window received a call to focus %v", notes)
	}
	if !slices.Equal(fakeCallers.checked, []string{"md.numen.a window:focus"}) {
		t.Errorf("checked %v, want window:focus checked once", fakeCallers.checked)
	}
}

func TestHostWindowFocusesTheSpansInTheirOrder(t *testing.T) {
	window := &recordingWindow{}
	client, fakeCallers := newWindowClient(t, window, "token-a")

	got, err := client.FocusFile(t.Context(), connect.NewRequest(&v1.FocusFileRequest{
		Path:     "library/Birds.pdf",
		Spans: []*v1.ByteSpan{{Start: 1500, Length: 80}, {Start: 400, Length: 700}},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if got.Msg.GetError() != v1.HostWindowErrorCode_HOST_WINDOW_ERROR_CODE_UNSPECIFIED {
		t.Errorf("returned %v", got.Msg)
	}
	want := []domain.Place{{Path: "library/Birds.pdf", Ranges: []domain.ByteRange{{Start: 1500, End: 1580}, {Start: 400, End: 1100}}}}
	if _, places, _ := window.getRequests(); !reflect.DeepEqual(places, want) {
		t.Errorf("the window received %+v, want %+v", places, want)
	}
	if !slices.Equal(fakeCallers.checked, []string{"md.numen.a window:focus"}) {
		t.Errorf("checked %v, want window:focus checked once", fakeCallers.checked)
	}
}

func TestHostWindowOpensAFileWithNoSpansAtItsBeginning(t *testing.T) {
	window := &recordingWindow{}
	client, _ := newWindowClient(t, window, "token-a")

	if _, err := client.FocusFile(t.Context(), connect.NewRequest(&v1.FocusFileRequest{Path: "library/Birds.pdf"})); err != nil {
		t.Fatal(err)
	}
	_, places, _ := window.getRequests()
	if len(places) != 1 || places[0].Path != "library/Birds.pdf" || len(places[0].Ranges) != 0 {
		t.Errorf("the window received %+v, want the file with no passages", places)
	}
}

func TestHostWindowReturnsOpenTabsWithPlacesAndNoState(t *testing.T) {
	window := &recordingWindow{}
	client, fakeCallers := newWindowClient(t, window, "token-a")

	got, err := client.ReadOpenTabs(t.Context(), connect.NewRequest(&v1.HostWindowServiceReadOpenTabsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if got.Msg.GetError() != v1.HostWindowErrorCode_HOST_WINDOW_ERROR_CODE_UNSPECIFIED || got.Msg.GetActiveId() != "tab-2" {
		t.Fatalf("returned %v", got.Msg)
	}
	tabs := got.Msg.GetTabs()
	if len(tabs) != 2 {
		t.Fatalf("returned %d tabs, want 2: %v", len(tabs), tabs)
	}
	if tabs[0].GetId() != "tab-1" || tabs[0].GetKind() != "note" || tabs[0].GetPath() != "notes/Heron.md" || tabs[0].GetTitle() != "Gray Heron" {
		t.Errorf("the first tab is %v", tabs[0])
	}
	if tabs[1].GetId() != "tab-2" || tabs[1].GetKind() != "reader:reader-pdf" || tabs[1].GetPath() != "library/Birds.pdf" ||
		tabs[1].GetTitle() != "Birds" || tabs[1].GetPlace().GetPage() != 4 || tabs[1].GetPlace().GetPageCount() != 80 {
		t.Errorf("the second tab is %v, want page 4 of 80", tabs[1])
	}
	if tabs[0].GetPlace() != nil {
		t.Errorf("the note tab has the place %v, want none", tabs[0].GetPlace())
	}
	requireOpenTabMessage(t, (&v1.HostWindowServiceReadOpenTabsResponse{}).ProtoReflect().Descriptor(), "tabs")
	if !slices.Equal(fakeCallers.checked, []string{"md.numen.a window:focus"}) {
		t.Errorf("checked %v, want window:focus checked once", fakeCallers.checked)
	}
}

func TestHostWindowRejectsEveryCallWithoutThePermission(t *testing.T) {
	window := &recordingWindow{}
	client, _ := newWindowClient(t, window, "token-b")
	ctx := t.Context()

	_, noteErr := client.FocusNote(ctx, connect.NewRequest(&v1.FocusNoteRequest{Path: "notes/Heron.md"}))
	_, fileErr := client.FocusFile(ctx, connect.NewRequest(&v1.FocusFileRequest{Path: "library/Birds.pdf"}))
	_, tabsErr := client.ReadOpenTabs(ctx, connect.NewRequest(&v1.HostWindowServiceReadOpenTabsRequest{}))
	for name, err := range map[string]error{"FocusNote": noteErr, "FocusFile": fileErr, "ReadOpenTabs": tabsErr} {
		if connect.CodeOf(err) != connect.CodePermissionDenied {
			t.Errorf("%s from a plugin without window:focus got %v, want permission denied", name, err)
		}
	}
	if notes, places, reads := window.getRequests(); len(notes) != 0 || len(places) != 0 || reads != 0 {
		t.Errorf("the window received %v, %v and %d reads for calls that were rejected", notes, places, reads)
	}
}

func TestHostWindowWithoutAWindowReturnsNoWindow(t *testing.T) {
	for name, window := range map[string]WindowControl{
		"no window bound":                 nil,
		"a window that reports no window": &recordingWindow{err: fmt.Errorf("focus: %w", plugin.ErrNoWindow)},
	} {
		t.Run(name, func(t *testing.T) {
			client, _ := newWindowClient(t, window, "token-a")
			ctx := t.Context()
			const noWindow = v1.HostWindowErrorCode_HOST_WINDOW_ERROR_CODE_NO_WINDOW

			if got := readWindowCode(client.FocusNote(ctx, connect.NewRequest(&v1.FocusNoteRequest{Path: "notes/Heron.md"}))); got != noWindow.String() {
				t.Errorf("FocusNote returned %s, want no window", got)
			}
			if got := readWindowCode(client.FocusFile(ctx, connect.NewRequest(&v1.FocusFileRequest{Path: "library/Birds.pdf"}))); got != noWindow.String() {
				t.Errorf("FocusFile returned %s, want no window", got)
			}
			if got := readWindowCode(client.ReadOpenTabs(ctx, connect.NewRequest(&v1.HostWindowServiceReadOpenTabsRequest{}))); got != noWindow.String() {
				t.Errorf("ReadOpenTabs returned %s, want no window", got)
			}
		})
	}
}

func TestHostWindowReturnsNotFoundForPathNotInVault(t *testing.T) {
	window := &recordingWindow{err: fmt.Errorf("this vault contains nothing at library/Gone.pdf: %w", fs.ErrNotExist)}
	client, _ := newWindowClient(t, window, "token-a")
	ctx := t.Context()
	const notFound = v1.HostWindowErrorCode_HOST_WINDOW_ERROR_CODE_NOT_FOUND

	if got := readWindowCode(client.FocusNote(ctx, connect.NewRequest(&v1.FocusNoteRequest{Path: "notes/Gone.md"}))); got != notFound.String() {
		t.Errorf("FocusNote returned %s, want not found", got)
	}
	if got := readWindowCode(client.FocusFile(ctx, connect.NewRequest(&v1.FocusFileRequest{Path: "library/Gone.pdf"}))); got != notFound.String() {
		t.Errorf("FocusFile returned %s, want not found", got)
	}
}

func TestHostWindowReturnsUseCaseRejectionAsInvalidArgument(t *testing.T) {
	window := &recordingWindow{err: fmt.Errorf("highlight at most 8 places of one document: %w", plugin.ErrInvalidFocus)}
	client, _ := newWindowClient(t, window, "token-a")
	ctx := t.Context()

	_, noteErr := client.FocusNote(ctx, connect.NewRequest(&v1.FocusNoteRequest{}))
	_, fileErr := client.FocusFile(ctx, connect.NewRequest(&v1.FocusFileRequest{
		Path: "library/Birds.pdf", Spans: []*v1.ByteSpan{{Start: -1, Length: 4}},
	}))
	for name, err := range map[string]error{"FocusNote": noteErr, "FocusFile": fileErr} {
		if connect.CodeOf(err) != connect.CodeInvalidArgument || !strings.Contains(err.Error(), "highlight at most 8 places") {
			t.Errorf("%s got %v, want invalid argument with the use case's message", name, err)
		}
	}
}

func TestHostWindowReportsAnyOtherFailureAsInternal(t *testing.T) {
	window := &recordingWindow{err: errors.New("the window stopped answering")}
	client, _ := newWindowClient(t, window, "token-a")

	_, err := client.FocusFile(t.Context(), connect.NewRequest(&v1.FocusFileRequest{Path: "library/Birds.pdf"}))
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Errorf("a failing window gave %v, want internal", err)
	}
}

func readWindowCode[T any, M interface {
	*T
	GetError() v1.HostWindowErrorCode
}](res *connect.Response[T], err error) string {
	if err != nil {
		return err.Error()
	}
	return M(res.Msg).GetError().String()
}

func TestHostWindowReturnsEveryPlaceFieldEachReaderReported(t *testing.T) {
	atStart, pastAnHour := 0.0, 3723.9
	window := &recordingWindow{open: &windowdomain.OpenTabs{ActiveID: "tab-1", Tabs: []windowdomain.Tab{
		{ID: "tab-1", Kind: "recording", Path: "talks/Heron Talk.mp3", Title: "Heron Talk", Place: &windowdomain.TabPlace{PositionSeconds: &atStart}},
		{ID: "tab-2", Kind: "recording", Path: "talks/Marsh Talk.mp3", Title: "Marsh Talk", Place: &windowdomain.TabPlace{PositionSeconds: &pastAnHour}},
		{ID: "tab-3", Kind: "reader:reader-epub", Path: "library/Marsh.epub", Title: "Marsh", Place: &windowdomain.TabPlace{CFI: "epubcfi(/6/4!/4/2/1:0)", Chapter: "The Reed Beds"}},
		{ID: "tab-4", Kind: "recording", Path: "talks/Bees.mp3", Title: "Bees", Place: &windowdomain.TabPlace{}},
	}}}
	client, _ := newWindowClient(t, window, "token-a")

	got, err := client.ReadOpenTabs(t.Context(), connect.NewRequest(&v1.HostWindowServiceReadOpenTabsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	tabs := got.Msg.GetTabs()
	if len(tabs) != 4 {
		t.Fatalf("returned %d tabs, want 4: %v", len(tabs), tabs)
	}
	if place := tabs[0].GetPlace(); place == nil || place.PositionSeconds == nil || place.GetPositionSeconds() != 0 {
		t.Errorf("the recording at its start is at %v, want 0 seconds", place)
	}
	if place := tabs[1].GetPlace(); place == nil || place.PositionSeconds == nil || place.GetPositionSeconds() != 3723.9 {
		t.Errorf("the recording past an hour is at %v, want 3723.9 seconds", place)
	}
	if place := tabs[2].GetPlace(); place == nil || place.GetCfi() != "epubcfi(/6/4!/4/2/1:0)" || place.GetChapter() != "The Reed Beds" || place.GetPage() != 0 {
		t.Errorf("the book is at %v, want its CFI and chapter", place)
	}
	if place := tabs[3].GetPlace(); place == nil || place.PositionSeconds != nil {
		t.Errorf("a recording that reported no time is at %v, want a place with no time", place)
	}
}

func TestHostWindowRejectsSpanPast32BitsBeforeCallingWindow(t *testing.T) {
	window := &recordingWindow{}
	client, _ := newWindowClient(t, window, "token-a")

	_, err := client.FocusFile(t.Context(), connect.NewRequest(&v1.FocusFileRequest{
		Path:     "library/Birds.pdf",
		Spans: []*v1.ByteSpan{{Start: 400, Length: 80}, {Start: math.MaxInt32 - 10, Length: 20}},
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("a passage past the 32-bit range got %v, want invalid argument", err)
	}
	if _, places, _ := window.getRequests(); len(places) != 0 {
		t.Errorf("the window received %+v for a passage past the 32-bit range", places)
	}
}

func TestHostWindowFocusesASpanThatEndsAtThe32BitLimit(t *testing.T) {
	window := &recordingWindow{}
	client, _ := newWindowClient(t, window, "token-a")

	if _, err := client.FocusFile(t.Context(), connect.NewRequest(&v1.FocusFileRequest{
		Path:     "library/Birds.pdf",
		Spans: []*v1.ByteSpan{{Start: math.MaxInt32 - 10, Length: 10}},
	})); err != nil {
		t.Fatal(err)
	}
	want := []domain.Place{{Path: "library/Birds.pdf", Ranges: []domain.ByteRange{{Start: math.MaxInt32 - 10, End: math.MaxInt32}}}}
	if _, places, _ := window.getRequests(); !reflect.DeepEqual(places, want) {
		t.Errorf("the window received %+v, want %+v", places, want)
	}
}

func requireOpenTabMessage(t *testing.T, message protoreflect.MessageDescriptor, field protoreflect.Name) {
	t.Helper()
	tab := message.Fields().ByName(field).Message()
	if want := (&v1.OpenTab{}).ProtoReflect().Descriptor().FullName(); tab == nil || tab.FullName() != want {
		t.Fatalf("the field %s of %s is not a %s", field, message.FullName(), want)
	}
	if state := tab.Fields().ByName("state"); state != nil {
		t.Errorf("the tab %s a plugin reads has the field state", tab.FullName())
	}
}
