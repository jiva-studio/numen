import { describe, expect, it, vi } from 'vitest'
import { createLostConnection } from '@numen/core-client/testing'
import type { CoreAnswers } from '@numen/core-client/testing'
import { createAnsweredApi } from '@/testing/coreAnswers'
import { createPluginRetry } from './retry'

const createAnsweredSettings = (answers: CoreAnswers) => createAnsweredApi(answers).settings

const words = {
  stopped: (name: string, reason: string) => `${name} stopped: ${reason}`,
  retryFailed: (name: string, message: string) => `Could not retry ${name}: ${message}`,
}

const createRetry = (answer: { failure: string } | { rejection: string }) => {
  const calls: string[] = []
  const settings = createAnsweredSettings({
    settingsService: {
      retryPlugin: (request) => {
        calls.push(request.pluginId)
        if ('rejection' in answer) throw createLostConnection(answer.rejection)
        return { failure: answer.failure }
      },
    },
  })
  const notify = vi.fn()
  const refresh = vi.fn(() => Promise.resolve())
  const getName = (pluginId: string) => (pluginId === 'md.numen.sample' ? 'Sample' : pluginId)
  return {
    calls,
    notify,
    refresh,
    retry: createPluginRetry({
      askPluginRetry: settings.retryPlugin,
      words,
      notify,
      refresh,
      getName,
    }),
  }
}

describe('createPluginRetry', () => {
  it('sends the plugin id to the core and refreshes the settings', async () => {
    const { calls, notify, refresh, retry } = createRetry({ failure: '' })
    await retry.retryPlugin('md.numen.sample')
    expect(calls).toStrictEqual(['md.numen.sample'])
    expect(notify).toHaveBeenCalledOnce()
    expect(notify).toHaveBeenLastCalledWith('')
    expect(refresh).toHaveBeenCalledOnce()
  })

  it('shows a repeated failure with the plugin name and the reason', async () => {
    const { notify, refresh, retry } = createRetry({ failure: 'Failed to start' })
    await retry.retryPlugin('md.numen.sample')
    expect(notify).toHaveBeenLastCalledWith('Sample stopped: Failed to start', 'warning')
    expect(refresh).toHaveBeenCalledOnce()
  })

  it('shows a failed request as an error with the plugin name, and refreshes the settings', async () => {
    const { notify, refresh, retry } = createRetry({ rejection: 'core unreachable' })
    await retry.retryPlugin('md.numen.sample')
    expect(notify).toHaveBeenLastCalledWith(
      expect.stringContaining('Could not retry Sample: '),
      'error',
    )
    expect(refresh).toHaveBeenCalledOnce()
  })
})
