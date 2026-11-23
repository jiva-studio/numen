/**
 * Where each plugin stands: which are active, which are starting, which have failed, which the
 * window tracks, and whether the loader runs. The state is made first and handed to the loader and to
 * whatever reads it, such as the host and the tabs of a plugin, so none of them needs the loader to
 * exist.
 */
import { shallowRef, type ShallowRef } from 'vue'
import type { Manifest } from '@numen/core-client'
import type { PluginModule } from '../module'
import type { ActivePluginEntry } from './activation'
import { createPluginStarts, type ContributionMatch, type PluginStarter } from './pluginStarts'

export interface PluginStatusDeps {
  readonly isEnabled: (id: string) => boolean
}

export interface PluginStatusState {
  readonly activePlugins: ShallowRef<ReadonlyMap<string, ActivePluginEntry>>
  readonly failures: ShallowRef<ReadonlyMap<string, string>>
  readonly starting: ShallowRef<ReadonlySet<string>>
  readonly manifests: Map<string, Manifest>
  readonly starter: { current: PluginStarter | undefined }
  readonly moduleLoaders: Map<string, () => Promise<PluginModule>>
  readonly tracked: ShallowRef<ReadonlySet<string>>
  readonly isRunning: ShallowRef<boolean>
  readonly isEnabled: (id: string) => boolean
  readonly listPlugins: () => string[]
}

/**
 * `state` is what the loader keeps and changes; the rest only reads it. `startTabPlugins` starts the
 * enabled plugins that declare one of the tab kinds, and every enabled plugin that declares a tab
 * kind where no kinds are given, and waits for them. `findPluginsOf` gives the enabled plugins whose
 * manifest, as read so far, declares the contribution. `isPluginStarting` is reactive and, before
 * the plugins are listed, true for every plugin, as none is known to be off yet.
 */
export interface PluginStatus {
  readonly state: PluginStatusState
  readonly startPlugin: (id: string) => Promise<void>
  readonly startPluginsOf: (match: ContributionMatch) => Promise<void>
  readonly startTabPlugins: (kinds?: readonly string[]) => Promise<void>
  readonly findPluginsOf: (match: ContributionMatch) => string[]
  readonly isPluginStarting: (id: string) => boolean
  readonly getPluginFailure: (id: string) => string | undefined
}

export function createPluginStatus(deps: PluginStatusDeps): PluginStatus {
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
