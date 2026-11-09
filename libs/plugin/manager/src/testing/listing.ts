import type { PluginInfo } from '../pluginInfo'
import type { PluginListing } from '../plugins'

/** What a listing made for a test is given. */
export interface MockListingOptions {
  readonly name?: string
  readonly version?: string
  readonly description?: string
  readonly category?: string
  readonly feature?: string
  readonly isAvailable?: boolean
  readonly tabKinds?: readonly string[]
}

/** A listing for the plugin `id`, installed unless the options say it is offered. */
export function createMockListing(id: string, options: MockListingOptions = {}): PluginListing {
  return {
    id,
    name: options.name ?? id,
    version: options.version ?? '1.0.0',
    description: options.description ?? '',
    category: (options.category ?? '').toLowerCase(),
    feature: options.feature ?? '',
    tabKinds: options.tabKinds ?? [],
    isAvailable: options.isAvailable ?? false,
  }
}

/** A plugin the host lists, with what a test cares about put over the defaults. */
export function createMockPluginInfo(
  id: string,
  name: string,
  overrides: Partial<PluginInfo> = {},
): PluginInfo {
  return {
    id,
    name,
    version: '1.0.0',
    description: '',
    isActive: false,
    settings: [],
    capabilities: [],
    secrets: [],
    hasFrontend: false,
    hasBackend: false,
    ...overrides,
  }
}
