/**
 * A plugin the host suspends, started once more.
 */
import { formatErrorMessage } from '@numen/wire'
import type { MessageWriter } from '@/shared/notices/messages'

/** Everything this says in the window's voice. */
export interface RetryWords {
  /** The plugin failed to start again, which a reason follows. */
  readonly failedAgain: string
  /** The plugin could not be asked to start. */
  readonly error: string
}

/** What this asks of the vault. */
export interface RetryDeps {
  retryPlugin(pluginId: string): Promise<string>
}

/** Where this says what happened, and how the settings are read again after. */
export interface RetryOptions {
  readonly words: RetryWords
  readonly notify: MessageWriter
  readonly refresh: () => Promise<void>
}

/**
 * Starting a plugin again. A plugin that fails again is reported with its reason,
 * and the settings are read again either way, so what is drawn as failed is what
 * the host holds.
 */
export function createPluginRetry(core: RetryDeps, { words, notify, refresh }: RetryOptions) {
  /** The plugin started once more. */
  const retryPlugin = async (pluginId: string): Promise<void> => {
    notify('')
    try {
      const failure = await core.retryPlugin(pluginId)
      if (failure !== '') notify(`${words.failedAgain} ${failure}`, 'caution')
    } catch (caughtError) {
      notify(`${words.error} ${formatErrorMessage(caughtError)}`, 'error')
    }
    await refresh()
  }

  return { retryPlugin }
}
