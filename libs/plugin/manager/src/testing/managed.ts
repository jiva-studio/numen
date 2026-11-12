import type { PluginInfo } from '../pluginInfo'
import { createMockListing, createMockPluginInfo, type MockListingOptions } from './listing'
import { createManagedPlugins, type ManagedPlugin } from '../plugins'

/** An installed plugin as the manager lists it, with the listing made from `listing`. */
export function createManagedPlugin(
  id: string,
  name: string,
  overrides: Partial<PluginInfo> = {},
  listing: MockListingOptions | undefined = undefined,
): ManagedPlugin {
  const info = createMockPluginInfo(id, name, overrides)
  const [plugin] = createManagedPlugins([info], listing ? [createMockListing(id, listing)] : [])
  if (!plugin) throw new Error('the plugin did not merge')
  return plugin
}

/** A plugin the catalog offers and the host does not have. */
export function createAvailablePlugin(id: string, name: string): ManagedPlugin {
  const [plugin] = createManagedPlugins([], [createMockListing(id, { name, isAvailable: true })])
  if (!plugin) throw new Error('the plugin did not merge')
  return plugin
}
