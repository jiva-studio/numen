import { CATALOG_STATE, type PluginInfo } from '@numen/plugin-sdk'
import {
  createMockCatalogEntry,
  createMockPluginInfo,
  type MockCatalogEntryOptions,
} from '@numen/plugin-sdk/testing'
import { createManagedPlugins, type ManagedPlugin } from '../plugins'

/** An installed plugin as the manager lists it, with the catalog entry made from `entry`. */
export function createManaged(
  id: string,
  name: string,
  overrides: Partial<PluginInfo> = {},
  entry: MockCatalogEntryOptions | undefined = undefined,
): ManagedPlugin {
  const info = createMockPluginInfo(id, name, overrides)
  const [plugin] = createManagedPlugins([info], entry ? [createMockCatalogEntry(id, entry)] : [])
  if (!plugin) throw new Error('the plugin did not merge')
  return plugin
}

/** A plugin the catalog offers and the host does not have. */
export function createOffered(id: string, name: string): ManagedPlugin {
  const [plugin] = createManagedPlugins(
    [],
    [createMockCatalogEntry(id, { name, state: CATALOG_STATE.AVAILABLE })],
  )
  if (!plugin) throw new Error('the plugin did not merge')
  return plugin
}
