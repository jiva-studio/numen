/**
 * Where the start of each plugin stands, as a tab of one of its kinds reads it: going on until the
 * plugin is active, has failed, or has passed PLUGIN_START_LIMIT_MS, which counts as failed.
 */
import { watch } from 'vue'
import type { ActivePluginEntry } from './activation'
import { PLUGIN_START_LIMIT_MS } from './limits'

export interface PluginStartsDeps {
  readonly isRunning: { readonly value: boolean }
  readonly listPlugins: () => readonly string[]
  readonly isEnabled: (id: string) => boolean
  readonly activePlugins: { readonly value: ReadonlyMap<string, ActivePluginEntry> }
  readonly failures: { readonly value: ReadonlyMap<string, string> }
}

export function createPluginStarts(deps: PluginStartsDeps) {
  const isSettled = (id: string): boolean =>
    deps.activePlugins.value.has(id) || deps.failures.value.has(id)

  /** Settles once the plugin is active or has failed, or after PLUGIN_START_LIMIT_MS. */
  const settlePlugin = (id: string): Promise<void> =>
    new Promise<void>((resolve) => {
      if (isSettled(id)) return resolve()
      const stop = watch([deps.activePlugins, deps.failures], () => {
        if (!isSettled(id)) return
        stop()
        clearTimeout(timer)
        resolve()
      })
      const timer = setTimeout(() => {
        stop()
        resolve()
      }, PLUGIN_START_LIMIT_MS)
    })

  return {
    settlePlugin,
    /**
     * Whether the start of the plugin is going on; reactive. Before the plugins are listed every
     * plugin counts as starting, as none is known to be off yet.
     */
    isPluginStarting: (id: string): boolean => {
      if (!deps.isRunning.value) return true
      return deps.listPlugins().includes(id) && deps.isEnabled(id) && !isSettled(id)
    },
  }
}
