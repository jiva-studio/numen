// Package service is the backend of the todos plugin: it serves the tasks of the notes to the
// plugin's frontend.
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/todos/backend/api"
	"github.com/jiva-studio/numen/plugins/todos/backend/internal/adapter/clock"
	hostadapter "github.com/jiva-studio/numen/plugins/todos/backend/internal/adapter/host"
	logadapter "github.com/jiva-studio/numen/plugins/todos/backend/internal/adapter/log"
	"github.com/jiva-studio/numen/plugins/todos/backend/internal/adapter/snapshot"
	"github.com/jiva-studio/numen/plugins/todos/backend/ports"
	"github.com/jiva-studio/numen/plugins/todos/backend/usecase/complete"
	"github.com/jiva-studio/numen/plugins/todos/backend/usecase/edit"
	"github.com/jiva-studio/numen/plugins/todos/backend/usecase/index"
	"github.com/jiva-studio/numen/plugins/todos/backend/wire/queryfns"
	"github.com/jiva-studio/numen/plugins/todos/backend/wire/tasktools"
	"github.com/jiva-studio/numen/plugins/todos/backend/wire/todos"
)

const (
	defaultLimit = 500
	maxLimit     = 2000
	ignoredKey   = "ignoredFolders"
	flushTimeout = 5 * time.Second
)

func getSummary(tasks *index.Index) func(context.Context, api.GetSummaryRequest) (api.Summary, error) {
	return func(_ context.Context, req api.GetSummaryRequest) (api.Summary, error) {
		summary := tasks.Summarize(req.Today)
		return api.Summary{
			IsReady:            summary.IsReady,
			OverdueCount:       summary.OverdueCount,
			TodayCount:         summary.TodayCount,
			NextSevenDaysCount: summary.NextSevenDaysCount,
			NoDateCount:        summary.NoDateCount,
			UnreadableCount:    summary.UnreadableCount,
		}, nil
	}
}

func listTasks(tasks *index.Index) func(context.Context, api.ListTasksRequest) (api.ListTasksResponse, error) {
	return func(_ context.Context, req api.ListTasksRequest) (api.ListTasksResponse, error) {
		listing := tasks.List(req.Today, todos.DecodeFilter(req.Filter), todos.ClampLimit(req.Limit, defaultLimit, maxLimit))
		encoded := make([]api.Task, 0, len(listing.Tasks))
		for _, task := range listing.Tasks {
			encoded = append(encoded, todos.EncodeTask(task))
		}
		return api.ListTasksResponse{Tasks: encoded, Total: listing.Total, Revision: listing.Revision}, nil
	}
}

func listFields(tasks *index.Index) func(context.Context, api.ListFieldsRequest) (api.ListFieldsResponse, error) {
	return func(_ context.Context, _ api.ListFieldsRequest) (api.ListFieldsResponse, error) {
		return todos.EncodeFields(tasks.ListFields()), nil
	}
}

func refreshIndex(tasks *index.Index) func(context.Context, api.RefreshIndexRequest) (api.RefreshIndexResponse, error) {
	return func(ctx context.Context, req api.RefreshIndexRequest) (api.RefreshIndexResponse, error) {
		renames := make([]index.Rename, 0, len(req.Renamed))
		for _, rename := range req.Renamed {
			renames = append(renames, index.Rename{From: rename.From, To: rename.To})
		}
		if err := tasks.Refresh(ctx, req.Paths, renames); err != nil {
			return api.RefreshIndexResponse{}, convertError("refresh the index", err)
		}
		return api.RefreshIndexResponse{}, nil
	}
}

func watchTasks(tasks *index.Index) func(context.Context, api.WatchTasksRequest, func(api.WatchTasksMessage) error) error {
	return func(ctx context.Context, _ api.WatchTasksRequest, send func(api.WatchTasksMessage) error) error {
		return tasks.Watch(ctx, func(state index.State) error {
			return send(api.WatchTasksMessage{Revision: state.Revision, IsReady: state.IsReady})
		})
	}
}

func convertError(action string, err error) error {
	switch {
	case errors.Is(err, host.ErrNoVault), errors.Is(err, host.ErrUnknownVault):
		return server.NewAPIError(server.CodeFailedPrecondition, err.Error())
	case errors.Is(err, host.ErrPermissionDenied):
		return server.NewAPIError(server.CodePermissionDenied, err.Error())
	}
	return fmt.Errorf("%s: %w", action, err)
}

func getClosedContext(s *server.Server) context.Context {
	ctx, stop := context.WithCancel(context.Background())
	s.OnClose(func() error {
		stop()
		return nil
	})
	return ctx
}

func setTaskDone(done *complete.SetTaskDone) func(context.Context, api.SetTaskDoneRequest) (api.SetTaskDoneResponse, error) {
	return func(ctx context.Context, req api.SetTaskDoneRequest) (api.SetTaskDoneResponse, error) {
		result, err := done.Execute(ctx, complete.Request{
			Path: req.Path, Line: req.Line, IsProseLine: req.IsProseLine, Raw: req.Raw, IsDone: req.IsDone, Today: req.Today,
		})
		if errors.Is(err, complete.ErrInvalidToday) {
			return api.SetTaskDoneResponse{}, server.NewAPIError(server.CodeInvalidArgument, err.Error())
		}
		if err != nil {
			return api.SetTaskDoneResponse{}, convertError("change the task", err)
		}
		return todos.EncodeResult(result), nil
	}
}

func mountAgentSurface(s *server.Server, vaults *Vaults, today ports.Clock) error {
	functions, err := queryfns.New(vaults, today)
	if err != nil {
		return err
	}
	registry, err := tasktools.New(vaults, today).BuildRegistry()
	if err != nil {
		return err
	}
	functions.Mount(s)
	registry.Mount(s)
	return nil
}

func mount(ctx context.Context, s *server.Server, tasks *index.Index, done *complete.SetTaskDone, log ports.Log) error {
	s.SetUpdateConfig(func(values map[string]string) {
		encoded, isSet := values[ignoredKey]
		if !isSet {
			return
		}
		var folders []string
		if err := json.Unmarshal([]byte(encoded), &folders); err != nil {
			log.Error("read the ignored folders", err)
			return
		}
		if err := tasks.SetIgnoredFolders(ctx, folders); err != nil {
			log.Error("apply the ignored folders", err)
		}
	})
	a := s.NewAPI(api.Service)
	server.Handle(a, api.MethodGetSummary, getSummary(tasks))
	server.Handle(a, api.MethodListTasks, listTasks(tasks))
	server.Handle(a, api.MethodListFields, listFields(tasks))
	server.Handle(a, api.MethodRefreshIndex, refreshIndex(tasks))
	server.Handle(a, api.MethodSetTaskDone, setTaskDone(done))
	server.Stream(a, api.MethodWatchTasks, watchTasks(tasks))
	return a.Err()
}

func startIndex(ctx context.Context, tasks *index.Index, log ports.Log) {
	err := tasks.Start(ctx)
	if err == nil || ctx.Err() != nil {
		return
	}
	log.Error("start the task index", err)
	if err := tasks.Scan(ctx); err != nil && ctx.Err() == nil {
		log.Error("scan the notes for tasks", err)
	}
}

// Mount registers the service on s and starts reading the notes of the vault the host shows.
func Mount(s *server.Server) error {
	vault, err := hostadapter.NewFromEnv()
	if err != nil {
		return err
	}
	log := logadapter.Slog{}
	tasks := todos.NewIndex(vault, vault, vault, index.Persistence{
		Snapshots: snapshot.New(""), Feed: vault, After: index.StartSystemTimer, Cursors: snapshot.NewCursorStore, Log: log,
	})
	lock := ports.NewWriteLock()
	done := complete.New(vault, vault, tasks, lock)
	vaults := NewVaults(vault.GetVaultID, tasks, tasktools.Scope{Tasks: tasks, Editor: edit.New(vault, vault, tasks, lock), Completer: done})
	ctx := getClosedContext(s)
	if err := mount(ctx, s, tasks, done, log); err != nil {
		return err
	}
	if err := mountAgentSurface(s, vaults, clock.System{}); err != nil {
		return err
	}
	s.OnClose(func() error {
		flushCtx, stop := context.WithTimeout(context.Background(), flushTimeout)
		defer stop()
		return tasks.Flush(flushCtx)
	})
	go startIndex(ctx, tasks, log)
	return nil
}
