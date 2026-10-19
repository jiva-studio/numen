import { describe, expect, it } from 'vitest'
import { createPluginInfo } from '@/testing/installation'
import { formatFailedAt, getFailedPlugins } from './failures'

describe('the plugins held off', () => {
  it('are those with a failure, in the order listed', () => {
    const failure = { reason: 'The plugin did not finish starting in time.', failedAt: 0 }
    const listed = [
      createPluginInfo('sample.b', 'B', { failure }),
      createPluginInfo('sample.ok', 'Ok'),
      createPluginInfo('sample.a', 'A', { failure }),
    ]
    expect(getFailedPlugins(listed).map((one) => one.id)).toStrictEqual(['sample.b', 'sample.a'])
  })

  it('are none where every plugin starts', () => {
    expect(getFailedPlugins([createPluginInfo('sample.ok', 'Ok')])).toStrictEqual([])
  })
})

describe('the time a plugin was given up', () => {
  it('is said as a date and a time', () => {
    const said = formatFailedAt(Date.UTC(2026, 9, 19, 12, 30), 'en-US')
    expect(said).toMatch(/2026/)
    expect(said).toMatch(/\d{1,2}:\d{2}/)
  })
})
