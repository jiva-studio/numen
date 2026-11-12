import { describe, expect, it } from 'vitest'
import type { PluginHealth } from './pluginInfo'
import { getPluginState, getWorstItem } from './state'

const item = (key: string, severity: PluginHealth['severity']): PluginHealth => ({
  key,
  severity,
  text: `The ${key} does not work`,
  detail: '',
  actionLabel: '',
})

describe('the worst item of a plugin', () => {
  it('is the first error, before any warning', () => {
    const health = [item('quota', 'warning'), item('connection', 'error'), item('model', 'error')]

    expect(getWorstItem(health)?.key).toBe('connection')
  })

  it('is the first warning where there is no error, and none where there is no item', () => {
    expect(getWorstItem([item('quota', 'warning'), item('cache', 'warning')])?.key).toBe('quota')
    expect(getWorstItem([])).toBeUndefined()
  })
})

describe('getPluginState', () => {
  it('is stopped for a plugin whose start failed, whatever its setting and its other items', () => {
    const health = [item('connection', 'error'), item('start', 'error')]

    expect(getPluginState(true, health)).toBe('stopped')
    expect(getPluginState(false, health)).toBe('stopped')
  })

  it('is the severity of the worst item of a plugin that runs', () => {
    expect(getPluginState(true, [item('quota', 'warning'), item('connection', 'error')])).toBe(
      'error',
    )
    expect(getPluginState(true, [item('quota', 'warning')])).toBe('warning')
  })

  it('follows the enabled setting for a plugin without items', () => {
    expect(getPluginState(true)).toBe('enabled')
    expect(getPluginState(false, [])).toBe('disabled')
  })
})
