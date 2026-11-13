/**
 * Where each plugin stands: which are active, which are starting, which have failed, which the
 * window tracks, and whether the loader runs. The state is made first and handed to the loader and to
 * whatever reads it, such as the host and the tabs of a plugin, so none of them needs the loader to
 * exist.
 */
import { shallowRef } from 'vue'
import type { Manifest } from '@numen/core-client'
import type { PluginModule } from '../module'
import type { ActivePluginEntry } from './activation'
import { createPluginStarts, type PluginStarter } from './pluginStarts'

export interface PluginStatusDeps {
  readonly isEnabled: (id: string) => boolean
}

export function createPluginStatus(deps: PluginStatusDeps) {
  const activePlugins = shallowRef<ReadonlyMap<string, ActivePluginEntry>>(new Map())
  const failures = shallowRef<ReadonlyMap<string, string>>(new Map())
  const starting = shallowRef<ReadonlySet<string>>(new Set())
  const manifests = new Map<string, Manifest>()
  const starter: { current: PluginStarter | undefined } = { current: undefined }
  const moduleLoaders = new Map<string, () => Promise<PluginModule>>()
  const tracked = shallowRef<ReadonlySet<string>>(new Set())
  const isRunning = shallowRef(false)
  const listPlugins = (): string[] => [...new Set([...moduleLoaders.keys(), ...tracked.value])]
  const starts = createPluginStarts({
    isRunning,
    listPlugins,
    isEnabled: deps.isEnabled,
    activePlugins,
    starting,
    manifests,
    starter,
  })
  return {
    /** What the loader keeps and changes; the rest only reads it. */
    state: {
      activePlugins,
      failures,
      starting,
      manifests,
      starter,
      moduleLoaders,
      tracked,
      isRunning,
      isEnabled: deps.isEnabled,
      listPlugins,
    },
    startPlugin: starts.startPlugin,
    startPluginsOf: starts.startPluginsOf,
    startTabPlugins: starts.startTabPlugins,
    findPluginsOf: starts.findPluginsOf,
    isPluginStarting: starts.isPluginStarting,
    getPluginFailure: (id: string): string | undefined => failures.value.get(id),
  }
}

export type PluginStatus = ReturnType<typeof createPluginStatus>
