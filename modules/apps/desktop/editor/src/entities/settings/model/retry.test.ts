import { describe, expect, it, vi } from 'vitest'
import { createPluginRetry, type RetryDeps } from './retry'

const words = { unstarted: 'It stopped again.', unasked: 'It could not be asked.' }

const createRetry = (answer: Promise<string>) => {
  const asked: string[] = []
  const core: RetryDeps = {
    retryPlugin: (pluginId) => {
      asked.push(pluginId)
      return answer
    },
  }
  const said = vi.fn()
  const refresh = vi.fn(() => Promise.resolve())
  return { asked, said, refresh, retry: createPluginRetry(core, { words, say: said, refresh }) }
}

describe('a plugin retried', () => {
  it('is handed to the vault by its id, and the settings are read again', async () => {
    const { asked, said, refresh, retry } = createRetry(Promise.resolve(''))
    await retry.retryPlugin('md.numen.sample')
    expect(asked).toStrictEqual(['md.numen.sample'])
    expect(said).toHaveBeenCalledOnce()
    expect(said).toHaveBeenLastCalledWith('')
    expect(refresh).toHaveBeenCalledOnce()
  })

  it('that fails again is said with its reason', async () => {
    const { said, refresh, retry } = createRetry(
      Promise.resolve('The plugin did not finish starting in time.'),
    )
    await retry.retryPlugin('md.numen.sample')
    expect(said).toHaveBeenLastCalledWith(
      'It stopped again. The plugin did not finish starting in time.',
      'caution',
    )
    expect(refresh).toHaveBeenCalledOnce()
  })

  it('that cannot be asked is said as an error, and the settings are read again', async () => {
    const { said, refresh, retry } = createRetry(Promise.reject(new Error('core unreachable')))
    await retry.retryPlugin('md.numen.sample')
    expect(said).toHaveBeenLastCalledWith(
      expect.stringContaining('It could not be asked.'),
      'error',
    )
    expect(refresh).toHaveBeenCalledOnce()
  })
})
