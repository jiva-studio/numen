import { describe, expect, it } from 'vitest'
import { getRegisteredViews, getSettingValues } from './settingsViews'
import { hasPluginSettings } from './pluginSettings'
import { createPluginInfo } from '@/pages/settings/testing/installation'

describe('getRegisteredViews', () => {
  const declared = [
    { id: 'top', title: 'Top' },
    { id: 'floor', title: 'Floor' },
    { id: 'missing', title: 'Missing' },
  ]
  const T = { name: 'T' }
  const F = { name: 'F' }
  const registered = new Map([
    ['top', { component: T }],
    ['floor', { component: F }],
  ])

  it('returns the registered views in declared order', () => {
    expect(getRegisteredViews(declared, registered)).toStrictEqual([
      { id: 'top', component: T },
      { id: 'floor', component: F },
    ])
  })

  it('returns nothing while no view is registered', () => {
    expect(getRegisteredViews(declared, new Map())).toStrictEqual([])
  })
})

describe('getSettingValues', () => {
  it('reads every declared setting by key', () => {
    const plugin = createPluginInfo('p', 'P', {
      settings: [
        { key: 'a', type: 'number', title: 'A', description: '', default: 1, palette: false },
        { key: 'b', type: 'string', title: 'B', description: '', default: '', palette: false },
      ],
    })
    expect(getSettingValues(plugin, (key) => (key === 'a' ? 5 : undefined))).toStrictEqual({
      a: 5,
      b: '',
    })
  })
})

describe('hasPluginSettings', () => {
  it('detects settings, settingsViews, and visible secrets', () => {
    const empty = createPluginInfo('p', 'P')
    expect(hasPluginSettings(empty, () => undefined)).toBe(false)

    const withSetting = createPluginInfo('p', 'P', {
      settings: [{ key: 'a', type: 'string', title: 'A', description: '', default: '' }],
    })
    expect(hasPluginSettings(withSetting, () => undefined)).toBe(true)

    const withView = createPluginInfo('p', 'P', {
      settingsViews: [{ id: 'v', title: 'V' }],
    })
    expect(hasPluginSettings(withView, () => undefined)).toBe(true)

    const withHiddenSecretOnly = createPluginInfo('p', 'P', {
      settings: [],
      secrets: [
        {
          name: 's1',
          title: 'S1',
          description: '',
          group: '',
          when: { setting: 'mode', is: ['on'] },
        },
      ],
    })
    expect(hasPluginSettings(withHiddenSecretOnly, () => 'off')).toBe(false)

    const withNoSettingsSecretsOnly = createPluginInfo('p', 'P', {
      settings: [],
      secrets: [{ name: 's1', title: 'S1', description: '', group: '' }],
    })
    expect(hasPluginSettings(withNoSettingsSecretsOnly, () => undefined)).toBe(true)
  })
})
