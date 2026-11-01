import { describe, expect, it } from 'vitest'
import type { Ranges, Sizes, Theme } from '@/entities/settings'
import { WORDS as words } from '@/shared/words'
import { INTERFACE_SCALE, TEXT_SCALE } from './appearanceValues'
import { getModeGroups, getSizeGroups, getThemeGroups } from './offers'

const THEMES: readonly Theme[] = [
  { name: 'preset:numen', title: 'numen', isBuiltIn: true, pluginId: '', isPinned: false },
  { name: 'preset:dracula', title: 'dracula', isBuiltIn: true, pluginId: '', isPinned: true },
  { name: 'mine:sea', title: 'sea', isBuiltIn: false, pluginId: '', isPinned: false },
]

const PLUGIN_THEMES: readonly Theme[] = [
  ...THEMES,
  {
    name: 'plugin:org.example.themes/night',
    title: 'Night',
    isBuiltIn: false,
    pluginId: 'org.example.themes',
    isPinned: true,
  },
  {
    name: 'plugin:org.example.themes/day',
    title: 'Day',
    isBuiltIn: false,
    pluginId: 'org.example.themes',
    isPinned: false,
  },
]

const getPluginName = (pluginId: string) =>
  pluginId === 'org.example.themes' ? 'Sample Themes' : pluginId

const SIZES: Sizes = { interfaceScale: 1, textScale: 1 }
const BOUNDS: Ranges = {
  interfaceScale: { min: 0.8, max: 2 },
  textScale: { min: 0.8, max: 1.75 },
}

describe('the themes offered', () => {
  it('remains the origin of the theme in force first', () => {
    expect(getThemeGroups(THEMES, 'mine:sea', words).map((one) => one.id)).toStrictEqual([
      'owned',
      'shipping',
    ])
    expect(getThemeGroups(THEMES, 'preset:numen', words).map((one) => one.id)).toStrictEqual([
      'shipping',
      'owned',
    ])
  })

  it('opens the origin of the theme in force', () => {
    const shipping = getThemeGroups(THEMES, 'preset:dracula', words)[0]

    expect(shipping?.items[0]?.title).toBe('dracula')
    expect(shipping?.items[0]?.isCurrent).toBe(true)
  })

  it('marks an owned theme as current when in force', () => {
    const owned = getThemeGroups(THEMES, 'mine:sea', words)[0]

    expect(owned?.items[0]?.title).toBe('sea')
    expect(owned?.items[0]?.isCurrent).toBe(true)
    expect(owned?.items[0]?.detail).toBe(words.current)
  })
})

describe('the themes a plugin contributes', () => {
  it('stand in one group titled with the name of the plugin, between the shipped and the own', () => {
    const groups = getThemeGroups(PLUGIN_THEMES, 'preset:numen', words, getPluginName)

    expect(groups.map((one) => one.id)).toStrictEqual([
      'shipping',
      'plugin:org.example.themes',
      'owned',
    ])
    expect(groups[1]?.title).toBe('Sample Themes')
    expect(groups[1]?.items.map((one) => one.title)).toStrictEqual(['Night', 'Day'])
  })

  it('open first when the theme in force is theirs, on the theme in force', () => {
    const groups = getThemeGroups(
      PLUGIN_THEMES,
      'plugin:org.example.themes/day',
      words,
      getPluginName,
    )

    expect(groups[0]?.id).toBe('plugin:org.example.themes')
    expect(groups[0]?.items[0]?.title).toBe('Day')
    expect(groups[0]?.items[0]?.isCurrent).toBe(true)
  })

  it('are titled with the id of the plugin where no name is known', () => {
    const groups = getThemeGroups(PLUGIN_THEMES, 'preset:numen', words)

    expect(groups[1]?.title).toBe('org.example.themes')
  })

  it('are left out of the owned group', () => {
    const owned = getThemeGroups(PLUGIN_THEMES, 'preset:numen', words).find(
      (one) => one.id === 'owned',
    )

    expect(owned?.items.map((one) => one.title)).toStrictEqual(['sea'])
  })
})

describe('the modes offered', () => {
  it('marks the one in force and leaves the rest unsaid', () => {
    const items = getModeGroups('dark', false, words).flatMap((one) => one.items)

    expect(items.filter((one) => one.isCurrent).map((one) => one.title)).toStrictEqual([words.dark])
    expect(items.some((one) => one.isDisabled)).toBe(false)
  })

  it('says why none can be chosen where the applied theme pins the two halves', () => {
    const items = getModeGroups('dark', true, words).flatMap((one) => one.items)

    expect(items.every((one) => one.isDisabled)).toBe(true)
    expect(items.every((one) => one.detail === words.pinned)).toBe(true)
  })
})

describe('the sizes offered', () => {
  const titles = (command: string, text = '') =>
    getSizeGroups(command, text, { sizes: SIZES, bounds: BOUNDS, words })
      .flatMap((one) => one.items)
      .map((one) => one.title)

  it('goes as far as the range of that command reaches', () => {
    expect(titles(INTERFACE_SCALE).at(-1)).toBe('200%')
    expect(titles(TEXT_SCALE).at(-1)).toBe('175%')
  })

  it('remains a number a person typed in its place, once', () => {
    expect(titles(INTERFACE_SCALE, '137')).toContain('137%')
    expect(titles(INTERFACE_SCALE, '137').indexOf('137%')).toBe(6)
    expect(titles(INTERFACE_SCALE, '150').filter((one) => one === '150%')).toHaveLength(1)
  })

  it('leaves out a number the range does not reach', () => {
    expect(titles(INTERFACE_SCALE, '250')).not.toContain('250%')
  })
})
