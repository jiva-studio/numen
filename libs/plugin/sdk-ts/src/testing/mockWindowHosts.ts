/**
 * Controllable stand-ins of the commands, the slots, the vaults, the window, the background work, the health items and the plugins that run of a window,
 * for the unit tests of the plugins that draw them.
 */
import type { Disposable } from '../types'
import type { CommandItem, CommandsHost, CommandTarget } from '../commands'
import type { SlotHost, SlotItem, SlotPlace } from '../slots'
import type { VaultsHost, VaultsState, WindowHost, WindowState } from '../window'
import type { BackgroundTask, BackgroundTaskHost } from '../backgroundTasks'
import type { HealthHost, HealthItem } from '../health'
import type { PluginProcess, ProcessHost } from '../processes'

function notifyAll<T>(handlers: Set<(value: T) => void>, value: T): void {
  for (const handler of handlers) handler(value)
}

function watchWith<T>(
  handlers: Set<(value: T) => void>,
  handler: (value: T) => void,
  value: T,
): Disposable {
  handlers.add(handler)
  handler(value)
  return { dispose: () => handlers.delete(handler) }
}

// MockCommandRun is one run of a command a test reads back.
export interface MockCommandRun {
  readonly command: string
  readonly target: CommandTarget | undefined
  readonly itemId: string | undefined
  readonly values?: Readonly<Record<string, string>>
}

// MockCommandsHost is a CommandsHost whose ids a test sets, and whose runs it reads back.
export interface MockCommandsHost extends CommandsHost {
  setIds(ids: readonly string[]): void
  // Sets the items a command offers, whatever the target.
  setItems(command: string, items: readonly CommandItem[]): void
  // Sets the path a run of the command resolves with; a run of any other command resolves with an empty path.
  setResult(command: string, path: string): void
  getRuns(): MockCommandRun[]
}

export function createMockCommandsHost(initial: readonly string[] = []): MockCommandsHost {
  let ids = initial
  const watchers = new Set<(value: readonly string[]) => void>()
  const runs: MockCommandRun[] = []
  const items = new Map<string, readonly CommandItem[]>()
  const results = new Map<string, string>()
  return {
    getIds: () => ids,
    watch: (handler) => watchWith(watchers, handler, ids),
    listItems: (command) => Promise.resolve(items.get(command) ?? []),
    run: (command, target, itemId, values) => {
      runs.push({ command, target, itemId, ...(values ? { values } : {}) })
      return Promise.resolve(results.get(command) ?? '')
    },
    setResult: (command, path) => {
      results.set(command, path)
    },
    setItems: (command, next) => {
      items.set(command, next)
    },
    setIds: (next) => {
      ids = next
      notifyAll(watchers, ids)
    },
    getRuns: () => [...runs],
  }
}

// MockSlotHost is a SlotHost whose entries a test sets.
export interface MockSlotHost extends SlotHost {
  // Sets the entries of a slot, whatever the target, and calls its watchers.
  setEntries(slot: SlotPlace, entries: readonly SlotItem[]): void
}

export function createMockSlotHost(): MockSlotHost {
  const entries = new Map<string, readonly SlotItem[]>()
  const watchers = new Map<string, Set<(value: readonly SlotItem[]) => void>>()
  const getWatchers = (slot: string) => {
    const found = watchers.get(slot) ?? new Set()
    watchers.set(slot, found)
    return found
  }
  return {
    getEntries: (slot) => entries.get(slot) ?? [],
    watchEntries: (slot, handler) => watchWith(getWatchers(slot), handler, entries.get(slot) ?? []),
    setEntries: (slot, next) => {
      entries.set(slot, next)
      notifyAll(getWatchers(slot), next)
    },
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
    switchTo: (id) => {
      switches.push(id)
      return Promise.resolve()
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
  readonly action: 'cancel' | 'retry' | 'dismiss' | 'runAction'
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
    cancel: (id) => {
      actions.push({ action: 'cancel', id })
      return Promise.resolve()
    },
    retry: (id) => {
      actions.push({ action: 'retry', id })
    },
    dismiss: (id) => {
      actions.push({ action: 'dismiss', id })
    },
    runAction: (id) => {
      actions.push({ action: 'runAction', id })
    },
    setTasks: (next) => {
      tasks = next
      notifyAll(watchers, tasks)
    },
    getActions: () => [...actions],
  }
}

// MockHealthHost is a HealthHost that keeps the items of the plugin under test, in the order they
// were first reported, for a test to read back.
export interface MockHealthHost extends HealthHost {
  getItems(): readonly HealthItem[]
}

export function createMockHealthHost(): MockHealthHost {
  const items = new Map<string, HealthItem>()
  return {
    report: (item) => {
      items.set(item.key, item)
      return Promise.resolve()
    },
    clear: (key) => {
      items.delete(key)
      return Promise.resolve()
    },
    getItems: () => [...items.values()],
  }
}

// MockProcessHost is a ProcessHost whose plugins a test sets, and whose watchers it counts.
export interface MockProcessHost extends ProcessHost {
  setProcesses(processes: readonly PluginProcess[]): void
  countWatchers(): number
}

export function createMockProcessHost(initial: readonly PluginProcess[] = []): MockProcessHost {
  let processes = initial
  const watchers = new Set<(value: readonly PluginProcess[]) => void>()
  return {
    watch: (handler) => watchWith(watchers, handler, processes),
    setProcesses: (next) => {
      processes = next
      notifyAll(watchers, processes)
    },
    countWatchers: () => watchers.size,
  }
}
