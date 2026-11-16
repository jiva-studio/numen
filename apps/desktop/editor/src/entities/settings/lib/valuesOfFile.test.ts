import { describe, expect, it } from 'vitest'
import type { PluginInfo, PluginSettingInfo } from './configuration'
import { getValuesOfFile } from './valuesOfFile'

const setting = (key: string, value?: number): PluginSettingInfo => ({
  key,
  type: 'range',
  title: key,
  description: '',
  value,
})

const createPlugin = (settings: readonly PluginSettingInfo[]): PluginInfo => ({
  id: 'p.q',
  name: 'P',
  version: '1.0.0',
  description: '',
  isActive: true,
  capabilities: [],
  hasFrontend: false,
  hasBackend: false,
  settings,
})

describe('the values of the declared settings in a file', () => {
  it('are the values the file holds at the path of each setting', () => {
    const plugin = createPlugin([setting('speed', 1), setting('depth', 2)])
    const taken = getValuesOfFile(
      { app: { appearance: { scale: 1.4 } }, 'p.q': { speed: 3 } },
      [setting('appearance.scale', 1), setting('appearance.mode')],
      [plugin],
    )
    expect(taken.app.map((one) => one.value)).toEqual([1.4, undefined])
    expect(taken.plugins[0]?.settings.map((one) => one.value)).toEqual([3, undefined])
    expect(taken.plugins[0]?.id).toBe('p.q')
  })
})
