import { describe, expect, it } from 'vitest'
import type { Theme } from './appearance'
import { getAppearanceGroups, readAppearancePick, type AppearanceChoiceState } from './choices'

const WORDS = {
  apply: 'Apply',
  shipping: 'Built-in',
  owned: 'Custom',
  half: 'Mode',
  current: 'Current',
  pinned: 'The theme sets light or dark itself',
  modes: { system: 'System', light: 'Light', dark: 'Dark' },
}

const NUMEN: Theme = {
  name: 'preset:numen',
  title: 'numen',
  origin: 'preset',
  pluginId: '',
  isPinned: false,
}
const NIGHT: Theme = {
  name: 'plugin:org.example.themes/night',
  title: 'Night',
  origin: 'other',
  pluginId: 'org.example.themes',
  isPinned: true,
}
const DAY: Theme = {
  name: 'plugin:org.example.themes/day',
  title: 'Day',
  origin: 'other',
  pluginId: 'org.example.themes',
  isPinned: false,
}
const SEA: Theme = { name: 'mine:sea', title: 'sea', origin: 'mine', pluginId: '', isPinned: false }

const STATE: AppearanceChoiceState = {
  themes: [NUMEN, NIGHT, DAY, SEA],
  applied: NUMEN.name,
  mode: 'system',
  getPluginName: (id) => (id === 'org.example.themes' ? 'Sample Themes' : id),
}

const getModes = (state: AppearanceChoiceState) =>
  getAppearanceGroups(state, WORDS, '')
    .flatMap((group) => group.items)
    .filter((one) => one.id.startsWith('mode:'))

describe('the themes the palette offers', () => {
  it('stand in the shipped group, one group for each plugin under its name, and the own group', () => {
    const groups = getAppearanceGroups(STATE, WORDS, '')

    expect(groups.map((one) => [one.id, one.title])).toStrictEqual([
      ['shipping', WORDS.shipping],
      ['plugin:org.example.themes', 'Sample Themes'],
      ['owned', WORDS.owned],
      ['half', WORDS.half],
    ])
    expect(groups.map((one) => one.items.map((item) => item.title))).toStrictEqual([
      ['numen'],
      ['Night', 'Day'],
      ['sea'],
      ['System', 'Light', 'Dark'],
    ])
  })

  it('mark the applied one and leave the others unmarked', () => {
    const groups = getAppearanceGroups(STATE, WORDS, '')

    expect(groups[0]?.items[0]?.detail).toBe(WORDS.current)
    expect(groups[0]?.items[0]?.actions).toStrictEqual([{ id: 'apply', text: WORDS.apply }])
    expect(groups[1]?.items[0]?.detail).toBeUndefined()
  })

  it('are narrowed by the text typed, whatever its case or surrounding spaces', () => {
    const titles = getAppearanceGroups(STATE, WORDS, '  NIG  ')
      .flatMap((group) => group.items)
      .map((one) => one.title)

    expect(titles).toStrictEqual(['Night'])
  })
})

describe('the modes the palette offers', () => {
  it('cannot be chosen while the applied theme sets light or dark itself', () => {
    const modes = getModes({ ...STATE, applied: NIGHT.name })

    expect(modes).toHaveLength(3)
    expect(modes.every((one) => one.isDisabled === true && one.detail === WORDS.pinned)).toBe(true)
  })

  it('mark the one in force, and stay enabled when the applied theme is unknown', () => {
    const modes = getModes({ ...STATE, themes: [NIGHT], applied: 'missing', mode: 'light' })

    expect(modes.every((one) => one.isDisabled === undefined)).toBe(true)
    expect(modes.filter((one) => one.detail === WORDS.current).map((one) => one.id)).toStrictEqual([
      'mode:light',
    ])
  })
})

describe('a choice picked', () => {
  it('is a theme by its name, a mode, or nothing', () => {
    expect(readAppearancePick(`theme:${SEA.name}`)).toStrictEqual({
      kind: 'theme',
      theme: SEA.name,
    })
    expect(readAppearancePick('mode:dark')).toStrictEqual({ kind: 'mode', mode: 'dark' })
    expect(readAppearancePick('mode:dusk')).toBeUndefined()
    expect(readAppearancePick('other:thing')).toBeUndefined()
  })
})
