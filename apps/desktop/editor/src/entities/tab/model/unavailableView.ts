import { CATALOG_INCOMPATIBILITY, CATALOG_STATE, type CatalogEntry } from '@numen/core-client'
import type { UnavailableLookup } from './unavailable'

/** What the person can do about the plugin: switch it on, install it, or look at it. */
export type UnavailableAction = 'enable' | 'install' | 'inspect'

/** The words and the action of the placeholder of one tab. */
export interface UnavailableView {
  readonly heading: string
  readonly message: string
  readonly action: UnavailableAction
  readonly actionText: string
  readonly pluginId: string | undefined
}

const MISSING: UnavailableView = {
  heading: 'Plugin unavailable',
  message: 'The plugin for this tab is disabled or missing.',
  action: 'inspect',
  actionText: 'Open Settings',
  pluginId: undefined,
}

const isIncompatible = (entry: CatalogEntry): boolean =>
  entry.incompatibility !== CATALOG_INCOMPATIBILITY.UNSPECIFIED

const createIncompatible = (entry: CatalogEntry): UnavailableView => ({
  heading: 'Plugin incompatible',
  message: `${entry.name} cannot run in this application: ${entry.incompatibilityDetail}`,
  action: 'inspect',
  actionText: 'Open Settings',
  pluginId: entry.id,
})

const createDisabled = (
  entry: CatalogEntry,
  feature: string,
  isEnabling: boolean,
): UnavailableView => ({
  heading: 'Plugin disabled',
  message: `The plugin for ${feature} is disabled.`,
  action: 'enable',
  actionText: isEnabling ? `Enabling ${entry.name}…` : `Enable ${entry.name}`,
  pluginId: entry.id,
})

const createAvailable = (entry: CatalogEntry, feature: string): UnavailableView => ({
  heading: 'Plugin not installed',
  message: `The plugin for ${feature} is not installed.`,
  action: 'install',
  actionText: `Install ${entry.name}`,
  pluginId: entry.id,
})

const createNotRunning = (entry: CatalogEntry, feature: string): UnavailableView => ({
  heading: 'Plugin not running',
  message: `The plugin for ${feature} is not running.`,
  action: 'inspect',
  actionText: 'Open Settings',
  pluginId: entry.id,
})

/**
 * What the placeholder of a tab says, read from the catalog entry of the plugin that provides its
 * kind. A lookup the catalog has not answered has no view.
 */
export function getUnavailableView(
  lookup: UnavailableLookup,
  isEnabling: boolean,
): UnavailableView | undefined {
  if (lookup.status === 'pending') return undefined
  if (lookup.status === 'missing') return MISSING
  const { entry } = lookup
  if (isIncompatible(entry)) return createIncompatible(entry)
  const feature = entry.contributes?.feature || entry.name
  if (entry.state === CATALOG_STATE.DISABLED) return createDisabled(entry, feature, isEnabling)
  if (entry.state === CATALOG_STATE.AVAILABLE) return createAvailable(entry, feature)
  return createNotRunning(entry, feature)
}
