// Package service is the backend of the search relevance plugin: the service the settings view
// calls to measure and set the search floor, and the handler of the events that make the plugin
// measure a vault once its vectors exist.
package service

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-search-relevance/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-search-relevance/backend/relevance"
)

// PluginID is the identity the manifest gives the plugin.
const PluginID = "md.numen.tools.search.relevance"

// calibratingTask is the task the list shows when the measurement of a vault failed.
const calibratingTask = "calibrating the search"

var errIncompleteHost = errors.New("search relevance needs the search index, the store and the queue of the host")

// errNotMounted is a call that reached the app before it was mounted.
var errNotMounted = server.NewAPIError(server.CodeUnavailable, "the plugin is not ready")

// Hosts is what the plugin works through the host.
type Hosts struct {
	Floors relevance.Host
	Queue  host.Queue
}

// App is the backend of the plugin. NewApp and NewAppOver make one.
type App struct {
	open  func() (Hosts, error)
	clock relevance.Clock

	mu        sync.Mutex
	floors    *relevance.Floors
	queue     host.Queue
	isChecked bool
}

// NewApp returns the backend that connects to the host the process was launched for when it is mounted.
func NewApp() *App {
	return &App{open: openFromEnv, clock: clock.System{}, mu: sync.Mutex{}, floors: nil, queue: nil, isChecked: false}
}

// NewAppOver returns the backend over the hosts given, with measurements stamped by the clock.
func NewAppOver(hosts Hosts, clk relevance.Clock) *App {
	return &App{
		open:  func() (Hosts, error) { return hosts, nil },
		clock: clk, mu: sync.Mutex{}, floors: nil, queue: nil, isChecked: false,
	}
}

// openFromEnv connects to the host the environment names.
func openFromEnv() (Hosts, error) {
	endpoint, err := hostcall.FromEnv()
	if err != nil {
		return Hosts{}, err
	}
	return Hosts{
		Floors: relevance.Host{Search: host.NewSearchClient(endpoint), Vault: host.NewClient(endpoint), Store: host.NewStoreClient(endpoint)},
		Queue:  host.NewQueueClient(endpoint),
	}, nil
}

// Mount registers the service on the server.
func (a *App) Mount(s *server.Server) error {
	hosts, err := a.open()
	if err != nil {
		return err
	}
	if hosts.Floors.Search == nil || hosts.Floors.Vault == nil || hosts.Floors.Store == nil || hosts.Queue == nil {
		return errIncompleteHost
	}
	a.mu.Lock()
	a.floors = relevance.New(hosts.Floors, a.clock)
	a.queue = hosts.Queue
	a.mu.Unlock()

	service := s.NewAPI(api.Service)
	server.Handle(service, api.MethodGetFloor, a.getFloor)
	server.Handle(service, api.MethodCalibrateFloor, a.calibrateFloor)
	server.Handle(service, api.MethodSetFloor, a.setFloor)
	server.Handle(service, api.MethodProbeFloor, a.probeFloor)
	// gomutants:disable-next-line RETURN_ERROR_NIL reason="Constant service and method names are always valid."
	return service.Err()
}

// getFloors returns the floors and the queue, and nil before the app is mounted.
func (a *App) getFloors() (*relevance.Floors, host.Queue) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.floors, a.queue
}

// HandleEvent measures the vault when the host reports that all of its vectors exist. The first event
// of any kind after the plugin starts checks the vault the host shows as well, so a plugin that starts
// after the vectors were made still measures.
//
// A vault that is measured, or whose floor a person set, is not measured again: an event the host
// repeats changes nothing.
func (a *App) HandleEvent(ctx context.Context, event string) error {
	floors, queue := a.getFloors()
	if floors == nil {
		return nil
	}
	vault, isVectorsReady := host.ReadVectorsReadyEvent(event)
	if !isVectorsReady && a.wasChecked() {
		return nil
	}
	if err := floors.Ensure(ctx, vault); err != nil {
		return a.reportFailure(ctx, queue, err)
	}
	a.markChecked()
	if err := queue.FinishTask(ctx, calibratingTask); err != nil {
		return fmt.Errorf("finish the task of the calibration: %w", err)
	}
	return nil
}

// wasChecked is whether a check of a vault succeeded since the plugin started.
func (a *App) wasChecked() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.isChecked
}

// markChecked records that a check of a vault succeeded.
func (a *App) markChecked() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.isChecked = true
}

// reportFailure lists the failure of a measurement as a task and returns it, joined with the failure
// of listing it. A measurement that was cancelled is no failure to list.
func (a *App) reportFailure(ctx context.Context, queue host.Queue, failure error) error {
	if errors.Is(failure, context.Canceled) {
		return failure
	}
	task := host.Task{
		ID: calibratingTask, Title: "Calibrating the search", Detail: "", CompletedCount: 0, TotalCount: 0,
		Unit: host.UnitThings, FailureReason: failure.Error(), IsUserStarted: false,
	}
	if err := queue.ReportTask(ctx, task); err != nil {
		return errors.Join(failure, fmt.Errorf("report the task of the calibration: %w", err))
	}
	return failure
}
