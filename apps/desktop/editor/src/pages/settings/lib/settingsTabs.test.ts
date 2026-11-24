import { describe, expect, it } from 'vitest'
import { DEFAULT_TAB, getViewTab } from './settingsTabs'
import { getRegisteredViews } from './settingsViews'
import { hasPluginSettings } from './pluginSettings'
import { createPluginInfo } from '../testing/installation'

describe('getViewTab', () => {
  it('puts a view that names no tab in the default tab', () => {
    expect(getViewTab({ id: 'a', title: 'A' })).toBe(DEFAULT_TAB)
    expect(getViewTab({ id: 'a', title: 'A', tab: '' })).toBe(DEFAULT_TAB)
  })
})

describe('hasPluginSettings with views of a contributed tab', () => {
  it('is false where every view sits in a contributed tab', () => {
    const plugin = createPluginInfo('a.b', 'B', {
      settingsViews: [{ id: 'vaults', title: 'Vaults', tab: 'sync' }],
    })

    expect(hasPluginSettings(plugin)).toBe(false)
  })

  it('is true for a view with no tab', () => {
    const plugin = createPluginInfo('a.b', 'B', { settingsViews: [{ id: 'v', title: 'V' }] })

    expect(hasPluginSettings(plugin)).toBe(true)
  })
})

describe('getRegisteredViews of a tab', () => {
  const declared = [
    { id: 'vaults', title: 'Vaults', tab: 'sync' },
    { id: 'plain', title: 'Plain' },
  ]
  const registered = new Map([
    ['vaults', { component: { name: 'Vaults' } }],
    ['plain', { component: { name: 'Plain' } }],
  ])

  it('lists the views of the tab asked for', () => {
    expect(getRegisteredViews(declared, registered, 'sync').map((one) => one.id)).toStrictEqual([
      'vaults',
    ])
  })

  it('lists the views of the default tab when no tab is asked for', () => {
    expect(getRegisteredViews(declared, registered).map((one) => one.id)).toStrictEqual(['plain'])
  })
})
