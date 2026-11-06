/**
 * Controllable stand-ins of the slots, the vaults, the window and the background work of a window,
 * for the unit tests of the plugins that draw them.
 */
import type { Disposable } from '../types'
import type { SlotHost, SlotItem, SlotPlace, SlotTarget } from '../slots'
import type { VaultsHost, VaultsState, WindowHost, WindowState } from '../window'
import type { BackgroundTask, BackgroundTaskHost } from '../backgroundTasks'

/** Calls every handler of the set with the value. */
function notifyAll<T>(handlers: Set<(value: T) => void>, value: T): void {
  for (const handler of handlers) handler(value)
}

/** Adds the handler, calls it with the value at once, and takes it out on dispose. */
function watchWith<T>(
  handlers: Set<(value: T) => void>,
  handler: (value: T) => void,
  value: T,
): Disposable {
  handlers.add(handler)
  handler(value)
  return { dispose: () => handlers.delete(handler) }
}

// MockSlotRun is one run of a command a test reads back.
export interface MockSlotRun {
  readonly command: string
  readonly target: SlotTarget | undefined
}

// MockSlotHost is a SlotHost whose entries a test sets, and whose command runs it reads back.
export interface MockSlotHost extends SlotHost {
  // Sets the entries of a slot, whatever the target, and calls its watchers.
  setEntries(slot: SlotPlace, entries: readonly SlotItem[]): void
  getRuns(): MockSlotRun[]
}

export function createMockSlotHost(): MockSlotHost {
  const entries = new Map<string, readonly SlotItem[]>()
  const watchers = new Map<string, Set<(value: readonly SlotItem[]) => void>>()
  const runs: MockSlotRun[] = []
  const getWatchers = (slot: string) => {
    const found = watchers.get(slot) ?? new Set()
    watchers.set(slot, found)
    return found
  }
  return {
    getEntries: (slot) => entries.get(slot) ?? [],
    watchEntries: (slot, handler) => watchWith(getWatchers(slot), handler, entries.get(slot) ?? []),
    runCommand: async (command, target) => {
      runs.push({ command, target })
    },
    setEntries: (slot, next) => {
      entries.set(slot, next)
      notifyAll(getWatchers(slot), next)
    },
    getRuns: () => [...runs],
  }
}

// MockVaultsHost is a VaultsHost whose state a test sets, and whose switches it reads back.
export interface MockVaultsHost extends VaultsHost {
  setState(state: VaultsState): void
  getSwitches(): string[]
}

export function createMockVaultsHost(
  initial: VaultsState = { vaults: [], current: null },
): MockVaultsHost {
  let state = initial
  const watchers = new Set<(value: VaultsState) => void>()
  const switches: string[] = []
  return {
    getState: () => state,
    watch: (handler) => watchWith(watchers, handler, state),
    switchTo: async (id) => {
      switches.push(id)
    },
    setState: (next) => {
      state = next
      notifyAll(watchers, state)
    },
    getSwitches: () => [...switches],
  }
}

// MockWindowAction is one call of a window button a test reads back.
export type MockWindowAction = 'minimize' | 'toggleMaximize' | 'close'

// MockWindowHost is a WindowHost whose state a test sets, and whose button calls it reads back.
export interface MockWindowHost extends WindowHost {
  setState(state: WindowState): void
  getActions(): MockWindowAction[]
}

export function createMockWindowHost(
  initial: WindowState = { isMaximized: false, isFullscreen: false },
): MockWindowHost {
  let state = initial
  const watchers = new Set<(value: WindowState) => void>()
  const actions: MockWindowAction[] = []
  return {
    getState: () => state,
    watch: (handler) => watchWith(watchers, handler, state),
    minimize: () => {
      actions.push('minimize')
    },
    toggleMaximize: () => {
      actions.push('toggleMaximize')
    },
    close: () => {
      actions.push('close')
    },
    setState: (next) => {
      state = next
      notifyAll(watchers, state)
    },
    getActions: () => [...actions],
  }
}

// MockTaskAction is one call on a background task a test reads back.
export interface MockTaskAction {
  readonly action: 'cancel' | 'retry' | 'dismiss' | 'openSettings'
  // The id of the task, or of the plugin for openSettings.
  readonly id: string
}

// MockBackgroundTaskHost is a BackgroundTaskHost whose tasks a test sets, and whose calls it reads back.
export interface MockBackgroundTaskHost extends BackgroundTaskHost {
  setTasks(tasks: readonly BackgroundTask[]): void
  getActions(): MockTaskAction[]
}

export function createMockBackgroundTaskHost(
  initial: readonly BackgroundTask[] = [],
): MockBackgroundTaskHost {
  let tasks = initial
  const watchers = new Set<(value: readonly BackgroundTask[]) => void>()
  const actions: MockTaskAction[] = []
  return {
    watch: (handler) => watchWith(watchers, handler, tasks),
    cancel: async (id) => {
      actions.push({ action: 'cancel', id })
    },
    retry: (id) => {
      actions.push({ action: 'retry', id })
    },
    dismiss: (id) => {
      actions.push({ action: 'dismiss', id })
    },
    openSettings: (pluginId) => {
      actions.push({ action: 'openSettings', id: pluginId })
    },
    setTasks: (next) => {
      tasks = next
      notifyAll(watchers, tasks)
    },
    getActions: () => [...actions],
  }
}
