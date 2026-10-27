/**
 * Retry of a plugin the host suspends.
 */
import { formatErrorMessage } from '@numen/wire'
import type { MessageWriter } from '@/shared/notices/messages'

/** Notice copy of a retry. */
export interface RetryWords {
  /** The plugin failed to start again: "<Name> stopped: <reason>". */
  readonly stopped: (name: string, reason: string) => string
  /** The retry request itself failed. */
  readonly retryFailed: (name: string, message: string) => string
}

/** The core call a retry makes. */
export interface RetryDeps {
  retryPlugin(pluginId: string): Promise<string>
}

/** Notice output, settings refresh and the plugin name lookup of a retry. */
export interface RetryOptions {
  readonly words: RetryWords
  readonly notify: MessageWriter
  readonly refresh: () => Promise<void>
  /** The display name of a plugin, or its id when it has none. */
  readonly getName: (pluginId: string) => string
}

/**
 * Starts a plugin again. A repeated failure is shown as a notice with the
 * plugin name and the reason; the settings are read again in every case.
 */
export function createPluginRetry(
  core: RetryDeps,
  { words, notify, refresh, getName }: RetryOptions,
) {
  const retryPlugin = async (pluginId: string): Promise<void> => {
    notify('')
    try {
      const failure = await core.retryPlugin(pluginId)
      if (failure !== '') notify(words.stopped(getName(pluginId), failure), 'warning')
    } catch (caughtError) {
      notify(words.retryFailed(getName(pluginId), formatErrorMessage(caughtError)), 'error')
    }
    await refresh()
  }

  return { retryPlugin }
}
