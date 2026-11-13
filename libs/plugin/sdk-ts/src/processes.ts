import type { Disposable } from './types'

// PluginProcess is a plugin that runs. Times are milliseconds since the Unix epoch. `lastUseTime` is
// when a lease was last taken or returned; a lease is an open call, an open stream or reported work.
// `idleStopTime` is from when the core stops the plugin for idleness, and is absent while the plugin
// holds a lease. `health` is the severity of the worst health item of the plugin, `ok` where it has
// none, and `healthText` the text of that item. `startedBy` is the holder whose lease started the
// plugin, and `holders` the holders of the leases open now, oldest first.
export interface PluginProcess {
  readonly pluginId: string
  readonly pluginName: string
  readonly startTime: number
  readonly lastUseTime: number
  readonly leaseCount: number
  readonly isInProcess: boolean
  readonly idleStopTime?: number
  readonly health: 'ok' | 'warning' | 'error'
  readonly healthText: string
  readonly startedBy: LeaseHolder
  readonly holders: readonly LeaseHolder[]
}

// LeaseHolder is who holds a lease on a plugin that runs and why: `who` is `core`, `window` or
// `plugin <id>`, followed by the call it came through where there is one.
export interface LeaseHolder {
  readonly who: string
  readonly why: string
}

// ProcessHost reports the plugins that run. It needs the permission plugins:read.
export interface ProcessHost {
  // Calls the handler with the plugins that run, in the order of their ids, and the time of the core
  // when it took the list, in milliseconds since the Unix epoch, at once and again on every change,
  // until the returned disposable is disposed. A list sent again unchanged carries its first time.
  watch(handler: (processes: readonly PluginProcess[], time: number) => void): Disposable
}
