package relevance

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
)

// ErrNotStarted is a call that reached the calibration before Start.
var ErrNotStarted = errors.New("the calibration is not started")

var errIncompleteHost = errors.New("search relevance needs the search index, the store and the queue of the host")

// Hosts is what the plugin works through the host.
type Hosts struct {
	Floors Host
	Queue  host.Queue
}

// calibratingTask is the task the list shows when the measurement of a vault failed.
const calibratingTask = "calibrating the search"

// Session measures vaults and tells the host when a measurement failed.
type Session struct {
	mu     sync.Mutex
	floors *Floors
	queue  host.Queue
}

// NewSession returns a session that ignores events until Start.
func NewSession() *Session {
	return &Session{mu: sync.Mutex{}, floors: nil, queue: nil}
}

// StartOver validates the hosts and starts the calibration over them with the clock.
func (c *Session) StartOver(hosts Hosts, clk Clock) error {
	if hosts.Floors.Search == nil || hosts.Floors.Vaults == nil || hosts.Floors.Store == nil || hosts.Queue == nil {
		return errIncompleteHost
	}
	c.Start(New(hosts.Floors, clk), hosts.Queue)
	return nil
}

// Start gives the calibration the floors it measures and the queue it reports to.
func (c *Session) Start(floors *Floors, queue host.Queue) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.floors = floors
	c.queue = queue
}

// GetFloors returns the floors, or ErrNotStarted before Start.
func (c *Session) GetFloors() (*Floors, error) {
	floors, _ := c.getParts()
	if floors == nil {
		return nil, ErrNotStarted
	}
	return floors, nil
}

// Get returns the floor in force for the shown vault.
func (c *Session) Get(ctx context.Context) (State, error) {
	floors, err := c.GetFloors()
	if err != nil {
		return State{}, err
	}
	return floors.Get(ctx, "")
}

// Calibrate measures the shown vault and keeps the result.
func (c *Session) Calibrate(ctx context.Context) (Outcome, State, error) {
	floors, err := c.GetFloors()
	if err != nil {
		return 0, State{}, err
	}
	return floors.Calibrate(ctx, "")
}

// Set puts the floor of the person in force for the shown vault, or returns to the automatic floor.
func (c *Session) Set(ctx context.Context, isManual bool, floor float64) (State, error) {
	floors, err := c.GetFloors()
	if err != nil {
		return State{}, err
	}
	return floors.Set(ctx, "", isManual, floor)
}

// Probe embeds the query and measures the nearest passage of the shown vault.
func (c *Session) Probe(ctx context.Context, query string) (ProbeResult, error) {
	floors, err := c.GetFloors()
	if err != nil {
		return ProbeResult{}, err
	}
	return floors.Probe(ctx, "", query)
}

func (c *Session) getParts() (*Floors, host.Queue) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.floors, c.queue
}

// HandleEvent measures the vault when the host reports that its vectors exist, and ignores any other
// event. A vault that is measured, or whose floor a person set, is not measured again: an event the host
// repeats changes nothing.
func (c *Session) HandleEvent(ctx context.Context, event string) error {
	floors, queue := c.getParts()
	vault, isSourceReady := host.ReadSearchSourceReadyEvent(event)
	if floors == nil || !isSourceReady {
		return nil
	}
	if err := floors.Ensure(ctx, vault); err != nil {
		return c.reportFailure(ctx, queue, err)
	}
	if err := queue.FinishTask(ctx, calibratingTask); err != nil {
		return fmt.Errorf("finish the task of the calibration: %w", err)
	}
	return nil
}

func (c *Session) reportFailure(ctx context.Context, queue host.Queue, failure error) error {
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
