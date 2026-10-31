import { CATALOG_STATE, type CatalogEntry, type PluginInfo } from '@numen/plugin-sdk'

/** The group a plugin without a category is listed under. */
export const OTHER_CATEGORY = 'other'

/**
 * One plugin as the manager lists it: what the host reports of a plugin it knows, and what the
 * catalog declares of it. A plugin the catalog offers and the host does not know has no `info`.
 */
export interface ManagedPlugin {
  readonly id: string
  readonly name: string
  readonly version: string
  readonly description: string
  /** The group the manifest declares, lowercase; empty where it declares none. */
  readonly category: string
  /** What the plugin provides, in words a person reads; empty where the manifest says none. */
  readonly feature: string
  /** The ids of the kinds of tab the plugin registers. */
  readonly tabKinds: readonly string[]
  /** The host's report of the plugin; absent for a plugin that is not installed. */
  readonly info: PluginInfo | undefined
  readonly entry: CatalogEntry | undefined
}

/** Whether the host has the plugin on disk. */
export const isInstalled = (plugin: ManagedPlugin): boolean => plugin.info !== undefined

const readCategory = (entry: CatalogEntry | undefined): string =>
  (entry?.contributes?.category || entry?.category || '').toLowerCase()

const readFeature = (entry: CatalogEntry | undefined): string => entry?.contributes?.feature ?? ''

const readTabKinds = (entry: CatalogEntry | undefined): readonly string[] =>
  entry?.contributes?.tabKinds.map((one) => one.kind) ?? []

const createInstalled = (info: PluginInfo, entry: CatalogEntry | undefined): ManagedPlugin => ({
  id: info.id,
  name: info.name,
  version: info.version,
  description: info.description,
  category: readCategory(entry),
  feature: readFeature(entry),
  tabKinds: readTabKinds(entry),
  info,
  entry,
})

const createOffered = (entry: CatalogEntry): ManagedPlugin => ({
  id: entry.id,
  name: entry.name,
  version: entry.version,
  description: entry.description,
  category: readCategory(entry),
  feature: readFeature(entry),
  tabKinds: readTabKinds(entry),
  info: undefined,
  entry,
})

/**
 * The plugins the host knows, each with its catalog entry, followed by the entries the catalog
 * offers that are not installed.
 */
export function createManagedPlugins(
  plugins: readonly PluginInfo[],
  entries: readonly CatalogEntry[],
): readonly ManagedPlugin[] {
  const byId = new Map(entries.map((one) => [one.id, one]))
  const installedIds = new Set(plugins.map((one) => one.id))
  const uninstalled = entries.filter(
    (one) => one.state === CATALOG_STATE.AVAILABLE && !installedIds.has(one.id),
  )
  return [
    ...plugins.map((one) => createInstalled(one, byId.get(one.id))),
    ...uninstalled.map(createOffered),
  ]
}
