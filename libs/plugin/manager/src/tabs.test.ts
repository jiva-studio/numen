import { describe, expect, it } from 'vitest'
import { getPanelElementId, getPluginTabs, getTabElementId, getTabIndexForKey } from './tabs'

describe('getPluginTabs', () => {
  it('lists Settings, About, Contributes and Data in that order', () => {
    expect(getPluginTabs(true).map((one) => one.text)).toStrictEqual([
      'Settings',
      'About',
      'Contributes',
      'Data',
    ])
  })

  it('leaves out Data where the plugin has none', () => {
    expect(getPluginTabs(false).map((one) => one.id)).toStrictEqual([
      'settings',
      'about',
      'contributions',
    ])
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
