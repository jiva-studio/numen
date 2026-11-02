/**
 * The mapping from the plugin catalog entries, which are protocol messages, to the listings the
 * plugin manager components render.
 */
import {
  createManagedPlugins,
  type ManagedPlugin,
  type PluginInfo,
  type PluginListing,
} from '@numen/plugin-manager'
import { CATALOG_STATE, type CatalogEntry } from '@numen/core-client'

/** The listing of one catalog entry: its category in lowercase and whether a source offers it. */
export const createPluginListing = (entry: CatalogEntry): PluginListing => ({
  id: entry.id,
  name: entry.name,
  version: entry.version,
  description: entry.description,
  category: (entry.contributes?.category || entry.category || '').toLowerCase(),
  feature: entry.contributes?.feature ?? '',
  tabKinds: entry.contributes?.tabKinds.map((one) => one.kind) ?? [],
  isOffered: entry.state === CATALOG_STATE.AVAILABLE,
})

/** The plugins the host knows, each with its catalog entry, followed by the entries on offer. */
export const createManagedFromEntries = (
  plugins: readonly PluginInfo[],
  entries: readonly CatalogEntry[],
): readonly ManagedPlugin[] => createManagedPlugins(plugins, entries.map(createPluginListing))
