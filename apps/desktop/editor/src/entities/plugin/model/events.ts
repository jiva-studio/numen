import type { CommandNote, CommandRegistration, Disposable } from '@numen/plugin-sdk'
import type { PluginHostDeps } from './types'

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
export function reportError(deps: PluginHostDeps, pluginId: string, error: unknown): void {
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
      .catch((error: unknown) => reportError(deps, pluginId, error))
  }
}

export async function runCommand(
  deps: PluginHostDeps,
  pluginId: string,
  reg: CommandRegistration,
  note: CommandNote,
): Promise<void> {
  try {
    await reg.handler(note)
  } catch (error) {
    reportError(deps, pluginId, error)
  }
}
