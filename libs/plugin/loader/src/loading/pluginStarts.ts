/**
 * The starts of the plugins on use. A plugin frontend that has no slot of the window starts when
 * something it contributes is used, and the use waits for it: until the plugin is active, has failed,
 * or has passed PLUGIN_START_LIMIT_MS, which counts as failed. A failed plugin starts again on the
 * next use.
 */
import type { Manifest } from '@numen/core-client'
import type { ActivePluginEntry } from './activation'

/** What starts a plugin and reads its manifest; the loader gives it once it exists. */
export interface PluginStarter {
  readonly start: (id: string) => Promise<boolean>
  readonly readManifest: (id: string) => Promise<Manifest | undefined>
}

/** Whether a manifest declares a contribution, such as a tab kind or a drop type. */
export type ContributionMatch = (manifest: Manifest) => boolean

export interface PluginStartsDeps {
  readonly isRunning: { readonly value: boolean }
  readonly listPlugins: () => readonly string[]
  readonly isEnabled: (id: string) => boolean
  readonly activePlugins: { readonly value: ReadonlyMap<string, ActivePluginEntry> }
  readonly starting: { readonly value: ReadonlySet<string> }
  readonly manifests: ReadonlyMap<string, Manifest>
  readonly starter: { readonly current: PluginStarter | undefined }
}

export function createPluginStarts(deps: PluginStartsDeps) {
  const getEnabledPluginIds = (): string[] => deps.listPlugins().filter((id) => deps.isEnabled(id))

  /** Starts the plugin and settles once it is active or has failed; nothing for a plugin that is off. */
  const startPlugin = async (id: string): Promise<void> => {
    if (!deps.isRunning.value || deps.activePlugins.value.has(id)) return
    if (!deps.listPlugins().includes(id) || !deps.isEnabled(id)) return
    await deps.starter.current?.start(id)
  }

  const readManifest = (id: string): Promise<Manifest | undefined> =>
    deps.starter.current?.readManifest(id).catch(() => undefined) ?? Promise.resolve(undefined)

  /** Starts the enabled plugins whose manifest declares the contribution, and waits for them. */
  const startPluginsOf = async (match: ContributionMatch): Promise<void> => {
    if (!deps.isRunning.value) return
    const ids = getEnabledPluginIds()
    const manifests = await Promise.all(ids.map(readManifest))
    const matched = ids.filter((_, index) => {
      const manifest = manifests[index]
      return manifest !== undefined && match(manifest)
    })
    await Promise.all(matched.map(startPlugin))
  }

  return {
    startPlugin,
    startPluginsOf,
    /**
     * Starts the enabled plugins that declare one of the tab kinds, and every enabled plugin that
     * declares a tab kind where no kinds are given, and waits for them.
     */
    startTabPlugins: (kinds?: readonly string[]): Promise<void> =>
      startPluginsOf((manifest) =>
        (manifest.contributes?.tabKinds ?? []).some(
          (one) => kinds === undefined || kinds.includes(one.kind),
        ),
      ),
    /** The enabled plugins whose manifest, as read so far, declares the contribution. */
    findPluginsOf: (match: ContributionMatch): string[] =>
      getEnabledPluginIds().filter((id) => {
        const manifest = deps.manifests.get(id)
        return manifest !== undefined && match(manifest)
      }),
    /**
     * Whether the start of the plugin is going on; reactive. Before the plugins are listed every
     * plugin counts as starting, as none is known to be off yet.
     */
    isPluginStarting: (id: string): boolean => !deps.isRunning.value || deps.starting.value.has(id),
  }
}
