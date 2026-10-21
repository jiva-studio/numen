import { describe, expect, it, vi } from 'vitest'
import { createPluginRetry, type RetryDeps } from './retry'

const words = { failedAgain: 'It stopped again.', failed: 'It could not be calls.' }

const createRetry = (answer: Promise<string>) => {
  const calls: string[] = []
  const core: RetryDeps = {
    retryPlugin: (pluginId) => {
      calls.push(pluginId)
      return answer
    },
  }
  const notified = vi.fn()
  const refresh = vi.fn(() => Promise.resolve())
  return {
    calls,
    notified,
    refresh,
    retry: createPluginRetry(core, { words, notify: notified, refresh }),
  }
}

describe('a plugin retried', () => {
  it('is handed to the vault by its id, and the settings are read again', async () => {
    const { calls, notified, refresh, retry } = createRetry(Promise.resolve(''))
    await retry.retryPlugin('md.numen.sample')
    expect(calls).toStrictEqual(['md.numen.sample'])
    expect(notified).toHaveBeenCalledOnce()
    expect(notified).toHaveBeenLastCalledWith('')
    expect(refresh).toHaveBeenCalledOnce()
  })

  it('that fails again is notified with its reason', async () => {
    const { notified, refresh, retry } = createRetry(
      Promise.resolve('The plugin did not finish starting in time.'),
    )
    await retry.retryPlugin('md.numen.sample')
    expect(notified).toHaveBeenLastCalledWith(
      'It stopped again. The plugin did not finish starting in time.',
      'caution',
    )
    expect(refresh).toHaveBeenCalledOnce()
  })

  it('that cannot be calls is notified as an error, and the settings are read again', async () => {
    const { notified, refresh, retry } = createRetry(Promise.reject(new Error('core unreachable')))
    await retry.retryPlugin('md.numen.sample')
    expect(notified).toHaveBeenLastCalledWith(
      expect.stringContaining('It could not be calls.'),
      'error',
    )
    expect(refresh).toHaveBeenCalledOnce()
  })
})
