// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import Decks from './Decks.vue'
import { getLearnedShare } from '../lib/progress'
import type { BudgetKeys, DeckCardsDue, VaultCardsDue } from '@/entities/vault'
import type { Preset, Settings } from '../types'

const settings: Settings = {
  goal: 'minutes',
  byDate: '',
  minutesADay: 20,
  newADay: 10,
  reviewsADay: 45,
  retention: 0.9,
  load: {},
  evenLoad: true,
}

/** What closes the day of the fixture, which is steered by its minutes. */
const closes: BudgetKeys = { new: '', reviews: '', minutes: 'minutes_a_day' }

const preset = (fields: Partial<Preset> = {}): Preset => ({
  path: 'Sanskrit.md',
  name: 'Sanskrit',
  settings,
  decks: ['decks/Words.md'],
  named: 1,
  faces: 20,
  cards: 16,
  budget: { new: 10, reviews: 45, minutes: 20 },
  closes,
  answered: 0,
  answeredNew: 0,
  answeredReviews: 0,
  took: 0,
  paused: '',
  wrong: '',
  ...fields,
})

const deck = (fields: Partial<DeckCardsDue> = {}): DeckCardsDue => ({
  deck: 'decks/Words.md',
  faces: 20,
  due: 8,
  new: 2,
  learned: 5,
  unbegun: 0,
  ...fields,
})

const vault = (fields: Partial<VaultCardsDue> = {}): VaultCardsDue => ({
  vault: '01A',
  name: 'Studies',
  path: '/vaults/01A',
  isCounted: true,
  faces: 20,
  due: 8,
  new: 2,
  decks: [deck()],
  presets: [],
  unread: '',
  isReading: false,
  ...fields,
})

/** The screen over a vault, and over the presets its decks were read to hold. */
const mountDecks = (over: VaultCardsDue, presets: readonly Preset[], isScheduled = true) =>
  mount(Decks, {
    props: {
      vault: over,
      days: new Map(),
      due: new Map(),
      presets,
      byDeck: new Map(presets.flatMap((one) => one.decks.map((at) => [at, one] as const))),
      hasPresets: isScheduled,
      today: '2026-09-05',
    },
  })

describe('how much of a deck stands learned', () => {
  it('is drawn as a share of the deck, with the word', () => {
    const one = mountDecks(vault(), [preset()])

    expect(one.findAll('.decks__learned').map((at) => at.text())).toEqual(['25% learned'])
  })

  // The tile above draws a bare share of the day's work, so this one carries
  // the word wherever it stands: the two are different questions.
  it('never stands as a figure alone', () => {
    const one = mountDecks(vault(), [preset({ answered: 10, took: 10 })])

    for (const at of one.findAll('.decks__learned')) {
      expect(at.text()).toContain('learned')
    }
  })

  it('says nothing of a deck holding no card face', () => {
    const one = mountDecks(
      vault({ faces: 0, due: 0, new: 0, decks: [deck({ faces: 0, due: 0, new: 0, learned: 0 })] }),
      [preset({ faces: 0, cards: 0 })],
    )

    expect(one.findAll('.decks__learned')).toHaveLength(0)
  })

  // A count that has not landed says a figure is coming and never that there
  // is none, and the row does not move when it arrives.
  it('is drawn as the shape it will be until the count lands', () => {
    const one = mountDecks(vault({ isCounted: false }), [])

    expect(one.findAll('.decks__learned')).toHaveLength(1)
    expect(one.find('.decks__learned').text()).toBe('')
    expect(one.find('.decks__learned').classes()).toContain('skeleton')
  })

  // The rule the share is counted by is the preset's, so a deck whose preset
  // could not be read has none to be counted by.
  it('says there is no rule to count by where the deck has no preset', () => {
    const one = mountDecks(vault(), [preset({ decks: [] })])

    expect(one.find('.decks__learned').text()).toBe('no rule to count by')
  })

  // The figure is the count's and the rule is the preset's, so a screen still
  // reading the presets draws the figure and holds its verdict.
  it('draws the count while the presets are still being read', () => {
    const one = mountDecks(vault(), [], false)

    expect(one.find('.decks__learned').text()).toBe('25% learned')
  })

  it('stands beside what the deck owes and not in its place', () => {
    const one = mountDecks(vault(), [preset()])

    expect(one.find('.decks__learned').exists()).toBe(true)
    expect(one.text()).toContain('10')
  })
})

describe('a deck as a share of itself', () => {
  it('is what stands learned over what the deck holds', () => {
    expect(getLearnedShare(deck({ faces: 20, learned: 5 }))).toBe(0.25)
    expect(getLearnedShare(deck({ faces: 4, learned: 4 }))).toBe(1)
    expect(getLearnedShare(deck({ faces: 4, learned: 0 }))).toBe(0)
  })

  it('is nothing at all for a deck holding no card face', () => {
    expect(getLearnedShare(deck({ faces: 0, learned: 0 }))).toBeNull()
  })
})

// Nothing here reads a preset's own day, and the fixture is a preset that
// schedules something.
it('draws a deck of a stopped preset with its share all the same', () => {
  const one = mountDecks(vault(), [preset({ paused: 'no cards a day' })])

  expect(one.find('.decks__learned').text()).toBe('25% learned')
})
