package logs_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"

	windowlogs "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/logs"
	domainlogs "github.com/jiva-studio/numen/libs/core/kernel/domain/logs"
	usecaselogs "github.com/jiva-studio/numen/libs/core/kernel/usecase/logs"
)

type fixedClock struct{}

func (fixedClock) Now() time.Time { return time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC) }

func TestWriteLogAndWatchLogsStreamEntriesOverConnect(t *testing.T) {
	entries := usecaselogs.NewEntries()
	record := usecaselogs.NewRecord(entries, nil, func() domainlogs.Level { return domainlogs.LevelDebug }, fixedClock{})
	watch := usecaselogs.NewWatch(entries, nil, fixedClock{})
	handler := windowlogs.NewHandler(watch, record, func() domainlogs.Level { return domainlogs.LevelDebug }, func(domainlogs.Level) error { return nil })

	mux := http.NewServeMux()
	path, served, _ := windowlogs.NewRoute(handler, connect.WithInterceptors())
	mux.Handle(path, served)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	client := numenv1connect.NewLogServiceClient(srv.Client(), srv.URL)
	if _, err := client.WriteLog(t.Context(), connect.NewRequest(&v1.WriteLogRequest{
		Level:          v1.LogLevel_LOG_LEVEL_INFO,
		Source:         "window",
		Message:        "opened tab",
		PropertiesJson: map[string]string{"kind": `"logs"`},
	})); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	stream, err := client.WatchLogs(ctx, connect.NewRequest(&v1.WatchLogsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := stream.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	}()

	if !stream.Receive() {
		t.Fatalf("expected initial batch: %v", stream.Err())
	}
	got := stream.Msg().GetEntries()
	if len(got) != 1 || got[0].GetMessage() != "opened tab" || got[0].GetPropertiesJson()["kind"] != `"logs"` {
		t.Errorf("got = %+v", got)
	}
}

func TestLogLevelReadsAndWritesApplicationSettingOverConnect(t *testing.T) {
	level := domainlogs.LevelInfo
	handler := windowlogs.NewHandler(nil, nil, func() domainlogs.Level { return level }, func(next domainlogs.Level) error {
		level = next
		return nil
	})

	mux := http.NewServeMux()
	path, served, _ := windowlogs.NewRoute(handler, connect.WithInterceptors())
	mux.Handle(path, served)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	client := numenv1connect.NewLogServiceClient(srv.Client(), srv.URL)
	got, err := client.GetLogLevel(t.Context(), connect.NewRequest(&v1.GetLogLevelRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if got.Msg.GetLevel() != v1.LogLevel_LOG_LEVEL_INFO {
		t.Errorf("level = %v, want %v", got.Msg.GetLevel(), v1.LogLevel_LOG_LEVEL_INFO)
	}

	if _, err := client.SetLogLevel(t.Context(), connect.NewRequest(&v1.SetLogLevelRequest{Level: v1.LogLevel_LOG_LEVEL_DEBUG})); err != nil {
		t.Fatal(err)
	}
	if level != domainlogs.LevelDebug {
		t.Errorf("level = %v, want %v", level, domainlogs.LevelDebug)
	}
}
