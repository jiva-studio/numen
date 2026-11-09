import type { Disposable } from './types'

// PluginTaskStatus is one task a plugin reported to the host. `pluginId` is the
// reporting plugin, as the host identifies the caller.
export interface PluginTaskStatus {
  readonly id: string
  readonly pluginId: string
  readonly pluginName: string
  readonly title: string
  readonly about: string
  readonly done: number
  readonly total: number
  readonly error: string
}

// TaskHost reports the tasks the plugins have reported. The handler is called with the current
// list at once and again on every change. The core's own work is not included.
export interface TaskHost {
  // The tasks of the calling plugin.
  watch(handler: (tasks: readonly PluginTaskStatus[]) => void): Disposable
  // The tasks of every plugin.
  watchAll(handler: (tasks: readonly PluginTaskStatus[]) => void): Disposable
}
