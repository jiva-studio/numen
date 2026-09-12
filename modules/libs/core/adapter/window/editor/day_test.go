package editor_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/window/editor"
	window "github.com/jiva-studio/numen/modules/libs/core/adapter/window/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
)

// One in the morning, with a day that begins at four: the calendar says the
// fifth and the day of review that began on the fourth is still running.
var morning = time.Date(2026, 9, 5, 1, 0, 0, 0, time.Local)

// none is an installation holding no vaults, which is all the review window
// needs to name the day its counts stand in.
type none struct{}

func (none) All() ([]domain.Vault, error)            { return nil, nil }
func (none) Save(domain.Vault) error                 { return nil }
func (none) Find(string) (domain.Vault, bool, error) { return domain.Vault{}, false, nil }
func (none) Remove(domain.VaultID) error             { return nil }
func (none) Opened(domain.VaultID) error             { return nil }
func (none) Last() (domain.Vault, bool, error)       { return domain.Vault{}, false, nil }

// The editor and the review window are told the same day for the same clock.
// Both windows count in review days, and a day worked out twice is two
// arithmetics that drift the moment one of them is touched.
func TestBothWindowsAreToldTheSameDay(t *testing.T) {
	day := review.Day{Starts: review.DayStarts}
	clock := func() time.Time { return morning }

	editing := &editor.API{Configuring: editor.SettingsPorts{
		Configured: func() (string, string, error) { return "{}", "", nil },
		Day:        day,
		Now:        clock,
	}}
	said, err := editing.GetSettings(t.Context(), connect.NewRequest(&v1.GetSettingsRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	reviewing := newFlashcardsClient(t, &window.API{Registry: none{}, Day: day, Now: clock})
	stream, err := reviewing.WatchCardsDue(t.Context(), connect.NewRequest(&v1.WatchCardsDueRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stream.Close() })
	if !stream.Receive() {
		t.Fatalf("the review window counted nothing: %v", stream.Err())
	}

	if said.Msg.GetDay() != stream.Msg().GetDay() {
		t.Errorf("the editor is told %q and the review window %q",
			said.Msg.GetDay(), stream.Msg().GetDay())
	}
	if said.Msg.GetDay() != "2026-09-04" {
		t.Errorf("an hour past midnight is the day %q", said.Msg.GetDay())
	}
}

// newFlashcardsClient is the review window's cards, asked the way that window asks.
func newFlashcardsClient(t *testing.T, api *window.API) numenv1connect.FlashcardsServiceClient {
	t.Helper()
	path, handler := numenv1connect.NewFlashcardsServiceHandler(api)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.CloseClientConnections)
	t.Cleanup(server.Close)
	return numenv1connect.NewFlashcardsServiceClient(server.Client(), server.URL)
}
