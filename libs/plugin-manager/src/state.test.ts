import { describe, expect, it } from 'vitest'
import { createMockPluginInfo as createPluginInfo } from '@numen/plugin-sdk/testing'
import { formatFailedAt, getFailedPlugins, getPluginState } from './state'

describe('the plugins suspended', () => {
  it('are those with a failure, in the order listed', () => {
    const failure = { reason: 'Timed out while starting', detail: 'handshake timeout', failedAt: 0 }
    const items = [
      createPluginInfo('sample.b', 'B', { failure }),
      createPluginInfo('sample.ok', 'Ok'),
      createPluginInfo('sample.a', 'A', { failure }),
    ]
    expect(getFailedPlugins(items).map((one) => one.id)).toStrictEqual(['sample.b', 'sample.a'])
  })

  it('are none where every plugin starts', () => {
    expect(getFailedPlugins([createPluginInfo('sample.ok', 'Ok')])).toStrictEqual([])
  })
})

describe('the time a plugin was given up', () => {
  it('is failure as a date and a time', () => {
    const failure = formatFailedAt(Date.UTC(2026, 9, 19, 12, 30), 'en-US')
    expect(failure).toMatch(/2026/)
    expect(failure).toMatch(/\d{1,2}:\d{2}/)
  })
})

describe('getPluginState', () => {
  const failure = { reason: 'Failed to start', detail: 'exec format error', failedAt: 0 }

  it('is stopped for a failed plugin, whatever its setting', () => {
    expect(getPluginState(createPluginInfo('sample.a', 'A', { failure }), true)).toBe('stopped')
    expect(getPluginState(createPluginInfo('sample.a', 'A', { failure }), false)).toBe('stopped')
  })

  it('follows the enabled setting for a plugin that has not failed', () => {
    expect(getPluginState(createPluginInfo('sample.a', 'A'), true)).toBe('enabled')
    expect(getPluginState(createPluginInfo('sample.a', 'A'), false)).toBe('disabled')
  })
})
