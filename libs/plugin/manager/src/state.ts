/**
 * The state of a plugin for the plugin list and page, from its enabled setting and its health items.
 */
import type { PluginHealth } from './pluginInfo'

/** What a plugin's state badge shows. */
export type PluginState = 'enabled' | 'disabled' | 'stopped' | 'error' | 'warning' | 'available'

/** The key of the item of a plugin whose start failed, which the host holds suspended. */
export const START_HEALTH_KEY = 'start'

/** The item a plugin whose start failed has, or none. */
export const getStartItem = (health: readonly PluginHealth[]): PluginHealth | undefined =>
  health.find((one) => one.key === START_HEALTH_KEY)

/** The worst of the items: an error over a warning, the first reported among equals. */
export const getWorstItem = (health: readonly PluginHealth[]): PluginHealth | undefined =>
  health.find((one) => one.severity === 'error') ?? health[0]

/** The state of a plugin: a failed start outranks an error, an error a warning, and those the setting. */
export const getPluginState = (
  isEnabled: boolean,
  health: readonly PluginHealth[] = [],
): Exclude<PluginState, 'available'> => {
  if (getStartItem(health)) return 'stopped'
  const worst = getWorstItem(health)
  if (worst) return worst.severity
  return isEnabled ? 'enabled' : 'disabled'
}
