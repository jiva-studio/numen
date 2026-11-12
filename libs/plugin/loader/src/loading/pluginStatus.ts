/**
 * Where each plugin stands: which are active, which have failed, which the window tracks, and
 * whether the loader runs. The state is made first and handed to the loader and to whatever reads
 * it, such as the host and the tabs of a plugin, so none of them needs the loader to exist.
 */
import { shallowRef } from 'vue'
import type { PluginModule } from '../module'
import type { ActivePluginEntry } from './activation'
import { createPluginStarts } from './pluginStarts'

export interface PluginStatusDeps {
  readonly isEnabled: (id: string) => boolean
}

export function createPluginStatus(deps: PluginStatusDeps) {
  const activePlugins = shallowRef<ReadonlyMap<string, ActivePluginEntry>>(new Map())
  const failures = shallowRef<ReadonlyMap<string, string>>(new Map())
  const moduleLoaders = new Map<string, () => Promise<PluginModule>>()
  const tracked = shallowRef<ReadonlySet<string>>(new Set())
  const isRunning = shallowRef(false)
  const listPlugins = (): string[] => [...new Set([...moduleLoaders.keys(), ...tracked.value])]
  const starts = createPluginStarts({
    isRunning,
    listPlugins,
    isEnabled: deps.isEnabled,
    activePlugins,
    failures,
  })
  return {
    /** What the loader keeps and changes; the rest only reads it. */
    state: {
      activePlugins,
      failures,
      moduleLoaders,
      tracked,
      isRunning,
      isEnabled: deps.isEnabled,
      listPlugins,
    },
    settlePlugin: starts.settlePlugin,
    isPluginStarting: starts.isPluginStarting,
    getPluginFailure: (id: string): string | undefined => failures.value.get(id),
  }
}

export type PluginStatus = ReturnType<typeof createPluginStatus>
