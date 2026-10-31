/**
 * Plugin state for the plugin list and page, and the failure time format.
 */
import type { PluginInfo } from '@numen/plugin-sdk'

/** What a plugin's state badge shows. */
export type PluginState = 'enabled' | 'disabled' | 'stopped' | 'available'

/** The plugins the host suspends, in the order they are listed. */
export const getFailedPlugins = (plugins: readonly PluginInfo[]): readonly PluginInfo[] =>
  plugins.filter((one) => one.failure !== undefined)

/** The state of a plugin: a failure outranks the enabled setting. */
export const getPluginState = (
  plugin: PluginInfo,
  isEnabled: boolean,
): Exclude<PluginState, 'available'> => {
  if (plugin.failure !== undefined) return 'stopped'
  return isEnabled ? 'enabled' : 'disabled'
}

/** The time a plugin was given up, in the locale's medium date and short time. */
export const formatFailedAt = (failedAt: number, locale?: string): string =>
  new Intl.DateTimeFormat(locale, { dateStyle: 'medium', timeStyle: 'short' }).format(
    new Date(failedAt),
  )
