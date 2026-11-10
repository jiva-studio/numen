/**
 * Retry of a plugin whose last start failed.
 */
import { formatError } from '@/shared/api/errors'
import type { MessageWriter } from '@/shared/model/notices/messages'
import { retryPlugin as askPluginRetry } from '../api/settings'

/** Notice copy of a retry. */
interface RetryWords {
  readonly stopped: (name: string, reason: string) => string
  readonly retryFailed: (name: string, message: string) => string
}

/** Notice output, settings refresh and the plugin name lookup of a retry. */
interface RetryOptions {
  readonly words: RetryWords
  readonly notify: MessageWriter
  readonly refresh: () => Promise<void>
  readonly getName: (pluginId: string) => string
}

/**
 * Starts a plugin again. A repeated failure is shown as a notice with the
 * plugin name and the reason; the settings are read again in every case.
 */
export function createPluginRetry({ words, notify, refresh, getName }: RetryOptions) {
  const retryPlugin = async (pluginId: string): Promise<void> => {
    notify('')
    try {
      const failure = await askPluginRetry(pluginId)
      if (failure !== '') notify(words.stopped(getName(pluginId), failure), 'warning')
    } catch (caughtError) {
      notify(words.retryFailed(getName(pluginId), formatError(caughtError)), 'error')
    }
    await refresh()
  }

  return { retryPlugin }
}
