import type { CommandNote, CommandRegistration, Disposable, Caught } from '@numen/plugin-sdk'
import type { CommandRun, PluginHostDeps } from './types'

export interface PluginListener<E> {
  readonly pluginId: string
  readonly handler: (event: E) => void | Promise<void>
}

export function addListener<E>(
  listeners: Set<PluginListener<E>>,
  listener: PluginListener<E>,
): Disposable {
  listeners.add(listener)
  return {
    dispose: () => {
      listeners.delete(listener)
    },
  }
}

/** Hands a plugin's failure to the host's reporter, under the id of the plugin. */
export function reportError(deps: PluginHostDeps, pluginId: string, error: Caught): void {
  deps.onError?.(pluginId, error)
}

/** Calls every listener with the event; a listener that fails is reported and the rest still run. */
export function notifyListeners<E>(
  listeners: ReadonlySet<PluginListener<E>>,
  event: E,
  deps: PluginHostDeps,
): void {
  for (const { pluginId, handler } of [...listeners]) {
    Promise.resolve()
      .then(() => handler(event))
      .catch((error: Caught) => reportError(deps, pluginId, error))
  }
}

/** A command a plugin registered, with the plugin. */
export interface CommandEntry {
  readonly pluginId: string
  readonly reg: CommandRegistration
}

export async function runCommand(
  deps: PluginHostDeps,
  entry: CommandEntry,
  run: { readonly note: CommandNote } & Pick<CommandRun, 'itemId' | 'values' | 'isAwaited'>,
): Promise<void> {
  const { note, itemId, values = {} } = run
  try {
    await entry.reg.handler(note, itemId, values)
  } catch (error) {
    if (run.isAwaited) throw error
    reportError(deps, entry.pluginId, error)
  }
}

/**
 * Starts the plugin in the background, for a use that does not wait for it, such as the draw of a
 * slot. A plugin whose start failed is left until a use that waits for it.
 */
export function requestPluginStart(deps: PluginHostDeps, pluginId: string): void {
  const { startPlugin } = deps
  if (!startPlugin || deps.isPluginFailed?.(pluginId)) return
  // The draw that asks may be inside a computed value, which must not change state as it is read.
  queueMicrotask(() => {
    startPlugin(pluginId).catch((error: Caught) => reportError(deps, pluginId, error))
  })
}
