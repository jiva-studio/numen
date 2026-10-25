package run

import (
	"context"
	"errors"
	"sync"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// pluginTaskPrefix starts the id a plugin's task is listed under. The plugin's
// id follows it, so the tasks of two plugins and of the core never share an id.
const pluginTaskPrefix = "plugin:"

// errPluginCleared is the wait of a plugin whose work was cleared.
var errPluginCleared = errors.New("the plugin's work was cleared")

// PluginWork is the lock and the list of tasks the core's own work uses, as
// the plugins use them. It is safe to use from several goroutines.
type PluginWork struct {
	lock  *Lock
	tasks *Tasks

	mu      sync.Mutex
	plugins map[string]*pluginState
	next    uint64
}

// pluginState is what one plugin holds: the ids of its tasks, its turns and
// the cancellations of its waits.
type pluginState struct {
	tasks map[string]bool
	turns map[*pluginTurn]bool
	waits map[uint64]context.CancelFunc
}

var _ pluginport.PluginWork = (*PluginWork)(nil)

// NewPluginWork opens the lock and the list of tasks to the plugins.
func NewPluginWork(lock *Lock, tasks *Tasks) *PluginWork {
	return &PluginWork{lock: lock, tasks: tasks, plugins: map[string]*pluginState{}}
}

// EnqueueWork waits at the lock for the plugin's turn and returns it.
func (w *PluginWork) EnqueueWork(
	ctx context.Context, pluginID string, isUserStarted bool, onWaiting func(),
) (pluginport.PluginTurn, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	w.mu.Lock()
	w.next++
	key := w.next
	w.getStateLocked(pluginID).waits[key] = cancel
	w.mu.Unlock()

	release, err := w.lock.Acquire(ctx, isUserStarted, onWaiting)

	w.mu.Lock()
	defer w.mu.Unlock()
	state, isKnown := w.plugins[pluginID]
	isWaiting := isKnown && state.waits[key] != nil
	if isWaiting {
		delete(state.waits, key)
	}
	if err != nil {
		if !isWaiting {
			return nil, errPluginCleared
		}
		return nil, err
	}
	// A plugin cleared while the lock was handed to it gives the lock back.
	if !isWaiting {
		release()
		return nil, errPluginCleared
	}
	turn := &pluginTurn{lock: w.lock, release: release, end: make(chan struct{})}
	turn.forget = func() {
		w.mu.Lock()
		defer w.mu.Unlock()
		if state, isKnown := w.plugins[pluginID]; isKnown {
			delete(state.turns, turn)
		}
	}
	state.turns[turn] = true
	return turn, nil
}

// ReportTask lists the task under the plugin's own id. A task with no id is
// not listed.
func (w *PluginWork) ReportTask(pluginID string, task pluginport.PluginTask) {
	if task.ID == "" {
		return
	}
	w.mu.Lock()
	w.getStateLocked(pluginID).tasks[task.ID] = true
	w.mu.Unlock()
	w.tasks.Set(Task{
		ID:            getPluginTaskID(pluginID, task.ID),
		Title:         task.Title,
		About:         task.About,
		Count:         task.Done,
		Total:         task.Total,
		Unit:          readUnit(task.Unit),
		Error:         task.Error,
		IsUserStarted: task.IsUserStarted,
	})
}

// FinishTask takes the plugin's task out of the list.
func (w *PluginWork) FinishTask(pluginID, taskID string) {
	w.mu.Lock()
	if state, isKnown := w.plugins[pluginID]; isKnown {
		delete(state.tasks, taskID)
	}
	w.mu.Unlock()
	w.tasks.Remove(getPluginTaskID(pluginID, taskID))
}

// ClearPlugin takes the plugin's tasks out of the list, releases its turns and
// ends its waits.
func (w *PluginWork) ClearPlugin(pluginID string) {
	w.mu.Lock()
	state, isKnown := w.plugins[pluginID]
	delete(w.plugins, pluginID)
	w.mu.Unlock()
	if !isKnown {
		return
	}
	for _, cancel := range state.waits {
		cancel()
	}
	for turn := range state.turns {
		turn.Release()
	}
	for taskID := range state.tasks {
		w.tasks.Remove(getPluginTaskID(pluginID, taskID))
	}
}

// getStateLocked returns what the plugin holds, making it on its first call.
// The caller holds w.mu.
func (w *PluginWork) getStateLocked(pluginID string) *pluginState {
	state, isKnown := w.plugins[pluginID]
	if !isKnown {
		state = &pluginState{
			tasks: map[string]bool{},
			turns: map[*pluginTurn]bool{},
			waits: map[uint64]context.CancelFunc{},
		}
		w.plugins[pluginID] = state
	}
	return state
}

func getPluginTaskID(pluginID, taskID string) string {
	return pluginTaskPrefix + pluginID + ":" + taskID
}

// units are the units of the list, by the unit a plugin counts in.
var units = map[pluginport.TaskUnit]Unit{
	pluginport.TaskUnitThings:  Things,
	pluginport.TaskUnitBytes:   Bytes,
	pluginport.TaskUnitSeconds: Seconds,
}

func readUnit(unit pluginport.TaskUnit) Unit {
	if listed, isKnown := units[unit]; isKnown {
		return listed
	}
	return Things
}

// pluginTurn is a plugin's hold on the lock.
type pluginTurn struct {
	lock    *Lock
	release func()
	forget  func()
	once    sync.Once
	end     chan struct{}
}

func (t *pluginTurn) Release() {
	t.once.Do(func() {
		t.forget()
		t.release()
		close(t.end)
	})
}

func (t *pluginTurn) WatchWaiting() <-chan struct{} { return t.lock.WatchWaiting() }

func (t *pluginTurn) WatchEnd() <-chan struct{} { return t.end }
