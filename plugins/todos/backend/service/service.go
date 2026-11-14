// Package service is the backend of the todos plugin: it serves the tasks of the notes to the
// plugin's frontend.
package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/todos/backend/api"
	hostadapter "github.com/jiva-studio/numen/plugins/todos/backend/internal/adapter/host"
	"github.com/jiva-studio/numen/plugins/todos/backend/usecase/index"
	"github.com/jiva-studio/numen/plugins/todos/backend/wire/todos"
)

const (
	defaultLimit = 500
	maxLimit     = 2000
	ignoredKey   = "ignoredFolders"
)

func getSummary(tasks *index.Index) func(context.Context, api.GetSummaryRequest) (api.Summary, error) {
	return func(_ context.Context, req api.GetSummaryRequest) (api.Summary, error) {
		summary := tasks.Summarize(req.Today)
		return api.Summary{
			IsReady:         summary.IsReady,
			OverdueCount:    summary.OverdueCount,
			TodayCount:      summary.TodayCount,
			NoDateCount:     summary.NoDateCount,
			UnreadableCount: summary.UnreadableCount,
		}, nil
	}
}

func listTasks(tasks *index.Index) func(context.Context, api.ListTasksRequest) (api.ListTasksResponse, error) {
	return func(_ context.Context, req api.ListTasksRequest) (api.ListTasksResponse, error) {
		listing := tasks.List(req.Today, todos.DecodeFilter(req.Filter), getLimit(req.Limit))
		encoded := make([]api.Task, 0, len(listing.Tasks))
		for _, task := range listing.Tasks {
			encoded = append(encoded, todos.EncodeTask(task))
		}
		return api.ListTasksResponse{Tasks: encoded, Total: listing.Total, Revision: listing.Revision}, nil
	}
}

func getLimit(asked int) int {
	switch {
	case asked <= 0:
		return defaultLimit
	case asked > maxLimit:
		return maxLimit
	}
	return asked
}

func refreshIndex(tasks *index.Index) func(context.Context, api.RefreshIndexRequest) (api.RefreshIndexResponse, error) {
	return func(ctx context.Context, req api.RefreshIndexRequest) (api.RefreshIndexResponse, error) {
		renames := make([]index.Rename, 0, len(req.Renamed))
		for _, rename := range req.Renamed {
			renames = append(renames, index.Rename{From: rename.From, To: rename.To})
		}
		if err := tasks.Refresh(ctx, req.Paths, renames); err != nil {
			return api.RefreshIndexResponse{}, convertError(err)
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

func convertError(err error) error {
	switch {
	case errors.Is(err, host.ErrNoVault), errors.Is(err, host.ErrUnknownVault):
		return server.NewAPIError(server.CodeFailedPrecondition, err.Error())
	case errors.Is(err, host.ErrPermissionDenied):
		return server.NewAPIError(server.CodePermissionDenied, err.Error())
	}
	return fmt.Errorf("refresh the index: %w", err)
}

func getClosedContext(s *server.Server) context.Context {
	ctx, stop := context.WithCancel(context.Background())
	s.OnClose(func() error {
		stop()
		return nil
	})
	return ctx
}

func ignore(_ error) {}

func mount(s *server.Server, tasks *index.Index) error {
	ctx := getClosedContext(s)
	s.SetUpdateConfig(func(values map[string]string) {
		folders, isSet := values[ignoredKey]
		if !isSet {
			return
		}
		// A failure leaves the folders set; the next full check of the index catches up.
		ignore(tasks.SetIgnoredFolders(ctx, folders))
	})
	a := s.NewAPI(api.Service)
	server.Handle(a, api.MethodGetSummary, getSummary(tasks))
	server.Handle(a, api.MethodListTasks, listTasks(tasks))
	server.Handle(a, api.MethodRefreshIndex, refreshIndex(tasks))
	server.Stream(a, api.MethodWatchTasks, watchTasks(tasks))
	return a.Err()
}

// Mount registers the service on s and starts reading the notes of the vault the host shows.
func Mount(s *server.Server) error {
	vault, err := hostadapter.FromEnv()
	if err != nil {
		return err
	}
	tasks := index.NewForNotes(vault, vault)
	if err := mount(s, tasks); err != nil {
		return err
	}
	ctx := getClosedContext(s)
	// A failed scan leaves the index not ready; the next full check of the index scans again.
	go func() { ignore(tasks.Scan(ctx)) }()
	return nil
}
