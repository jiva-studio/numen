// Package calibration measures a vault once its vectors exist, and keeps the floors and the queue of
// the host that the calls of the settings view and the events of the host share.
package calibration

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/plugins/tools-search-relevance/backend/relevance"
)

// ErrNotStarted is a call that reached the calibration before Start.
var ErrNotStarted = errors.New("the calibration is not started")

// calibratingTask is the task the list shows when the measurement of a vault failed.
const calibratingTask = "calibrating the search"

// Calibration measures vaults and tells the host when a measurement failed.
type Calibration struct {
	mu        sync.Mutex
	floors    *relevance.Floors
	queue     host.Queue
	isChecked bool
}

// New returns a calibration that ignores events until Start.
func New() *Calibration {
	return &Calibration{mu: sync.Mutex{}, floors: nil, queue: nil, isChecked: false}
}

// Start gives the calibration the floors it measures and the queue it reports to.
func (c *Calibration) Start(floors *relevance.Floors, queue host.Queue) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.floors = floors
	c.queue = queue
}

// GetFloors returns the floors, or ErrNotStarted before Start.
func (c *Calibration) GetFloors() (*relevance.Floors, error) {
	floors, _ := c.getParts()
	if floors == nil {
		return nil, ErrNotStarted
	}
	return floors, nil
}

func (c *Calibration) getParts() (*relevance.Floors, host.Queue) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.floors, c.queue
}

// HandleEvent measures the vault when the host reports that all of its vectors exist. The first event of any kind after the plugin starts checks the vault the host shows as well,
// so a plugin that starts after the vectors were made still measures.
// A vault that is measured, or whose floor a person set, is not measured again: an event the host repeats changes nothing.
func (c *Calibration) HandleEvent(ctx context.Context, event string) error {
	floors, queue := c.getParts()
	if floors == nil {
		return nil
	}
	vault, isVectorsReady := host.ReadVectorsReadyEvent(event)
	if !isVectorsReady && c.wasChecked() {
		return nil
	}
	if err := floors.Ensure(ctx, vault); err != nil {
		return c.reportFailure(ctx, queue, err)
	}
	c.markChecked()
	if err := queue.FinishTask(ctx, calibratingTask); err != nil {
		return fmt.Errorf("finish the task of the calibration: %w", err)
	}
	return nil
}

func (c *Calibration) wasChecked() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.isChecked
}

func (c *Calibration) markChecked() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.isChecked = true
}

func (c *Calibration) reportFailure(ctx context.Context, queue host.Queue, failure error) error {
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
