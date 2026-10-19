/**
 * The plugins the host suspends, and the words that say when.
 */
import type { PluginInfo } from '@/entities/settings'

/** The plugins the host suspends, in the order they are listed. */
export const getFailedPlugins = (plugins: readonly PluginInfo[]): readonly PluginInfo[] =>
  plugins.filter((one) => one.failure !== undefined)

/** The time a plugin was given up, as a person reads it. */
export const formatFailedAt = (failedAt: number, locale?: string): string =>
  new Intl.DateTimeFormat(locale, { dateStyle: 'medium', timeStyle: 'short' }).format(
    new Date(failedAt),
  )
