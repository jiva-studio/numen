import { describe, expect, it, vi } from 'vitest'
import { createPluginRetry, type RetryDeps } from './retry'

const words = {
  stopped: (name: string, reason: string) => `${name} stopped: ${reason}`,
  retryFailed: (name: string, message: string) => `Could not retry ${name}: ${message}`,
}

const createRetry = (answer: Promise<string>) => {
  const calls: string[] = []
  const core: RetryDeps = {
    retryPlugin: (pluginId) => {
      calls.push(pluginId)
      return answer
    },
  }
  const notify = vi.fn()
  const refresh = vi.fn(() => Promise.resolve())
  const getName = (pluginId: string) => (pluginId === 'md.numen.sample' ? 'Sample' : pluginId)
  return {
    calls,
    notify,
    refresh,
    retry: createPluginRetry(core, { words, notify, refresh, getName }),
  }
}

describe('createPluginRetry', () => {
  it('sends the plugin id to the core and refreshes the settings', async () => {
    const { calls, notify, refresh, retry } = createRetry(Promise.resolve(''))
    await retry.retryPlugin('md.numen.sample')
    expect(calls).toStrictEqual(['md.numen.sample'])
    expect(notify).toHaveBeenCalledOnce()
    expect(notify).toHaveBeenLastCalledWith('')
    expect(refresh).toHaveBeenCalledOnce()
  })

  it('shows a repeated failure with the plugin name and the reason', async () => {
    const { notify, refresh, retry } = createRetry(Promise.resolve('Failed to start'))
    await retry.retryPlugin('md.numen.sample')
    expect(notify).toHaveBeenLastCalledWith('Sample stopped: Failed to start', 'warning')
    expect(refresh).toHaveBeenCalledOnce()
  })

  it('shows a failed request as an error with the plugin name, and refreshes the settings', async () => {
    const { notify, refresh, retry } = createRetry(Promise.reject(new Error('core unreachable')))
    await retry.retryPlugin('md.numen.sample')
    expect(notify).toHaveBeenLastCalledWith(
      expect.stringContaining('Could not retry Sample: '),
      'error',
    )
    expect(refresh).toHaveBeenCalledOnce()
  })
})
