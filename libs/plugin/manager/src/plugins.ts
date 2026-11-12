import type { PluginInfo } from './pluginInfo'

/** The group a plugin without a category is listed under. */
export const OTHER_CATEGORY = 'other'

/** One plugin a catalog describes, as the manager lists it. */
export interface PluginListing {
  readonly id: string
  readonly name: string
  readonly version: string
  readonly description: string
  readonly category: string
  readonly feature: string
  readonly tabKinds: readonly string[]
  readonly isAvailable: boolean
}

/**
 * One plugin as the manager lists it: what the host reports of a plugin it knows, and what the
 * catalog declares of it. A plugin the catalog offers and the host does not know has no `info`.
 */
export interface ManagedPlugin {
  readonly id: string
  readonly name: string
  readonly version: string
  readonly description: string
  readonly category: string
  readonly feature: string
  readonly tabKinds: readonly string[]
  readonly info: PluginInfo | undefined
}

/** Whether the host has the plugin on disk. */
export const isInstalled = (plugin: ManagedPlugin): boolean => plugin.info !== undefined

const createInstalledPlugin = (info: PluginInfo, listing: PluginListing | undefined): ManagedPlugin => ({
  id: info.id,
  name: info.name,
  version: info.version,
  description: info.description,
  category: listing?.category ?? '',
  feature: listing?.feature ?? '',
  tabKinds: listing?.tabKinds ?? [],
  info,
})

const createAvailablePlugin = (listing: PluginListing): ManagedPlugin => ({
  id: listing.id,
  name: listing.name,
  version: listing.version,
  description: listing.description,
  category: listing.category,
  feature: listing.feature,
  tabKinds: listing.tabKinds,
  info: undefined,
})

/**
 * The plugins the host knows, each with its catalog listing, followed by the listings that are
 * offered and not installed.
 */
export function createManagedPlugins(
  plugins: readonly PluginInfo[],
  listings: readonly PluginListing[],
): readonly ManagedPlugin[] {
  const byId = new Map(listings.map((one) => [one.id, one]))
  const installedIds = new Set(plugins.map((one) => one.id))
  const uninstalled = listings.filter((one) => one.isAvailable && !installedIds.has(one.id))
  return [
    ...plugins.map((one) => createInstalledPlugin(one, byId.get(one.id))),
    ...uninstalled.map(createAvailablePlugin),
  ]
}
