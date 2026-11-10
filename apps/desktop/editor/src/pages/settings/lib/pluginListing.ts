/**
 * The mapping from the plugin catalog entries to the listings the plugin manager components
 * render.
 */
import {
  createManagedPlugins,
  type ManagedPlugin,
  type PluginInfo,
  type PluginListing,
} from '@numen/plugin-manager'
import type { CatalogEntry } from '@/entities/plugin'

const createPluginListing = (entry: CatalogEntry): PluginListing => ({
  id: entry.id,
  name: entry.name,
  version: entry.version,
  description: entry.description,
  category: entry.category.toLowerCase(),
  feature: entry.feature,
  tabKinds: [...entry.tabKinds],
  isAvailable: entry.isAvailable,
})

/** The plugins the host knows, each with its catalog entry, followed by the entries on offer. */
export const createManagedFromEntries = (
  plugins: readonly PluginInfo[],
  entries: readonly CatalogEntry[],
): readonly ManagedPlugin[] => createManagedPlugins(plugins, entries.map(createPluginListing))
