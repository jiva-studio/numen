import { describe, expect, it } from 'vitest'
import {
  getFirstPluginTab,
  getPanelElementId,
  getPluginTabs,
  getTabElementId,
  getTabIndexForKey,
} from './tabs'

describe('getPluginTabs', () => {
  it('lists Settings, Slots, About, Contributes and Data in that order', () => {
    expect(
      getPluginTabs({ hasSettings: true, hasSlots: true, hasData: true }).map((one) => one.text),
    ).toStrictEqual(['Settings', 'Slots', 'About', 'Contributes', 'Data'])
  })

  it('leaves out Data where the plugin has none', () => {
    expect(getPluginTabs({ hasSettings: true, hasData: false }).map((one) => one.id)).toStrictEqual(
      ['settings', 'about', 'contributions'],
    )
  })

  it('leaves out Settings where the plugin has none', () => {
    expect(
      getPluginTabs({ hasSettings: false, hasData: false }).map((one) => one.id),
    ).toStrictEqual(['about', 'contributions'])
  })
})

describe('getFirstPluginTab', () => {
  it('is Settings with settings and About without', () => {
    expect(getFirstPluginTab(true)).toBe('settings')
    expect(getFirstPluginTab(false)).toBe('about')
  })
})

describe('getTabIndexForKey', () => {
  it('steps with the arrow keys and wraps around', () => {
    expect(getTabIndexForKey('ArrowRight', 0, 3)).toBe(1)
    expect(getTabIndexForKey('ArrowRight', 2, 3)).toBe(0)
    expect(getTabIndexForKey('ArrowLeft', 0, 3)).toBe(2)
  })

  it('goes to the first tab on Home and to the last on End', () => {
    expect(getTabIndexForKey('Home', 2, 4)).toBe(0)
    expect(getTabIndexForKey('End', 0, 4)).toBe(3)
  })

  it('answers nothing for another key', () => {
    expect(getTabIndexForKey('Enter', 1, 3)).toBeUndefined()
    expect(getTabIndexForKey('ArrowDown', 1, 3)).toBeUndefined()
  })
})

describe('the ids of a tab and its panel', () => {
  it('differ, and both carry the base and the tab', () => {
    expect(getTabElementId('plugin-a', 'about')).toBe('plugin-a-tab-about')
    expect(getPanelElementId('plugin-a', 'about')).toBe('plugin-a-panel-about')
  })
})

describe('the Slots tab', () => {
  it('is there only where the plugin has slot entries, after Settings', () => {
    expect(
      getPluginTabs({ hasSettings: true, hasSlots: true, hasData: false }).map((one) => one.id),
    ).toStrictEqual(['settings', 'slots', 'about', 'contributions'])
    expect(getPluginTabs({ hasSettings: true, hasData: false }).map((one) => one.id)).not.toContain(
      'slots',
    )
  })
})

describe('the tabs a plugin contributes', () => {
  const contributed = [{ id: 'sync', title: 'Sync' }]

  it('follow Settings and come before Slots, About, Contributes and Data', () => {
    expect(
      getPluginTabs({ hasSettings: true, hasSlots: true, hasData: true, contributed }).map(
        (one) => one.text,
      ),
    ).toStrictEqual(['Settings', 'Sync', 'Slots', 'About', 'Contributes', 'Data'])
  })

  it('come first where the plugin has no Settings tab', () => {
    expect(
      getPluginTabs({ hasSettings: false, hasData: false, contributed }).map((one) => one.id),
    ).toStrictEqual(['sync', 'about', 'contributions'])
  })

  it('leave the tabs of a plugin that contributes none as they were', () => {
    expect(
      getPluginTabs({ hasSettings: true, hasData: false, contributed: [] }).map((one) => one.id),
    ).toStrictEqual(['settings', 'about', 'contributions'])
  })

  it('get element ids from their own id', () => {
    expect(getTabElementId('plugin-a', 'sync')).toBe('plugin-a-tab-sync')
    expect(getPanelElementId('plugin-a', 'sync')).toBe('plugin-a-panel-sync')
  })
})
