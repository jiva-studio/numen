import { describe, expect, it } from 'vitest'
import { DEFAULT_TAB, getViewTab, listSettingsTabs } from './settingsTabs'
import { getRegisteredViews } from './settingsViews'

describe('listSettingsTabs', () => {
  it('lists only the default tab for a plugin that declares none', () => {
    expect(listSettingsTabs({}, 'Settings')).toStrictEqual([{ id: DEFAULT_TAB, text: 'Settings' }])
  })

  it('puts the default tab before the declared ones', () => {
    const tabs = listSettingsTabs({ settingsTabs: [{ id: 'sync', title: 'Sync' }] }, 'Settings')

    expect(tabs.map((one) => one.id)).toStrictEqual(['settings', 'sync'])
  })
})

describe('getViewTab', () => {
  it('puts a view that names no tab in the default tab', () => {
    expect(getViewTab({ id: 'a', title: 'A' })).toBe(DEFAULT_TAB)
    expect(getViewTab({ id: 'a', title: 'A', tab: '' })).toBe(DEFAULT_TAB)
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
