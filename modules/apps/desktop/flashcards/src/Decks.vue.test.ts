// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { Goal } from '@numen/protocol'

import Decks from './Decks.vue'
import type { Owing } from './core'
import type { Preset, Settings } from './scheduling'

const settings = (said: Partial<Settings> = {}): Settings => ({
  goal: Goal.MINUTES_A_DAY,
  byDate: '',
  minutesADay: 20,
  newADay: 10,
  reviewsADay: 45,
  retention: 0.9,
  lightDays: [],
  evenLoad: true,
  ...said,
})

const preset = (said: Partial<Preset> = {}): Preset => ({
  path: 'Sanskrit.md',
  name: 'Sanskrit',
  settings: settings(),
  decks: ['decks/Words.md'],
  named: 1,
  faces: 20,
  cards: 16,
  budget: { new: 10, reviews: 45, minutes: 20 },
  answered: 0,
  took: 0,
  paused: '',
  ...said,
})

const vault: Owing = {
  vaultId: '01A',
  name: 'Studies',
  path: '/vaults/01A',
  faces: 40,
  due: 12,
  new: 4,
  decks: [
    { deck: 'decks/Words.md', faces: 20, due: 8, new: 2 },
    { deck: 'decks/Roots.md', faces: 20, due: 4, new: 2 },
  ],
  presets: [
    {
      preset: 'Sanskrit.md',
      title: 'Sanskrit',
      decks: 1,
      cards: 20,
      answered: 0,
      took: 0,
      new: 10,
      reviews: 45,
      minutes: 20,
    },
  ],
  unread: '',
}

const shown = (presets: readonly Preset[], over: Owing = vault) =>
  mount(Decks, {
    props: {
      vault: over,
      days: new Map(),
      due: new Map(),
      presets,
      byDeck: new Map(presets.flatMap((one) => one.decks.map((deck) => [deck, one] as const))),
      today: '2026-09-05',
    },
  })

describe('the decks of a vault', () => {
  it('says which preset schedules each deck', () => {
    const one = shown([preset({ decks: ['decks/Words.md', 'decks/Roots.md'] })])

    expect(one.findAll('.decks__by').map((by) => by.text())).toEqual(['Sanskrit', 'Sanskrit'])
  })

  it('shows a deck of a preset that schedules nothing as not studied today', () => {
    const one = shown([
      preset({ decks: ['decks/Words.md'], paused: 'no cards a day', cards: 0 }),
      preset({ path: 'Pali.md', name: 'Pali', decks: ['decks/Roots.md'] }),
    ])
    const rows = one.findAll('.decks__deck')

    expect(one.find('.decks__stopped').text()).toBe('no cards a day')
    expect(rows[0]?.attributes('disabled')).toBeDefined()
    expect(rows[1]?.attributes('disabled')).toBeUndefined()
  })

  it('leaves a deck alone where nothing says which preset schedules it', () => {
    const one = shown([])

    expect(one.findAll('.decks__by')).toHaveLength(0)
    expect(one.findAll('.decks__stopped')).toHaveLength(0)
  })
})

describe('a deck with nothing waiting', () => {
  /** A vault whose one deck holds these cards and owes this much of them. */
  const holding = (faces: number, due: number, fresh: number): Owing => ({
    ...vault,
    due,
    new: fresh,
    decks: [{ deck: 'decks/Words.md', faces, due, new: fresh }],
  })

  it('says the day is done where the deck holds cards and owes none', () => {
    const one = shown([preset({ cards: 0 })], holding(20, 0, 0))

    expect(one.find('.decks__met').text()).toBe('Done today')
    expect(one.findAll('.owed')).toHaveLength(1)
    expect(one.find('.decks__all').text()).toContain('0')
  })

  it('says the reason instead where the preset schedules nothing today', () => {
    const one = shown([preset({ cards: 0, paused: 'no cards a day' })], holding(20, 0, 0))

    expect(one.find('.decks__stopped').text()).toBe('no cards a day')
    expect(one.findAll('.decks__met')).toHaveLength(0)
  })

  it('says neither where the deck holds no card at all', () => {
    const one = shown([preset({ cards: 0 })], holding(0, 0, 0))

    expect(one.findAll('.decks__met')).toHaveLength(0)
    expect(one.findAll('.decks__stopped')).toHaveLength(0)
    // The pill on the button of the whole vault is the only count on the row.
    expect(one.find('.decks__deck').findAll('.owed')).toHaveLength(0)
  })

  it('counts what is waiting where something is', () => {
    const one = shown([preset()], holding(20, 8, 2))

    expect(one.find('.decks__deck').find('.owed').text()).toBe('10 to review')
    expect(one.findAll('.decks__met')).toHaveLength(0)
  })
})
